# Day-to-day targets. `make deploy` is meant to run ON the sprite, from the
# checkout at /home/sprite/src/chiron, and swaps the served binary the way
# scripts/sprite-deploy.sh does from the Mac.

GO      := cd server-go && go
SERVED  := /home/sprite/chiron
BIN     := $(SERVED)/bin/chiron-server

.PHONY: test lint deploy deploy-gate

# -p 2: the sprite that runs these also serves the book and has 8 GB; the
# full-parallel suite (page renders through Chromium included) took the
# whole sprite down once, and the restore lost half an hour of writes.
test:
	$(GO) test -p 2 ./...
	node scripts/test-book-js.mjs

lint:
	$(GO) run ./cmd/corpus-lint corpus corpus-v2

# Build, keep yesterday's binary under a dated name, swap, restart, verify.
# The book server sits on 8081 behind the gate; 8080 answers through it.
deploy: test
	$(GO) build -ldflags="-s -w" -o $(BIN).new ./cmd/chiron-server
	[ -f $(BIN) ] && cp $(BIN) $(BIN).$$(date +%b%d | tr A-Z a-z) || true
	mv $(BIN).new $(BIN)
	sprite-env services restart chiron-server
	sleep 3
	@printf 'server /ping  %s\n' "$$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8081/ping)"
	@printf 'gate   /ping  %s\n' "$$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/ping)"

# The gate is the way in; deploy it only from a session that can survive
# the restart (tmux keeps the shell, ssh reconnects through the new gate).
deploy-gate:
	$(GO) build -ldflags="-s -w" -o $(SERVED)/bin/chiron-gate.new ./cmd/chiron-gate
	cp $(SERVED)/bin/chiron-gate $(SERVED)/bin/chiron-gate.$$(date +%b%d | tr A-Z a-z) || true
	mv $(SERVED)/bin/chiron-gate.new $(SERVED)/bin/chiron-gate
	sprite-env services restart chiron-gate
