package devagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Keeper keeps the machine awake while a request is worked. The sprite
// pauses when no one is connected, and a request's Claude Code, suite and
// deploy freeze with it; a hold stops that until it is released or runs
// out.
type Keeper interface {
	Hold(name string, d time.Duration) error
	Renew(name string, d time.Duration) error
	Release(name string) error
}

// SpriteTasks holds the sprite awake with the runtime's tasks: a named
// task keeps it from pausing until it is deleted or expires. An hour is
// the most one task may ask for; a crashed agent's task expires on its
// own and the sprite sleeps again.
type SpriteTasks struct {
	// Socket is the runtime's API, /.sprite/api.sock on a sprite.
	Socket string
}

func (k SpriteTasks) Hold(name string, d time.Duration) error {
	return k.call("POST", "/v1/tasks", task{Name: name, Expire: int(d.Seconds())})
}

func (k SpriteTasks) Renew(name string, d time.Duration) error {
	return k.call("PUT", "/v1/tasks/"+name, task{Expire: int(d.Seconds())})
}

func (k SpriteTasks) Release(name string) error {
	return k.call("DELETE", "/v1/tasks/"+name, nil)
}

// A task as the API takes it: its expiry in seconds.
type task struct {
	Name   string `json:"name,omitempty"`
	Expire int    `json:"expire"`
}

func (k SpriteTasks) call(method, path string, body any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "http://sprite"+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c := http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", k.Socket)
		}},
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// How long a hold asks for, and how often it is renewed: renewed well
// before it runs out, and short enough that a crashed agent's hold lets
// the sprite sleep soon after.
const (
	holdFor    = 15 * time.Minute
	renewEvery = 5 * time.Minute
)

// keepAwake holds the machine awake for one request and returns what
// lets it go. A hold that cannot be made is logged, not fatal: the work
// still runs, only slower if the sprite pauses.
func (a *Agent) keepAwake(id string) (release func()) {
	if a.Awake == nil {
		return func() {}
	}
	name := "request-" + id
	if err := a.Awake.Hold(name, holdFor); err != nil {
		log.Printf("keep awake: %v", err)
	}
	every := a.RenewEvery
	if every <= 0 {
		every = renewEvery
	}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if err := a.Awake.Renew(name, holdFor); err != nil {
					log.Printf("keep awake: %v", err)
				}
			}
		}
	}()
	return func() {
		close(stop)
		<-done
		if err := a.Awake.Release(name); err != nil {
			log.Printf("keep awake: %v", err)
		}
	}
}
