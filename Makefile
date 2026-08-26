BIN=kilo-agent
GOFLAGS=-ldflags "-s -w"

RUN=$(BIN)
TESTFLAGS?=-v

.PHONY: all run build test clean install vet fmt

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