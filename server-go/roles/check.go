package roles

import (
	// math/rand, deliberately: this shuffles quiz items and breaks ties between
	// equally-weighted callbacks. Nothing here is a secret or a token, and a
	// seedable generator is what makes check composition reproducible in tests.
	"math/rand"
	"sort"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/state"
)

// ComposeCheck builds the cumulative check: roughly 60% current unit,
// constructed items first, and 40% callbacks from prior units weighted toward
// fragile concepts and active misconceptions.
//
// This is deliberately deterministic code rather than a model call. What gets
// asked decides what the learner retains, and it should not vary with a
// model's mood.
func ComposeCheck(unit *corpus.Unit, l *state.Learner, c *corpus.Corpus,
	nItems int, callbackFraction float64, rng *rand.Rand) []corpus.Question {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	current := append([]corpus.Question{}, unit.Questions.Check...)
	rng.Shuffle(len(current), func(i, j int) { current[i], current[j] = current[j], current[i] })
	// Constructed items first: the goal is to derive and explain, and transfer
	// is measurably better when the practice format matches the goal format.
	sort.SliceStable(current, func(i, j int) bool {
		return rank(current[i]) < rank(current[j])
	})

	nCurrent := int(float64(nItems)*(1-callbackFraction) + 0.5)
	if nCurrent < 1 {
		nCurrent = 1
	}
	if nCurrent > len(current) {
		nCurrent = len(current)
	}
	items := make([]corpus.Question, 0, nItems)
	for _, q := range current[:nCurrent] {
		q.Unit = unit.ID
		items = append(items, q)
	}

	fragile := setOf(l.FragileConcepts())
	activeMis := setOf(l.ActiveMisconceptions())

	type weighted struct {
		weight int
		tie    float64
		q      corpus.Question
	}
	var callbacks []weighted
	cleared := l.ClearedUnits()
	for _, uid := range c.UnitOrder() {
		if !cleared[uid] || uid == unit.ID {
			continue
		}
		prior, ok := c.Units[uid]
		if !ok {
			continue
		}
		for _, q := range prior.Questions.Check {
			if !q.CallbackEligible {
				continue
			}
			w := 1
			if fragile[q.Concept] {
				w += 2
			}
			for _, o := range q.Options {
				if o.Misconception != "" && activeMis[o.Misconception] {
					w += 2
					break
				}
			}
			q.Unit = uid
			callbacks = append(callbacks, weighted{w, rng.Float64(), q})
		}
	}
	sort.Slice(callbacks, func(i, j int) bool {
		if callbacks[i].weight != callbacks[j].weight {
			return callbacks[i].weight > callbacks[j].weight
		}
		return callbacks[i].tie < callbacks[j].tie
	})
	for _, cb := range callbacks {
		if len(items) >= nItems {
			break
		}
		items = append(items, cb.q)
	}

	// Early units have no callback pool; top up from the current unit rather
	// than shipping a short check. Fewer than eight items is statistical noise
	// for an 80% gate.
	if len(items) < nItems {
		seen := map[string]bool{}
		for _, q := range items {
			seen[q.ID] = true
		}
		for _, q := range current {
			if len(items) >= nItems {
				break
			}
			if !seen[q.ID] {
				q.Unit = unit.ID
				items = append(items, q)
			}
		}
	}
	rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	return items
}

func rank(q corpus.Question) int {
	if q.Kind == "constructed" {
		return 0
	}
	return 1
}

func setOf(xs []string) map[string]bool {
	out := make(map[string]bool, len(xs))
	for _, x := range xs {
		out[x] = true
	}
	return out
}
