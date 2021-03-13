// Copyright 2021 Hailiang Zhao <hliangzhao@zju.edu.cn>
// This file is part of the lightChain.
//
// The lightChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lightChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lightChain. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	`flag`
	`fmt`
	`lightChain/core`
	`lightChain/utils`
	`log`
	`os`
	`strconv`
)

// CLI is the command line interface for lightChain.
type CLI struct{}

const usage = `Usage:
	createchain -addr ADDR                  --- Create lightChain and send coinbase reward of genesis block to ADDR
	createwallet                            --- Generate a new wallet (public-private key pair) and save it into file
	listaddr                                --- List all addresses saved in the wallet file
	printchain                              --- Print all the blocks in lightChain
	printtx -b BLOCK_IDX -tx TX_IDX         --- Print the the TX_IDX-th transaction of the BLOCK_IDX-th block
	getblocknum                             --- Print the number of blocks in lightChain
	send -src ADDR1 -dst ADDR2 -amount AMT  --- Send AMT of coins from ADDR1 to ADDR2
	getbalance -addr ADDR                   --- Get the balance of ADDR`

// printUsage prints the usage of the cli.
func (cli *CLI) printUsage() {
	fmt.Println(usage)
}

// validateArgs detects whether the args number is legal.
func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) listAddrs() {
	wallets, err := core.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addrs := wallets.GetAddrs()
	for addrIdx, addr := range addrs {
		fmt.Printf("#%d: %s\n", addrIdx, addr)
	}
	fmt.Println()
}

// printChain prints all blocks of lightChain from the newest to the oldest.
func (cli *CLI) printChain() {
	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	iter := chain.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Timestamp: %d\n", block.TimeStamp)
		fmt.Printf("Previous block's hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		// new a validator with the mined block to examine the nonce
		pow := core.NewPoW(block)
		fmt.Printf("Proof: PoW, Validated: %s\n\n", strconv.FormatBool(pow.Validate()))

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

// printTx prints the required Transaction's details. Note blockIdx is relative to the newest block (from the newest to the oldest).
func (cli *CLI) printTx(blockIdx, txIdx int64) {
	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	tx, err := chain.GetTx(blockIdx, txIdx)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(tx)
}

// getBlockNum returns the number of blocks in lightChain.
func (cli *CLI) getBlockNum() {
	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	fmt.Printf("%d\n\n", chain.GetBlocksNum())
}

// createBlockChain creates lightChain on the whole network.
func (cli *CLI) createBlockChain(addr string) {
	if !core.ValidateAddr(addr) {
		log.Panic("Error: address is not valid")
	}
	chain := core.CreateBlockChain(addr)
	err := chain.Db.Close()
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("Done!\n\n")
}

func (cli *CLI) createWallet() {
	wallets, _ := core.NewWallets()
	addr := wallets.CreateWallet()
	wallets.Save2File()
	fmt.Printf("The newly created address: %s\n\n", addr)

	// save addr to file temporarily for test
	f, err := os.OpenFile("addresses.dat", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	if _, err := f.WriteString(addr + "\n"); err != nil {
		log.Panic(err)
	}
}

// getBalance prints the balance of the wallet whose address is addr.
func (cli *CLI) getBalance(addr string) {
	if !core.ValidateAddr(addr) {
		log.Panic("Error: address is not valid")
	}

	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	balance := 0.0
	pubKeyHash := utils.Base58Decoding([]byte(addr))
	pubKeyHash = pubKeyHash[1: len(pubKeyHash) - 4]
	UTXO := chain.FindUTXO(pubKeyHash)

	for _, output := range UTXO {
		balance += output.Value
	}
	fmt.Printf("The balance of '%s': %f\n\n", addr, balance)
}

// send invoke a transfer transaction from srcAddr to dstAddr with certain amount.
func (cli *CLI) send(srcAddr, dstAddr string, amount float64) {
	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	tx := core.NewUTXOTx(srcAddr, dstAddr, amount, chain)
	// TODO: a block should contain the coinbase tx. Call decrease coinbase reward and new a coinbase tx beforehand!
	chain.MineBlock([]*core.Transaction{tx})
	fmt.Printf("Success!\n\n")
}

func (cli *CLI) Run() {
	cli.validateArgs()

	// define flag set
	createChainSubCmd := flag.NewFlagSet("createchain", flag.ExitOnError)
	addr2GetReward := createChainSubCmd.String("addr", "", "The address to get the coinbase reward of the genesis block")

	createWalletSubCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)

	listAddrSubCmd := flag.NewFlagSet("listaddr", flag.ExitOnError)

	getBlockNumSubCmd := flag.NewFlagSet("getblocknum", flag.ExitOnError)

	printChainSubCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	printTxSubCmd := flag.NewFlagSet("printtx", flag.ExitOnError)
	blockIdx := printTxSubCmd.Int64("b", 0, "The block index since the newest block (starts from 1)")
	txIdx := printTxSubCmd.Int64("tx", 0, "The transaction index (starts from 0)")

	sendSubCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendSubCmd.String("src", "", "Source wallet address")
	sendTo := sendSubCmd.String("dst", "", "Destination wallet address")
	sendAmt := sendSubCmd.Float64("amount", 0.0, "Amount of coins to send")

	getBalanceSubCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	addr2QueryBalance := getBalanceSubCmd.String("addr", "", "The address to query balance")

	// parse flag set
	switch os.Args[1] {
	case "createchain":
		err := createChainSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddr":
		err := listAddrSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getblocknum":
		err := getBlockNumSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printtx":
		err := printTxSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	// use flag set
	if createChainSubCmd.Parsed() {
		if *addr2GetReward == "" {
			createChainSubCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockChain(*addr2GetReward)
	}
	if createWalletSubCmd.Parsed() {
		cli.createWallet()
	}
	if listAddrSubCmd.Parsed() {
		cli.listAddrs()
	}
	if printChainSubCmd.Parsed() {
		cli.printChain()
	}
	if printTxSubCmd.Parsed() {
		if blockIdx == nil || txIdx == nil {
			printTxSubCmd.Usage()
			os.Exit(1)
		}
		cli.printTx(*blockIdx, *txIdx)
	}
	if getBlockNumSubCmd.Parsed() {
		cli.getBlockNum()
	}
	if sendSubCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmt == 0 {
			sendSubCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmt)
	}
	if getBalanceSubCmd.Parsed() {
		if *addr2QueryBalance == "" {
			getBalanceSubCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*addr2QueryBalance)
	}
}
