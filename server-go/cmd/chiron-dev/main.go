// Command chiron-dev reaches the Chiron sprite from the Mac without the
// sprite CLI: `chiron-dev proxy` is an ssh ProxyCommand that opens the
// gate's /ssh tunnel (which also wakes the sprite) and pipes ssh through it.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mjbraun/chiron/server/devconfig"
	"github.com/mjbraun/chiron/server/gate/client"
)

const usage = `usage:
  chiron-dev proxy        ssh ProxyCommand: stdin/stdout <-> the sprite's sshd
  chiron-dev ssh-config   print the ~/.ssh/config stanza that uses it

settings: ~/.config/chiron-dev/config (url = ..., key = ... or op://...,
op_account = ...), overridden by CHIRON_URL and CHIRON_KEY.
`

const sshConfig = `Host chiron
  ProxyCommand chiron-dev proxy
  User sprite
  IdentityFile ~/.ssh/chiron_ed25519
  IdentitiesOnly yes
  ServerAliveInterval 15
  ServerAliveCountMax 4
`

// How long to keep knocking while a hibernated sprite wakes. Measured wake
// is about a second; a restore from a checkpoint can be much longer.
const patience = 30 * time.Second

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "proxy":
		if err := proxy(); err != nil {
			fmt.Fprintf(os.Stderr, "chiron-dev: %v\n", err)
			os.Exit(1)
		}
	case "ssh-config":
		fmt.Print(sshConfig)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func proxy() error {
	cfg, err := devconfig.Load()
	if err != nil {
		return err
	}
	// ssh shows the ProxyCommand's stderr, so the wait is visible.
	conn, err := client.Dial(context.Background(), cfg.URL, cfg.Key, patience, func(s string) {
		fmt.Fprintf(os.Stderr, "chiron-dev: waking the sprite (%s)\n", s)
	})
	if err != nil {
		return err
	}
	client.Pipe(conn, os.Stdin, os.Stdout)
	return nil
}
