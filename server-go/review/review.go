// Package review builds the post-session spaced-review schedule.
//
// Multi-day spacing is the one high-utility technique a single long session
// cannot use, so the session's payoff is banked by exporting what to review and
// when. Cepeda's ridgeline puts the optimal gap at roughly 10-20% of the target
// retention interval; for "still know this months from now" that lands on
// day 1 / day 3 / day 10 from the session.
//
// What goes in each pass follows the confidence x accuracy routing used during
// the session:
//
//	day 1  - everything missed, and every active misconception (highest decay
//	         risk, and wrong models harden if left uncorrected)
//	day 3  - the day-1 set again, plus fragile items (right but unconfident, or
//	         shaky concepts) - retrieval while still retrievable
//	day 10 - one item per concept covered, error-weighted; a cumulative sweep
//	         rather than a re-drill
//
// Items carry their prompt and reference answer so the schedule is
// self-contained: it has to work with no server, no model and no network.
package review

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/state"
)

type pass struct {
	label  string
	offset int
}

var passes = []pass{{"Day 1", 1}, {"Day 3", 3}, {"Day 10", 10}}

type item struct {
	ID      string
	Unit    string
	Prompt  string
	Answer  string
	Concept string
}

// missedItems returns items graded wrong, newest verdict winning - a later pass
// can redeem an earlier miss.
func missedItems(logPath string) []state.Event {
	f, err := os.Open(logPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	latest := map[string]state.Event{}
	var order []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		var ev state.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Kind != "item_graded" {
			continue
		}
		if _, seen := latest[ev.Item]; !seen {
			order = append(order, ev.Item)
		}
		latest[ev.Item] = ev
	}
	var out []state.Event
	for _, id := range order {
		ev := latest[id]
		if ev.Verdict != "pass" && ev.Verdict != "valid_alternative_path" {
			out = append(out, ev)
		}
	}
	return out
}

func lookup(c *corpus.Corpus, itemID string) *item {
	if q, unit := c.FindQuestion(itemID); q != nil {
		return &item{ID: itemID, Unit: unit, Prompt: q.Prompt,
			Answer: q.Answer.String(), Concept: q.Concept}
	}
	if b, unit := c.FindBeat(itemID); b != nil {
		return &item{ID: itemID, Unit: unit, Prompt: b.Prompt,
			Answer: b.Answer.String(), Concept: b.Concept}
	}
	return nil
}

// Build renders the schedule as self-contained markdown.
func Build(l *state.Learner, c *corpus.Corpus, subjectTitle string, today time.Time) string {
	if today.IsZero() {
		today = time.Now()
	}
	var missed []item
	for _, ev := range missedItems(l.LogPath()) {
		if it := lookup(c, ev.Item); it != nil {
			missed = append(missed, *it)
		}
	}
	fragile := l.FragileConcepts()
	misconceptions := l.ActiveMisconceptions()

	snapshot := l.Snapshot()
	var mastered []string
	for cid, con := range snapshot.Concepts {
		if con.Level == "mastered" {
			mastered = append(mastered, cid)
		}
	}
	sort.Strings(mastered)
	masteredSet := map[string]bool{}
	for _, cid := range mastered {
		masteredSet[cid] = true
	}

	// One representative item per concept for the day-10 sweep, preferring
	// concepts that produced an error at some point.
	byConcept := map[string]item{}
	var conceptOrder []string
	add := func(cid string, it item) {
		if cid == "" {
			return
		}
		if _, seen := byConcept[cid]; seen {
			return
		}
		byConcept[cid] = it
		conceptOrder = append(conceptOrder, cid)
	}
	for _, it := range missed {
		add(it.Concept, it)
	}
	for _, uid := range c.UnitOrder() {
		u, ok := c.Units[uid]
		if !ok {
			continue
		}
		for _, q := range u.Questions.Check {
			if q.Concept != "" && masteredSet[q.Concept] {
				add(q.Concept, item{ID: q.ID, Unit: u.ID, Prompt: q.Prompt,
					Answer: q.Answer.String(), Concept: q.Concept})
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Review schedule - %s\n\n", subjectTitle)
	fmt.Fprintf(&b, "Session %s. Retrieval beats rereading: cover the answer, say it "+
		"out loud, then check. A pass you skip is the pass that mattered.\n\n",
		today.Format("2006-01-02"))

	for _, p := range passes {
		due := today.AddDate(0, 0, p.offset)
		fmt.Fprintf(&b, "## %s - %s\n\n", p.label, due.Format("2006-01-02"))

		var pool []item
		var note string
		switch p.label {
		case "Day 1":
			pool, note = missed, "Everything you missed, while the corrections are still fresh."
		case "Day 3":
			pool, note = missed, "The same misses again, plus anything below."
		default:
			for _, cid := range conceptOrder {
				pool = append(pool, byConcept[cid])
			}
			note = "One item per concept covered - a sweep, not a re-drill."
		}
		b.WriteString(note + "\n\n")

		if len(pool) == 0 {
			b.WriteString("Nothing outstanding for this pass.\n\n")
		}
		for _, it := range pool {
			answer := strings.TrimSpace(it.Answer)
			if answer == "" {
				answer = "(see the chapter)"
			}
			fmt.Fprintf(&b, "**[%s]** %s\n\n> %s\n\n", it.Unit, strings.TrimSpace(it.Prompt), answer)
		}

		if (p.label == "Day 1" || p.label == "Day 3") && len(misconceptions) > 0 {
			b.WriteString("### Wrong models to actively contradict\n\n")
			for _, mid := range misconceptions {
				m, ok := c.Misconceptions[mid]
				if !ok {
					continue
				}
				fmt.Fprintf(&b, "- **%s** - you leaned on: *%s*\n", m.Name, m.WrongModel)
				fmt.Fprintf(&b, "  - It fails because: %s\n", m.FailingPrediction)
				fmt.Fprintf(&b, "  - True: %s\n\n", m.Correction)
			}
		}

		if p.label == "Day 3" && len(fragile) > 0 {
			b.WriteString("### Shaky - answered right but without confidence, or missed once\n\n")
			for _, cid := range fragile {
				name, unit := cid, "?"
				if u := c.ConceptUnit(cid); u != "" {
					unit = u
				}
				for _, uid := range c.UnitOrder() {
					u, ok := c.Units[uid]
					if !ok {
						continue
					}
					for _, ref := range u.Concepts {
						if ref.ID == cid {
							name = ref.Name
						}
					}
				}
				fmt.Fprintf(&b, "- %s (%s)\n", name, unit)
			}
			b.WriteString("\n")
		}
	}

	fmt.Fprintf(&b, "---\n\nConcepts mastered this session: %d. Still shaky: %d. "+
		"Misconceptions still active: %d.\n", len(mastered), len(fragile), len(misconceptions))
	return b.String()
}
