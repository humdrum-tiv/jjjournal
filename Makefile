BIN ?= $(HOME)/bin
REPO ?= .

.PHONY: build install check smoke

build:
	mkdir -p bin
	go build -o bin/jjj ./cmd/jjj

install:
	mkdir -p $(BIN)
	go build -o $(BIN)/jjj ./cmd/jjj

# Everything that must stay green: our tests, upstream's tests, and the
# untouched upstream binary.
check:
	go vet ./...
	go build ./...
	go build -o /dev/null ./cmd/jjui
	go test ./...

smoke: build
	python3 scripts/pty-smoke.py bin/jjj $(REPO) --wait 12 --expect "revset:"
