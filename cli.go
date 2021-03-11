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
	`log`
	`os`
	`strconv`
)

// CLI is the command line interface for lightChain.
type CLI struct {}

const usage = `Usage:
	createchain -addr ADDR                   --- Create lightChain and send coinbase reward of the genesis block to ADDR
	printchain                               --- Print all the blocks in lightChain
	send -src ADDR1 -dst ADDR2 -amount AMT   --- Send AMT of coins from ADDR1 to ADDR2
	getbalance -addr ADDR                    --- Get the balance of ADDR`

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

// TODO: implement a function to print all the transactions an addr involved.

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
		proof := core.NewPoW(block)
		fmt.Printf("Proof: Pow, Validated: %s\n\n", strconv.FormatBool(proof.Validate()))

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

// createBlockChain calls CreateBlockChain() to create lightChain.
func (cli *CLI) createBlockChain(addr string) {
	chain := core.CreateBlockChain(addr)
	err := chain.Db.Close()
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("Done!\n\n")
}

// getBalance prints the balance of the node addr.
func (cli *CLI) getBalance(addr string) {
	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	balance := 0
	UTXO := chain.FindUTXO(addr)
	for _, output := range UTXO {
		balance += output.Value
	}
	fmt.Printf("The balance of '%s': %d\n\n", addr, balance)
}

// send invoke a transfer transaction from srcAddr to dstAddr with certain amount.
func (cli *CLI) send(srcAddr, dstAddr string, amount int) {
	chain := core.NewBlockChain()
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	tx := core.NewUTXOTx(srcAddr, dstAddr, amount, chain)
	// TODO: a block can contain more than one tx.
	chain.MineBlock([]*core.Transaction{tx})
	fmt.Printf("Success!\n\n")
}

func (cli *CLI) Run() {
	cli.validateArgs()

	// define flag set
	createChainSubCmd := flag.NewFlagSet("createchain", flag.ExitOnError)
	addr2GetReward := createChainSubCmd.String("addr", "", "The address to get the coinbase reward of the genesis block")

	printChainSubCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	sendSubCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendSubCmd.String("src", "", "Source wallet address")
	sendTo := sendSubCmd.String("dst", "", "Destination wallet address")
	sendAmt := sendSubCmd.Int("amount", 0, "Amount of coins to send")

	getBalanceSubCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	addr2QueryBalance := getBalanceSubCmd.String("addr", "", "The address to query balance")

	// parse flag set
	switch os.Args[1] {
	case "createchain":
		err := createChainSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainSubCmd.Parse(os.Args[2:])
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
	if printChainSubCmd.Parsed() {
		cli.printChain()
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
