# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

BIN := build/lightChain

all: build

build: deps
	@echo "==> Go build"
	@go build -o $(BIN)
	@echo "==> Build successfully!"

deps:
	@go get -v -u github.com/boltdb/bolt
	@go get -v -u golang.org/x/crypto/ripemd160

.PHONY: build deps