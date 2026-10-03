BIN := vpn
PKG := ./...
VERSION ?= dev
PREFIX ?= /usr/local

.PHONY: build test test-race vet lint golden integration clean install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/vpn

install: build
	install -m 0755 $(BIN) $(PREFIX)/bin/$(BIN)

test:
	go test $(PKG)

test-race:
	go test -race $(PKG)

vet:
	go vet $(PKG)

lint:
	golangci-lint run ./...

golden:
	UPDATE_GOLDEN=1 go test ./internal/core/singbox/

integration:
	go test -tags integration ./internal/core/singbox/ -v

clean:
	rm -f $(BIN)
