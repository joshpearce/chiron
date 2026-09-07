package corpus

import (
	"hash/fnv"
	"math/rand"

	"github.com/mjbraun/chiron/server/checkers"
)

// Shuffled puts an item's options in an order fixed by the item's id, so
// the correct option is not the first one every author wrote first, and
// so every parse of the item, for delivery or for grading, agrees on the
// order without the file changing. A screener keeps its authored order,
// which runs novice to expert.
func Shuffled(id string, opts []checkers.Option) []checkers.Option {
	if len(opts) < 2 {
		return opts
	}
	h := fnv.New64a()
	h.Write([]byte(id))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	out := make([]checkers.Option, len(opts))
	copy(out, opts)
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (q *Question) shuffleOptions() {
	if q.Check == "screener" {
		return
	}
	q.Options = Shuffled(q.ID, q.Options)
}
