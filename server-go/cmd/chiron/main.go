// Command chiron drives the book server from a shell: the shelf, captures
// at every scale, a draft's planning conversation, builds, and the text of
// what was written. It is how an agent on the Mac sends work to Chiron.
//
// The server and its key come from ~/.config/chiron-dev/config (url, key,
// op_account), or CHIRON_URL and CHIRON_KEY, the same as chiron-dev.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mjbraun/chiron/server/devconfig"
)

const usage = `usage:
  chiron shelf                         every subject on the shelf and where it stands
  chiron capture -scale S -prompt Q [-title T] [-url U] [-app A] [-text T | -file F]
                                       send a passage in; S is summary, detail,
                                       primer or book; the passage is stdin when neither
                                       -text nor -file is given
  chiron plan ID [-say TEXT]           a draft's planning conversation; -say answers
  chiron build ID [-brief B | -brief-file F]
                                       write the primer or start the book, with the
                                       brief the plan settled on or the one given here
  chiron discard ID                    drop a draft
  chiron status ID                     a draft's plan, a book job, or the shelf row
  chiron wait ID [-timeout D]          until ID is ready or failed; prints the row
  chiron read ID [-unit U]             the text of a primer, or of a book's unit
  chiron page URL                      read a page on the web here: its article goes
                                       on the shelf, to highlight and ask about
  chiron follow URL                    follow a blog by its Atom or RSS feed; its
                                       posts become the chapters of a shelf card
  chiron feeds [-check]                the blogs followed; -check asks them what is new
  chiron teach -title T (-brief B | -brief-file F) [-slug S] [-source NAME ...]
                                       a book from a brief; -source names the open
                                       text to start from (first) and to interleave

Replies are JSON except shelf (a table) and read (text). Exit 1 on failure.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := devconfig.Load()
	if err != nil {
		fail(err)
	}
	c := &client{url: strings.TrimRight(cfg.URL, "/"), key: cfg.Key, http: &http.Client{Timeout: 10 * time.Minute}}
	out, err := run(c, os.Args[1], os.Args[2:], os.Stdin)
	if out != "" {
		fmt.Print(out)
		if !strings.HasSuffix(out, "\n") {
			fmt.Println()
		}
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "chiron: %v\n", err)
	if errors.Is(err, errUsage) {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	os.Exit(1)
}

var errUsage = errors.New("usage")

// run is the whole command surface; main only adds the config and the exit
// code, so tests drive it against a server of their own.
func run(c *client, cmd string, args []string, stdin io.Reader) (string, error) {
	switch cmd {
	case "shelf":
		return c.shelf()
	case "capture":
		return c.capture(args, stdin)
	case "plan":
		return c.plan(args)
	case "build":
		return c.build(args)
	case "discard":
		return c.discard(args)
	case "status":
		return c.status(args)
	case "wait":
		return c.wait(args)
	case "read":
		return c.read(args)
	case "page":
		return c.page(args)
	case "follow":
		return c.follow(args)
	case "feeds":
		return c.feeds(args)
	case "teach":
		return c.teach(args)
	default:
		return "", fmt.Errorf("%w: unknown command %q", errUsage, cmd)
	}
}

type client struct {
	url  string
	key  string
	http *http.Client
	// patience is how long to keep knocking while a hibernated sprite wakes.
	patience time.Duration
}

type apiError struct {
	status int
	detail string
}

func (e *apiError) Error() string { return fmt.Sprintf("%d: %s", e.status, e.detail) }

// call sends one request and decodes the JSON reply into out. A sprite
// that is waking answers with a gateway error or no connection at all;
// those are retried for patience.
func (c *client) call(method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	patience := c.patience
	if patience == 0 {
		patience = 90 * time.Second
	}
	deadline := time.Now().Add(patience)
	for {
		req, err := http.NewRequest(method, c.url+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.http.Do(req)
		if err != nil {
			var netErr net.Error
			if (errors.As(err, &netErr) || strings.Contains(err.Error(), "connection refused")) && time.Now().Before(deadline) {
				fmt.Fprintf(os.Stderr, "chiron: waiting for the server (%v)\n", err)
				time.Sleep(3 * time.Second)
				continue
			}
			return err
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout {
			if time.Now().Before(deadline) {
				fmt.Fprintf(os.Stderr, "chiron: waiting for the server (%d)\n", resp.StatusCode)
				time.Sleep(3 * time.Second)
				continue
			}
		}
		if resp.StatusCode/100 != 2 {
			var e struct {
				Detail string `json:"detail"`
			}
			json.Unmarshal(data, &e)
			if e.Detail == "" {
				e.Detail = strings.TrimSpace(string(data))
			}
			return &apiError{status: resp.StatusCode, detail: e.Detail}
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(data, out)
	}
}

func pretty(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b) + "\n"
}

// A row of the shelf, as GET /subjects lists it.
type row struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Kind         string `json:"kind"`
	UnitsTotal   int    `json:"units_total"`
	UnitsCleared int    `json:"units_cleared"`
	Status       string `json:"status,omitempty"`
	Error        string `json:"error,omitempty"`
	Scale        string `json:"scale,omitempty"`
	Book         string `json:"book,omitempty"`
	Progress     string `json:"progress,omitempty"`
}

type shelfReply struct {
	Subjects []row  `json:"subjects"`
	Active   string `json:"active"`
}

func (c *client) rows() (shelfReply, error) {
	var s shelfReply
	err := c.call("GET", "/subjects", nil, &s)
	return s, err
}

// where says, in a few words, where a row stands.
func where(r row, active string) string {
	switch {
	case r.Status == "failed":
		return "failed: " + r.Error
	case r.Status != "" && r.Status != "ready":
		if r.Progress != "" {
			return r.Status + ", " + r.Progress
		}
		return r.Status
	case r.Kind == "primer":
		return "ready"
	case r.ID == active:
		return fmt.Sprintf("%d/%d units, open now", r.UnitsCleared, r.UnitsTotal)
	default:
		return fmt.Sprintf("%d/%d units", r.UnitsCleared, r.UnitsTotal)
	}
}

func (c *client) shelf() (string, error) {
	s, err := c.rows()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, r := range s.Subjects {
		fmt.Fprintf(&b, "%-44s %-8s %-28s %s\n", r.ID, r.Kind, where(r, s.Active), r.Title)
	}
	return b.String(), nil
}

func flags(name string, args []string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// textOf is the passage: -text, the contents of -file, or stdin.
func textOf(text, file string, stdin io.Reader) (string, error) {
	switch {
	case text != "" && file != "":
		return "", fmt.Errorf("%w: give -text or -file, not both", errUsage)
	case text != "":
		return text, nil
	case file != "":
		b, err := os.ReadFile(file)
		return string(b), err
	default:
		b, err := io.ReadAll(stdin)
		return string(b), err
	}
}

func (c *client) capture(args []string, stdin io.Reader) (string, error) {
	fs := flags("capture", args)
	scale := fs.String("scale", "", "summary, detail, primer or book")
	prompt := fs.String("prompt", "", "what you want to know, or what to make of it")
	title := fs.String("title", "", "a title for the primer or the book")
	url := fs.String("url", "", "where the passage came from")
	app := fs.String("app", "", "the app it came from")
	text := fs.String("text", "", "the passage")
	file := fs.String("file", "", "a file holding the passage")
	if err := fs.Parse(args); err != nil {
		return "", fmt.Errorf("%w: %v", errUsage, err)
	}
	if *scale == "" || *prompt == "" {
		return "", fmt.Errorf("%w: capture needs -scale and -prompt", errUsage)
	}
	passage, err := textOf(*text, *file, stdin)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(passage) == "" {
		return "", fmt.Errorf("%w: nothing to capture", errUsage)
	}
	var out map[string]any
	err = c.call("POST", "/primer/capture", map[string]any{
		"text": passage, "prompt": *prompt, "scale": *scale, "title": *title,
		"source_url": *url, "source_app": *app,
	}, &out)
	if err != nil {
		return "", err
	}
	return pretty(out), nil
}

func idOf(fs *flag.FlagSet, args []string) (string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", fmt.Errorf("%w: %s needs a subject id", errUsage, fs.Name())
	}
	if err := fs.Parse(args[1:]); err != nil {
		return "", fmt.Errorf("%w: %v", errUsage, err)
	}
	return args[0], nil
}

func (c *client) plan(args []string) (string, error) {
	fs := flags("plan", args)
	say := fs.String("say", "", "your answer to the tutor")
	id, err := idOf(fs, args)
	if err != nil {
		return "", err
	}
	var out map[string]any
	if *say != "" {
		err = c.call("POST", "/primer/"+id+"/plan", map[string]string{"text": *say}, &out)
	} else {
		err = c.call("GET", "/primer/"+id+"/plan", nil, &out)
	}
	if err != nil {
		return "", err
	}
	return pretty(out), nil
}

func briefOf(brief, file string) (string, error) {
	switch {
	case brief != "" && file != "":
		return "", fmt.Errorf("%w: give -brief or -brief-file, not both", errUsage)
	case file != "":
		b, err := os.ReadFile(file)
		return string(b), err
	default:
		return brief, nil
	}
}

func (c *client) build(args []string) (string, error) {
	fs := flags("build", args)
	brief := fs.String("brief", "", "the brief for the writer")
	briefFile := fs.String("brief-file", "", "a file holding the brief")
	id, err := idOf(fs, args)
	if err != nil {
		return "", err
	}
	b, err := briefOf(*brief, *briefFile)
	if err != nil {
		return "", err
	}
	var body any
	if strings.TrimSpace(b) != "" {
		body = map[string]string{"brief": b}
	}
	var out map[string]any
	if err := c.call("POST", "/primer/"+id+"/build", body, &out); err != nil {
		return "", err
	}
	return pretty(out), nil
}

func (c *client) discard(args []string) (string, error) {
	fs := flags("discard", args)
	id, err := idOf(fs, args)
	if err != nil {
		return "", err
	}
	var out map[string]any
	if err := c.call("POST", "/primer/"+id+"/discard", nil, &out); err != nil {
		return "", err
	}
	return pretty(out), nil
}

// status: the shelf decides what the id is. A draft shows its plan, a
// book being generated its job, anything else its row; the shelf is asked
// first so a wrong guess does not log a miss on the server.
func (c *client) status(args []string) (string, error) {
	fs := flags("status", args)
	id, err := idOf(fs, args)
	if err != nil {
		return "", err
	}
	s, err := c.rows()
	if err != nil {
		return "", err
	}
	var out map[string]any
	for _, r := range s.Subjects {
		if r.ID != id {
			continue
		}
		if r.Status != "" && r.Status != "ready" {
			if err := c.call("GET", "/primer/"+id+"/plan", nil, &out); err == nil {
				return pretty(out), nil
			}
		}
		return pretty(r), nil
	}
	if err := c.call("GET", "/teach/jobs?slug="+id, nil, &out); err == nil {
		return pretty(out), nil
	}
	return "", fmt.Errorf("no subject %q", id)
}

// wait polls the shelf until the subject is ready or failed. A book draft
// hands over to the book it builds: the draft's row goes when the book is
// on the shelf, so the wait follows it there.
func (c *client) wait(args []string) (string, error) {
	fs := flags("wait", args)
	timeout := fs.Duration("timeout", 45*time.Minute, "how long to wait")
	every := fs.Duration("every", 5*time.Second, "how often to look")
	id, err := idOf(fs, args)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	book := ""
	last := ""
	for {
		s, err := c.rows()
		if err != nil {
			return "", err
		}
		var found *row
		for i := range s.Subjects {
			if s.Subjects[i].ID == id {
				found = &s.Subjects[i]
			}
		}
		if found == nil && book != "" {
			for i := range s.Subjects {
				if s.Subjects[i].ID == book {
					found = &s.Subjects[i]
				}
			}
		}
		if found == nil {
			// Not a shelf row: a book being generated from a brief has only
			// its job until it registers.
			var job struct {
				Slug  string `json:"slug"`
				Stage string `json:"stage"`
				Done  int    `json:"units_done"`
				Total int    `json:"units_total"`
				Error string `json:"error"`
			}
			if err := c.call("GET", "/teach/jobs?slug="+id, nil, &job); err != nil {
				return "", fmt.Errorf("no subject %q on the shelf and no job for it", id)
			}
			switch job.Stage {
			case "failed":
				return pretty(job), fmt.Errorf("%s failed: %s", id, job.Error)
			case "ready":
				found = &row{ID: id, Kind: "book"}
			default:
				line := fmt.Sprintf("%s, %d/%d units", job.Stage, job.Done, job.Total)
				if line != last {
					fmt.Fprintf(os.Stderr, "chiron: %s is %s\n", id, line)
					last = line
				}
				select {
				case <-ctx.Done():
					return pretty(job), fmt.Errorf("still %s after %v", job.Stage, *timeout)
				case <-time.After(*every):
				}
				continue
			}
		}
		if found.Book != "" {
			book = found.Book
		}
		switch found.Status {
		case "failed":
			return pretty(found), fmt.Errorf("%s failed: %s", found.ID, found.Error)
		case "", "ready":
			return pretty(found), nil
		}
		if line := where(*found, s.Active); line != last {
			fmt.Fprintf(os.Stderr, "chiron: %s is %s\n", found.ID, line)
			last = line
		}
		select {
		case <-ctx.Done():
			return pretty(found), fmt.Errorf("still %s after %v", found.Status, *timeout)
		case <-time.After(*every):
		}
	}
}

var (
	blockEnd = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|blockquote|pre|tr|section|article|figcaption)>|<br\s*/?>`)
	tags     = regexp.MustCompile(`(?s)<[^>]*>`)
	blanks   = regexp.MustCompile(`\n{3,}`)
)

// plain is the text of a rendered chapter: block ends become line breaks,
// tags go, entities come back.
func plain(h string) string {
	h = blockEnd.ReplaceAllString(h, "\n\n")
	h = tags.ReplaceAllString(h, "")
	h = html.UnescapeString(h)
	lines := strings.Split(h, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.TrimSpace(blanks.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

// page sends a URL to be read here, and prints the card it became.
func (c *client) page(args []string) (string, error) {
	return c.sendLink("page", "/readings/page", args)
}

// follow subscribes to a blog, and prints the card it became.
func (c *client) follow(args []string) (string, error) {
	return c.sendLink("follow", "/feeds", args)
}

func (c *client) sendLink(name, path string, args []string) (string, error) {
	fs := flags(name, args)
	url, err := idOf(fs, args)
	if err != nil {
		return "", fmt.Errorf("%w: %s needs a URL", errUsage, name)
	}
	var out json.RawMessage
	if err := c.call("POST", path, map[string]any{"url": url}, &out); err != nil {
		return "", err
	}
	return pretty(out), nil
}

// feeds lists the blogs followed, or asks them what is new.
func (c *client) feeds(args []string) (string, error) {
	fs := flags("feeds", args)
	check := fs.Bool("check", false, "ask every blog what is new, now")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	var out json.RawMessage
	if *check {
		if err := c.call("POST", "/feeds/refresh?force=1", nil, &out); err != nil {
			return "", err
		}
		return pretty(out), nil
	}
	if err := c.call("GET", "/feeds", nil, &out); err != nil {
		return "", err
	}
	return pretty(out), nil
}

func (c *client) read(args []string) (string, error) {
	fs := flags("read", args)
	unit := fs.String("unit", "", "a unit of a book; the current one when empty")
	id, err := idOf(fs, args)
	if err != nil {
		return "", err
	}
	var out struct {
		Chapter *struct {
			Unit  string `json:"unit"`
			Title string `json:"title"`
			HTML  string `json:"html"`
		} `json:"chapter"`
		Authoring      bool   `json:"authoring"`
		AuthoringError string `json:"authoring_error"`
	}
	path := "/chapter/" + id
	if *unit != "" {
		path += "?unit=" + *unit
	}
	if err := c.call("GET", path, nil, &out); err != nil {
		return "", err
	}
	if out.Chapter == nil && *unit == "" && !out.Authoring && out.AuthoringError == "" {
		// Nothing has been opened yet: open it the way the app does, which
		// makes the first unit current and returns its chapter.
		if err := c.call("POST", "/exchange", map[string]string{"subject": id, "phase": "start"}, &out); err != nil {
			return "", err
		}
	}
	if out.Chapter == nil {
		if out.AuthoringError != "" {
			return "", fmt.Errorf("%s: %s", id, out.AuthoringError)
		}
		if out.Authoring {
			return "", fmt.Errorf("%s is still being written", id)
		}
		return "", fmt.Errorf("%s has no chapter to read", id)
	}
	return fmt.Sprintf("# %s\n\n%s\n", out.Chapter.Title, plain(out.Chapter.HTML)), nil
}

func (c *client) teach(args []string) (string, error) {
	fs := flags("teach", args)
	title := fs.String("title", "", "the book's title")
	slug := fs.String("slug", "", "its id on the shelf; from the title when empty")
	brief := fs.String("brief", "", "what the book should teach, and to whom")
	briefFile := fs.String("brief-file", "", "a file holding the brief")
	var srcs multi
	fs.Var(&srcs, "source", "an open text to build from; repeat for interleaves")
	planOnly := fs.Bool("plan-only", false, "write the syllabus and stop; the same command again authors from it")
	if err := fs.Parse(args); err != nil {
		return "", fmt.Errorf("%w: %v", errUsage, err)
	}
	b, err := briefOf(*brief, *briefFile)
	if err != nil {
		return "", err
	}
	if *title == "" || strings.TrimSpace(b) == "" {
		return "", fmt.Errorf("%w: teach needs -title and a brief", errUsage)
	}
	if *slug == "" {
		*slug = *title
	}
	body := map[string]any{"slug": *slug, "title": *title, "brief": b}
	if len(srcs) > 0 {
		body["sources"] = []string(srcs)
	}
	if *planOnly {
		body["plan_only"] = true
	}
	var out map[string]any
	if err := c.call("POST", "/teach/create", body, &out); err != nil {
		return "", err
	}
	return pretty(out), nil
}

// multi is a repeatable string flag.
type multi []string

func (m *multi) String() string     { return strings.Join(*m, ", ") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }
