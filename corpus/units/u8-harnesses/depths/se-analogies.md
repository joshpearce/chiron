---
unit: u8
depth: se-analogies
---

Every analogy here comes in a pair, because a single analogy is a trap: you keep
whichever half is convenient and never find the seam. Two analogies that agree
on a structure make the structure the thing you remember. Each pair states where
it breaks, and that clause is the load-bearing one.

## It is all one token stream

**A conversation is like a stateless HTTP request whose body is the entire prior
history, and like an append-only event log replayed from offset zero on every
read.**

Shared structure: no server-side session. The full state travels with every
request, the receiver holds nothing between requests, and correctness depends
only on what is in the payload. Both make "what does the server remember?" a
category error - the answer is nothing, and the client re-supplies everything.

Where this breaks: an HTTP body is order-insensitive at the structural level (a
JSON object's keys can be permuted; the parser does not care) and an event log's
replay is deterministic per-event. Neither is true here. Position is part of the
computation - the same tokens in a different order are a different input, and
where a token sits relative to the end changes how much attention it receives.
And unlike a stateless server, part of the work *is* memoized across requests:
the KV prefix cache. So it is stateless in semantics and stateful in
performance, which is an unusual combination and the source of most surprises.

**Role delimiters are like SQL parameterized queries, and like HTML escaping of
user input.**

Shared structure: a lexical boundary that keeps attacker-supplied data from
being read as control structure. Both work by ensuring the data cannot express
the delimiter - a bound parameter is never parsed as SQL, an escaped `<` is never
parsed as a tag. Reserved special token IDs do exactly this: user characters
cannot encode `<|start|>`.

Where this breaks - and this is the most important sentence in this file -
parameterized SQL is a *complete* defense, because after parameterization the
value is inert. It is data forever; there is no path by which it becomes
executable. Token delimiting is not complete, because the consumer downstream of
the boundary is not a parser with a grammar. It is a next-token predictor over
one flat sequence, in which "this is an instruction" is a learned property of
how text reads, not a property of which side of a delimiter it sits on.
Parameterizing SQL removes the attack. Delimiting roles removes only the
forgery, and leaves persuasion fully intact. Anyone reasoning about prompt
injection with the SQL-injection analogy in mind will over-trust their harness.

## The tool loop: the agent is the loop

**A tool call is like a userspace process trapping to the kernel on a syscall,
and like a coroutine yielding to a scheduler.**

Shared structure: computation suspends at a well-defined point, an external
party performs an effect the suspended computation could not perform itself, a
result is written where the computation will see it, and execution resumes.
In all three cases the suspended party has no ability to do the work directly -
that is the whole reason for the boundary.

Where this breaks: a syscall returns into the same stack frame with the register
file and the entire address space intact, and a coroutine resumes with its
locals live. The model resumes with none of that, because there is no frame, no
registers, and no locals - the only thing that persists is tokens. The
"resumption" is a fresh forward pass over a longer string. Anything the model
had "in mind" that it did not write into the stream is gone. That is why
scratchpad and chain-of-thought output are not a stylistic choice: writing it
down is the only mechanism by which intermediate state survives a tool call.

Second break: a syscall's return value is produced by a kernel the process is
obliged to trust and cannot inspect. Here the "kernel" is your harness plus an
arbitrary MCP server, the return value is prose, and the model has no way to
distinguish a real result from a fabricated one. `read()` cannot lie to you about
having read. A tool absolutely can.

**MCP is like the Language Server Protocol, and like a JDBC driver.**

Shared structure: a standardized protocol between a host application and a
pluggable capability provider, with runtime discovery of what the provider
offers, so the host can integrate an arbitrary backend it was not compiled
against. Your editor does not know Rust; it speaks LSP to something that does.
Your harness does not know your ticket system; it speaks MCP to something that
does.

Where this breaks: in LSP and JDBC, the client that speaks the protocol is also
the thing that consumes the results programmatically - the editor parses
structured completions and renders them. In MCP the consumer is a language
model that does not speak the protocol at all. The harness sits in the middle
doing a lossy translation in both directions: it renders discovered tool schemas
*into English-ish JSON in the prompt*, and it parses generated text back into a
dispatch. There is no type checking across that gap, only a trained tendency to
emit conforming JSON. An LSP client that sends a malformed request has a bug. A
model that emits a malformed tool call is behaving normally and your harness
needs a retry path.

## What the loop costs

**Prefix caching is like Docker layer caching, and like a Merkle-tree /
content-addressed build cache.**

Shared structure: work is memoized against an exact prefix or content hash of
its inputs; a hit lets you skip straight to the first divergence; and a change
early in the chain invalidates everything downstream regardless of whether the
downstream content itself changed. The practical advice is identical in all
three: put volatile things last. `COPY package.json` before `COPY .` is the same
optimization as putting your timestamp at the end of the context rather than in
the system prompt.

Where this breaks: Docker's cache keys are explicit and inspectable, and layer
boundaries are ones you authored. KV cache boundaries are not yours - they are
block-granular on the server side (caches are paged, often in blocks of some
tens of tokens), invisible to you, and subject to eviction for reasons that have
nothing to do with your content: TTL, memory pressure, another tenant. A Docker
cache miss is deterministic and reproducible. A KV cache miss can happen twice
with byte-identical input, which means cache economics are statistical, not
guaranteed, and you should design for a good hit rate rather than assume a hit.

**The naive re-prefill cost is like an $O(n^2)$ string-concatenation loop, and
like re-reading a whole file on every append instead of seeking to the end.**

Shared structure: an operation that ought to cost the size of the increment
instead costs the size of the accumulated whole, so a loop of $k$ appends costs
$O(k^2)$. Every engineer has fixed this bug in a string builder.

Where this breaks: the string-concat fix is total - a rope or a buffer makes
append genuinely $O(1)$ amortized, and the quadratic disappears. Prefix caching
only removes the *re-prefill* quadratic. The attention quadratic underneath
survives, because each newly appended token must attend to every token behind
it, and that is inherent to the architecture rather than to any caching choice.
So the correct expectation after adding prefix caching is "much cheaper, and
still superlinear in session length", not "fixed".

## Context is the scarce resource

**Compaction is like Kafka log compaction, and like a generational garbage
collector.**

Shared structure: unbounded history meets bounded storage, so you discard what
is judged no longer needed and keep a smaller representation that preserves what
matters. Both buy headroom by forgetting on purpose.

Where this breaks, on both halves, and the break is severe. Kafka's compaction
is a deterministic rule with a stated guarantee: keep the latest value per key,
and the guarantee is checkable. A GC has a reachability proof - it collects only
what provably cannot be referenced again. LLM compaction has neither. It is a
*model* summarizing, so the retention policy is a sample from a distribution;
there is no key, no proof, and no invariant. Nothing establishes that the
discarded tokens were dead, and the same conversation compacted twice yields
different summaries. Treat compaction as lossy transcoding of a recording, not
as garbage collection - the useful instinct from that framing is that you should
be reluctant to do it repeatedly, because generation loss accumulates.

**A sub-agent is like `fork()` with a fresh address space, and like a map-reduce
worker.**

Shared structure: isolated memory, work performed out of sight, and only a small
result crossing back to the caller. The parent's own memory footprint does not
grow with the child's working set, which is the entire reason to do it.

Where this breaks: `fork()` copies the parent's address space, so the child
starts knowing everything the parent knew. A sub-agent starts knowing *only what
you explicitly serialized into its prompt* - it is much closer to spawning a
process with an empty environment and a single argv string. Everything the
parent knew that you did not restate is unavailable, silently, and the sub-agent
cannot ask. And unlike a map-reduce worker, whose output is a typed partial
result that composes deterministically with its siblings, a sub-agent returns
prose, and two sub-agents' prose reports may quietly contradict each other with
nothing in the system able to notice.

## Putting the three claims together

**The harness is like a REPL driving a pure function, and like a game loop
driving a stateless renderer.**

Shared structure: all state lives in the driver; the callee is a pure function
from state to output; and the illusion of continuity comes entirely from the
driver calling the function repeatedly with an evolving state. `render(state)`
does not remember the last frame. `f(tokens)` does not remember the last turn.

Where this breaks: in both a REPL and a game loop, the driver holds state in a
rich structured form of its own choosing, and only projects it for the callee.
Here the state *is* the callee's input format - the token stream is both your
representation and the wire format, so you cannot hold richer state cheaply and
project it. Every piece of state you want the model to act on must be paid for
in tokens, at that turn, at the going rate. There is no free `state` struct on
the harness side that the model can consult. That single constraint is what
makes context management a first-class engineering problem rather than a
tuning detail.
