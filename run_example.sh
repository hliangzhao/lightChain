#!/bin/bash

BIN=./build/lightChain

echo "Clear environment..."
ADDR_FILE=./addresses.dat
DB_FILE=./lightChain.db
WALLETS_FILE=./wallets.dat
if [ -e $ADDR_FILE ]; then
  rm $ADDR_FILE
fi
if [ -e $DB_FILE ]; then
  rm $DB_FILE
fi
if [ -e $WALLETS_FILE ]; then
  rm $WALLETS_FILE
fi

echo "==> Creat wallet twice:"
$BIN createwallet
$BIN createwallet

echo "==> List all addresses:"
$BIN listaddr

echo "==> Creat lightChain:"
$BIN createchain -addr $( sed -n 1p addresses.dat )

echo "==> Print lightChain:"
$BIN printchain

echo "==> Call send twice:"
$BIN  send -src $( sed -n 1p addresses.dat ) -dst $( sed -n 2p addresses.dat ) -amount 520.0
$BIN  send -src $( sed -n 2p addresses.dat ) -dst $( sed -n 1p addresses.dat ) -amount 52.1

echo "==> Print the coinbase transaction in genesis block:"
$BIN printtx -b 3 -tx 0

echo "==> Show balance:"
$BIN getbalance -addr $( sed -n 1p addresses.dat )
$BIN getbalance -addr $( sed -n 2p addresses.dat )

echo "==> Print blocks' number:"
$BIN getblocknum

echo "==> Print lightChain:"
$BIN printchain