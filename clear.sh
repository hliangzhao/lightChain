#!/bin/bash

BIN=./build/lightChain

echo "Clear environment..."
ADDR_DIR=tmp
DB_DIR=db
WALLETS_DIR=wallets
if [ -e $ADDR_DIR ]; then
  rm -rf $ADDR_DIR/*
fi
if [ -e $DB_DIR ]; then
  rm -rf $DB_DIR/*
fi
if [ -e $WALLETS_DIR ]; then
  rm -rf $WALLETS_DIR/*
fi