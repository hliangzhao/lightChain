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
type CLI struct {
	chain *core.BlockChain
}

const usage = `Usage:
	addblock -data BLOCK_DATA      add a block to lightChain
	printchain                     print all the blocks in lightChain`

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

// appendBlock is a command to add new block to lightChain.
func (cli *CLI) appendBlock(data string) {
	cli.chain.AppendBlock(data)
	fmt.Println("Successfully append a new block to lightChain")
}

// printChain prints all blocks of lightChain from the newest to the oldest.
func (cli *CLI) printChain() {
	iter := cli.chain.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Timestamp: %d\n", block.TimeStamp)
		fmt.Printf("Previous block's hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)

		// new a validator with the mined block to examine the nonce
		proof := core.NewPoW(block)
		fmt.Printf("Proof: Pow, Validated: %s\n", strconv.FormatBool(proof.Validate()))

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	// define flag set
	addBlockSubCmd := flag.NewFlagSet("addblock", flag.ExitOnError)
	addBlockData := addBlockSubCmd.String("data", "", "Fill in block data")
	printChainSubCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	// parse flag set
	switch os.Args[1] {
	case "addblock":
		err := addBlockSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	// use flag set
	if addBlockSubCmd.Parsed() {
		if *addBlockData == "" {
			addBlockSubCmd.Usage()
		}
		cli.appendBlock(*addBlockData)
	}
	if printChainSubCmd.Parsed() {
		cli.printChain()
	}
}
