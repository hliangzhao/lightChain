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

echo "==> Creat two wallets:"
$BIN createwallet
$BIN createwallet

echo "==> List all addresses:"
$BIN listaddr

echo "==> Creat lightChain:"
$BIN createchain -addr $( sed -n 1p addresses.dat )

echo "==> Print lightChain:"
$BIN printchain

echo "==> Send from addr1 to addr2:"
$BIN send -src $( sed -n 1p addresses.dat ) -dst $( sed -n 2p addresses.dat ) -amount 520
echo "==> Show balance:"
$BIN getbalance -addr $( sed -n 1p addresses.dat )
$BIN getbalance -addr $( sed -n 2p addresses.dat )
echo "==> Rebuild UTXO:"
$BIN rebuildutxo

echo "==> Send from addr2 to addr1:"
$BIN send -src $( sed -n 2p addresses.dat ) -dst $( sed -n 1p addresses.dat ) -amount 100
echo "==> Show balance:"
$BIN getbalance -addr $( sed -n 1p addresses.dat )
$BIN getbalance -addr $( sed -n 2p addresses.dat )
echo "==> Rebuild UTXO:"
$BIN rebuildutxo

echo "==> Send from addr2 to addr1:"
$BIN send -src $( sed -n 2p addresses.dat ) -dst $( sed -n 1p addresses.dat ) -amount 100
echo "==> Show balance:"
$BIN getbalance -addr $( sed -n 1p addresses.dat )
$BIN getbalance -addr $( sed -n 2p addresses.dat )
echo "==> Rebuild UTXO:"
$BIN rebuildutxo

echo "==> Print blocks' number:"
$BIN getblocknum

echo "==> Print all transactions:"
echo "== Block #0 =="
$BIN printtx -b 2 -tx 0
echo "== Block #1 =="
$BIN printtx -b 1 -tx 0
$BIN printtx -b 1 -tx 1
echo "== Block #2 =="
$BIN printtx -b 0 -tx 0
$BIN printtx -b 0 -tx 1

echo "==> Print lightChain:"
$BIN printchain