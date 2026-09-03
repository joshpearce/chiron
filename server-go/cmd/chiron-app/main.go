// Command chiron-app is the sprite agent's hands on the iPad app: each verb
// goes to the book server, which relays it to the app the reader is holding
// and prints what came back.
//
//	chiron-app state
//	chiron-app open subject=ai
//	chiron-app answer mode=mixed
//	chiron-app shot page.png
//	chiron-app status
//
// The server and key come from the same settings as chiron-dev
// (~/.config/chiron-dev/config or CHIRON_URL / CHIRON_KEY); on the sprite the
// url is the book server itself, http://127.0.0.1:8081.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mjbraun/chiron/server/devconfig"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: chiron-app <verb> [name=value ...] | chiron-app shot <file.png> | chiron-app status")
		os.Exit(2)
	}
	cfg, err := devconfig.Load()
	if err != nil {
		fail(err)
	}
	verb := os.Args[1]
	switch verb {
	case "status":
		body, err := call(cfg, "GET", "/agent/status", nil)
		if err != nil {
			fail(err)
		}
		fmt.Println(strings.TrimSpace(string(body)))
	case "shot":
		if len(os.Args) < 3 {
			fail(fmt.Errorf("shot needs a file name"))
		}
		res, err := command(cfg, "screenshot", nil, 60)
		if err != nil {
			fail(err)
		}
		var shot struct {
			PNG string `json:"png_b64"`
		}
		if err := json.Unmarshal(res, &shot); err != nil || shot.PNG == "" {
			fail(fmt.Errorf("no screenshot in the reply: %s", res))
		}
		png, err := base64.StdEncoding.DecodeString(shot.PNG)
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(os.Args[2], png, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("%s (%d bytes)\n", os.Args[2], len(png))
	default:
		args := map[string]any{}
		for _, kv := range os.Args[2:] {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				fail(fmt.Errorf("argument %q: want name=value", kv))
			}
			if n, err := strconv.Atoi(v); err == nil {
				args[k] = n
			} else {
				args[k] = v
			}
		}
		res, err := command(cfg, verb, args, 600)
		if err != nil {
			fail(err)
		}
		var pretty bytes.Buffer
		if json.Indent(&pretty, res, "", "  ") == nil {
			fmt.Println(pretty.String())
		} else {
			fmt.Println(string(res))
		}
	}
}

func command(cfg devconfig.Config, verb string, args map[string]any, timeout float64) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]any{"verb": verb, "args": args, "timeout": timeout})
	return call(cfg, "POST", "/agent/cmd", body)
}

func call(cfg devconfig.Config, method, path string, body []byte) (json.RawMessage, error) {
	req, err := http.NewRequest(method, strings.TrimRight(cfg.URL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 620 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "chiron-app: %v\n", err)
	os.Exit(1)
}
