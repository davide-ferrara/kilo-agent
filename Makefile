BIN=kilo-agent
GOFLAGS=-ldflags "-s -w"
BINDIR?=$(HOME)/.local/bin

RUN=$(BIN)
TESTFLAGS?=-v

.PHONY: all run build test clean install uninstall vet fmt

all: build

build:
	go build $(GOFLAGS) -o $(BIN) .

run: build
	./$(RUN)

test:
	go test $(TESTFLAGS) ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BIN)

install: build
	install -d -m 0755 "$(BINDIR)"
	install -m 0755 "$(BIN)" "$(BINDIR)/$(BIN)"

uninstall:
	rm -f -- "$(BINDIR)/$(BIN)"
