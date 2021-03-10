# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

BINARY := build/lightChain

all: build test

build: deps
	@echo "==> Go build"
	@go build -o $(BINARY)

deps:
	@go get -v -u github.com/boltdb/bolt

test:
	@echo "==> Running"

	@echo "==> Call printchain:"
	@./$(BINARY) printchain

	@echo "==> Call addblock:"
	@./$(BINARY) addblock -data "Send 1 LIG to hliangzhao"

	@echo "==> Call addblock:"
	@./$(BINARY) addblock -data "Pay 0.13334 LIG for a coffee"

	@echo "==> Call printchain:"
	@./$(BINARY) printchain

.PHONY: build deps test