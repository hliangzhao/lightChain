# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

BINARY := build/lightChain

all: build run

build:
	@echo "==> Go build"
	@go build -o $(BINARY)

run:
	@echo "==> Running"
	@./$(BINARY)

.PHONY: build run