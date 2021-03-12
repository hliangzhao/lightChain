# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

BIN := build/lightChain

all: build test

build: dependencies
	@echo "==> Go build"
	@go build -o $(BIN)
	@echo "==> Build successfully!\n\n"

dependencies:
	@go get -v -u github.com/boltdb/bolt
	@go get -v -u golang.org/x/crypto/ripemd160

test:
	@echo "==> Running test"

	@echo "==> Create lightChain"
	@./$(BIN) createchain -addr "hliangzhao"

	@echo "==> Call printchain:"
	@./$(BIN) printchain

	@echo "==> Call send:"
	@./$(BIN) send -src "hliangzhao" -dst "alei" -amount 520

	@echo "==> Call send:"
	@./$(BIN) send -src "hliangzhao" -dst "alei" -amount 6

	@echo "==> Call getbalance:"
	@./$(BIN) getbalance -addr "hliangzhao"

	@echo "==> Call getbalance:"
	@./$(BIN) getbalance -addr "alei"

	@echo "==> Call printchain:"
	@./$(BIN) printchain

.PHONY: build dependencies test