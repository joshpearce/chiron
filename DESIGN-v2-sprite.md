# Chiron v2: sprite-backed cloud tutor (design capture, not built)

Idea (Matt, 2026-07-26): each Chiron user gets a Fly.io sprite hosting a
Claude instance as their tutor backend. Read `~/.claude/sprites.md` before
implementing.

## Chapter loop on a sprite

- iPad hits the SAME `/exchange` protocol, pointed at the sprite's URL.
- Sprite wakes on request, Claude runs the grader/planner/author roles
  (dramatically better than local Qwen, especially free-text grading and
  section rewriting), returns the payload, sprite sleeps. Cost profile fits
  the duty cycle: minutes of reading between seconds of inference.
- The existing server code is the starting point: `llm.py`'s upstream chain
  gains an Anthropic-API upstream (or the roles migrate to Claude Agent SDK
  running IN the sprite, which also unlocks tool use - e.g. mechanical
  verification of generated numbers before they ship).
- Tiering: sprite+Claude (online) -> Mac+LM Studio (offline flights) ->
  built-in static bundle (nothing available). App already handles the
  fallback ladder; add the sprite URL as the first-choice base URL.

## "Teach me" - user-driven curriculum generation

Flow: app gains a Teach Me chat. Dialog with Claude (in the sprite) elicits
scope, prior knowledge, goal depth, time budget -> converges to a brief.
Then the sprite runs tonight's pipeline as a batch job:

1. Author `syllabus.yaml` (prereq graph, time budgets, extension triggers)
   + `misconception-bank.yaml` seeded for the domain
2. Fan out unit authoring against `corpus/authoring-spec.md` (the contract
   is already subject-agnostic)
3. Adversarial verify pass (math recomputation, tolerance-collision audit,
   rubric adjudication) - non-negotiable; it caught ~40 real defects in the
   AI corpus
4. Render + register as a new subject; deliver to the app

Server already supports multiple subjects (config list + per-subject state);
delivery = the app's subject library refreshing from `/subjects`.

## Async delivery / notifications

- Curriculum generation takes tens of minutes -> async completion signal.
- APNs requires the PAID Apple Developer Program; free personal-team
  provisioning has no push entitlement. Decide: $99/yr for real pushes, or
  poll-on-foreground (subject library already refreshes on appear - "your
  book is ready" just shows up). Background App Refresh + polling is a
  middle path without APNs.
- If paid account: token-based APNs from the sprite is straightforward.

## Open questions

- Auth between app and sprite (per-user sprite -> per-user token; sprites.md
  patterns apply)
- Learner-state residency: sprite-local (event log, as today) with export?
- Cost ceiling per user for authoring runs (a full verified book was ~10
  author agents + 4 verify agents of frontier-model work)
- Sharing: can a generated+verified subject be published to other users?
