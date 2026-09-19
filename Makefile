# Day-to-day targets. `make deploy` is meant to run ON the sprite, from the
# checkout at /home/sprite/src/chiron, and swaps the served binary the way
# scripts/sprite-deploy.sh does from the Mac.

GO      := cd server-go && go
SERVED  := /home/sprite/chiron
BIN     := $(SERVED)/bin/chiron-server

.PHONY: test lint deploy deploy-gate app-build

# On the sprite (which also serves the book, on 8 GB) the suite runs two
# packages at a time and without browser renders, so a test run never
# competes with the reader for memory. On a Mac renders are on; RENDER=0
# turns them off.
RENDER ?= $(shell [ -d /.sprite ] && echo 0 || echo 1)

test:
	cd server-go && CHIRON_RENDER=$(RENDER) go test -p 2 ./...
	node scripts/test-book-js.mjs

lint:
	$(GO) run ./cmd/corpus-lint ../corpus
	$(GO) run ./cmd/corpus-lint ../corpus-v2

# Build, keep yesterday's binary under a dated name, swap, restart, verify.
# The book server sits on 8081 behind the gate; 8080 answers through it.
deploy: test
	$(GO) build -ldflags="-s -w" -o $(BIN).new ./cmd/chiron-server
	[ -f $(BIN) ] && cp $(BIN) $(BIN).$$(date +%b%d | tr A-Z a-z) || true
	mv $(BIN).new $(BIN)
	# The served corpus is a copy, not the checkout: the files the server
	# reads at build time (the authoring contract, the source index) go
	# with the binary. Unit files stay: they are the live book.
	mkdir -p $(SERVED)/corpus/sources
	cp corpus/authoring-spec.md $(SERVED)/corpus/authoring-spec.md
	cp corpus/sources/index.yaml $(SERVED)/corpus/sources/index.yaml
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

# The app, built and signed on the MacBook (SPRITE-DEV-PLAN.md phase E):
# push main there, build it, bring the build back under the served
# tree's builds/<token>/, where the gate offers it to the devices.
app-build:
	git push -q macbook main
	@id=$$(ssh macbook build main | tail -1) && [ -n "$$id" ] && \
	  token=$$(ssh macbook fetch $$id build.json | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])') && \
	  dir=$(SERVED)/builds/$$token && mkdir -p $$dir && \
	  for f in build.json manifest.plist Chiron.ipa Chiron-mac.zip build.log; do ssh macbook fetch $$id $$f > $$dir/$$f; done && \
	  ln -sfn $$token $(SERVED)/builds/latest && \
	  python3 -c 'import json,sys; b=json.load(open(sys.argv[1])); print("build", b["id"], b["status"], b["version"], "(%d)" % b["build"], "tests", b["tests"])' $$dir/build.json
