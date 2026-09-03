package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Settings come from the environment first (CHIRON_URL, CHIRON_KEY), then
// from ~/.config/chiron-dev/config, "name = value" per line. The key may be
// a 1Password reference (op://...), resolved with `op read` at run time so
// the secret never sits in a file.
type config struct {
	URL       string
	Key       string
	OpAccount string
}

func parseConfig(text string) (config, error) {
	var c config
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return c, fmt.Errorf("config line %q: want name = value", line)
		}
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		switch name {
		case "url":
			c.URL = value
		case "key":
			c.Key = value
		case "op_account":
			c.OpAccount = value
		default:
			return c, fmt.Errorf("config: unknown setting %q", name)
		}
	}
	return c, sc.Err()
}

func loadConfig() (config, error) {
	var c config
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "chiron-dev", "config")
	if text, err := os.ReadFile(path); err == nil {
		if c, err = parseConfig(string(text)); err != nil {
			return c, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return c, err
	}
	if v := os.Getenv("CHIRON_URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv("CHIRON_KEY"); v != "" {
		c.Key = v
	}
	if c.URL == "" {
		return c, fmt.Errorf("no server: set url in %s or CHIRON_URL", path)
	}
	if c.Key == "" {
		return c, fmt.Errorf("no key: set key in %s or CHIRON_KEY", path)
	}
	if strings.HasPrefix(c.Key, "op://") {
		args := []string{"read", c.Key}
		if c.OpAccount != "" {
			args = append(args, "--account", c.OpAccount)
		}
		out, err := exec.Command("op", args...).Output()
		if err != nil {
			return c, fmt.Errorf("op read %s: %w", c.Key, err)
		}
		c.Key = strings.TrimSpace(string(out))
	}
	return c, nil
}
