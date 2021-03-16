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
	`lightChain/network`
	`lightChain/utils`
	`log`
	`os`
	`strconv`
)

// CLI is the command line interface for lightChain.
type CLI struct{}

// the "addr" below means wallet address!

const usage = `Usage:
	createchain -addr ADDR                          --- Create lightChain and send coinbase reward of genesis block to ADDR
	createwallet                                      --- Generate a new wallet (public-private key pair) and save it into file
	listaddr                                          --- List all addresses saved in local wallet file
	printchain                                        --- Print all the blocks in local lightChain
	printtx -b BLOCK_IDX -tx TX_IDX                   --- Print the the TX_IDX-th transaction of the BLOCK_IDX-th block of local lightChain
	printalltxs                                       --- Print all transactions in every block of local lightChain
	getblocknum                                       --- Print the number of blocks in local lightChain
	send -src ADDR1 -dst ADDR2 -amount AMT -mine  --- Send AMT of coins from ADDR1 to ADDR2, mine on the same node if -mine is set
	getbalance -addr ADDR                           --- Get the balance of ADDR
	rebuildutxo                                       --- Rebuild the UTXO
	startnode -miner ADDR                           --- Add a new node to lightChain network with Node Id specified in NODE_ID environment variable. Enable mining if -miner set`

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

func (cli *CLI) listAddrs(nodeId string) {
	wallets, err := core.NewWallets(nodeId)
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
func (cli *CLI) printChain(nodeId string) {
	chain := core.NewBlockChain(nodeId)
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
func (cli *CLI) printTx(nodeId string, blockIdx, txIdx int) {
	chain := core.NewBlockChain(nodeId)
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	tx, err := chain.GetTx(blockIdx+1, txIdx)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(tx)
}

// printAllTxs prints all Transaction's details for all blocks in current lightChain. The print is form the most
// recent block to the genesis block.
func (cli *CLI) printAllTxs(nodeId string) {
	chain := core.NewBlockChain(nodeId)
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	blockIdx := chain.GetBlocksNum() - 1
	iter := chain.Iterator()
	for {
		fmt.Printf("== Block #%d ==", blockIdx)
		block := iter.Next()
		for txIdx := range block.Transactions {
			cli.printTx(nodeId, blockIdx, txIdx)
		}
		blockIdx--

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

// getBlockNum returns #blocks in lightChain.
func (cli *CLI) getBlockNum(nodeId string) {
	chain := core.NewBlockChain(nodeId)
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	fmt.Printf("%d\n\n", chain.GetBlocksNum())
}

// createBlockChain creates lightChain on the whole network.
func (cli *CLI) createBlockChain(addr, nodeId string) {
	if !core.ValidateAddr(addr) {
		log.Panic("Error: address is not valid")
	}
	chain := core.CreateBlockChain(addr, nodeId)
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	// rebuild UTXO
	utxoSet := core.UTXOSet{BlockChain: chain}
	utxoSet.Rebuild()
	fmt.Printf("Done!\n\n")
}

func (cli *CLI) createWallet(nodeId string) {
	wallets, _ := core.NewWallets(nodeId)
	addr := wallets.CreateWallet()
	wallets.Save2File(nodeId)
	fmt.Printf("The newly created address: %s\n\n", addr)

	// save addr to local file temporarily (this is for run_example.sh)
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

// send invoke a transfer transaction from srcAddr to dstAddr with certain amount. If mineNow is true, the sender node
// will mine this block directly. Otherwise, the tx will be broadcast
func (cli *CLI) send(srcAddr, dstAddr string, amount float64, nodeId string, mineNow bool) {
	if !core.ValidateAddr(srcAddr) {
		log.Panic("Error: srcAddr is not valid")
	}
	if !core.ValidateAddr(dstAddr) {
		log.Panic("Error: dstAddr is not valid")
	}

	chain := core.NewBlockChain(nodeId)
	utxoSet := core.UTXOSet{BlockChain: chain}
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	wallets, err := core.NewWallets(nodeId)
	if err != nil {
		log.Panic(err)
	}

	senderWallet, err := wallets.GetWallet(srcAddr)
	if err != nil {
		log.Panic(err)
	}
	tx := core.NewUTXOTx(&senderWallet, dstAddr, amount, &utxoSet)

	if mineNow {
		coinbaseTx := core.NewCoinbaseTx(srcAddr, "")
		txs := []*core.Transaction{coinbaseTx, tx}

		newBlock := chain.MineBlock(txs)
		utxoSet.Update(newBlock)
	} else {
		network.SendTx(network.CentralNode, tx)
	}

	fmt.Printf("Success!\n\n")
}

// getBalance prints the balance of the wallet whose address is addr.
func (cli *CLI) getBalance(addr, nodeId string) {
	if !core.ValidateAddr(addr) {
		log.Panic("Error: address is not valid")
	}

	chain := core.NewBlockChain(nodeId)
	utxoSet := core.UTXOSet{BlockChain: chain}
	defer func() {
		err := chain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	balance := 0.0
	pubKeyHash := utils.Base58Decoding([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	utxo := utxoSet.FindUTXO(pubKeyHash)

	for _, output := range utxo {
		balance += output.Value
	}
	fmt.Printf("The balance of '%s': %f\n\n", addr, balance)
}

// rebuildUTXO rebuilds the UTXO incrementally when lightChain changes.
func (cli *CLI) rebuildUTXO(nodeId string) {
	chain := core.NewBlockChain(nodeId)
	utxoSet := core.UTXOSet{BlockChain: chain}
	utxoSet.Rebuild()

	fmt.Printf("Done! %d transactions found in UTXO set.\n\n", utxoSet.CountTxs())
}

func (cli *CLI) startNode(nodeId, nodeMinerAddr string) {
	fmt.Printf("Starting node %s...\n", nodeId)
	if len(nodeMinerAddr) > 0 {
		if core.ValidateAddr(nodeMinerAddr) {
			fmt.Printf("Mining is on! The address to receive rewards: %s\n", nodeMinerAddr)
		} else {
			log.Panic("Miner address is illegal!")
		}
	}
	network.StartNode(nodeId, nodeMinerAddr)
}

func (cli *CLI) Run() {
	cli.validateArgs()

	nodeId := os.Getenv("NODE_ID")
	if nodeId == "" {
		fmt.Printf("NODE_ID is not set.")
		os.Exit(1)
	}

	// define flag set
	createChainSubCmd := flag.NewFlagSet("createchain", flag.ExitOnError)
	addr2GetReward := createChainSubCmd.String("addr", "", "The wallet address to get the coinbase reward of the genesis block")

	createWalletSubCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)

	listAddrSubCmd := flag.NewFlagSet("listaddr", flag.ExitOnError)

	getBlockNumSubCmd := flag.NewFlagSet("getblocknum", flag.ExitOnError)

	printChainSubCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	printTxSubCmd := flag.NewFlagSet("printtx", flag.ExitOnError)
	blockIdx := printTxSubCmd.Int("b", 0, "The block index since the newest block (starts from 0)")
	txIdx := printTxSubCmd.Int("tx", 0, "The transaction index (starts from 0)")

	printAllTxsSubCmd := flag.NewFlagSet("printalltxs", flag.ExitOnError)

	sendSubCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendSubCmd.String("src", "", "Source wallet address")
	sendTo := sendSubCmd.String("dst", "", "Destination wallet address")
	sendAmt := sendSubCmd.Float64("amount", 0.0, "Amount of coins to send")
	sendMine := sendSubCmd.Bool("mine", false, "Mine immediately on the same node")

	getBalanceSubCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	addr2QueryBalance := getBalanceSubCmd.String("addr", "", "The address to query balance")

	rebuildUTXOSubCmd := flag.NewFlagSet("rebuildutxo", flag.ExitOnError)

	startNodeSubCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	nodeMinerAddr := startNodeSubCmd.String("miner", "", "Enable mining and send reward to ADDR")

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
	case "printalltxs":
		err := printAllTxsSubCmd.Parse(os.Args[2:])
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
	case "rebuildutxo":
		err := rebuildUTXOSubCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "startnode":
		err := startNodeSubCmd.Parse(os.Args[2:])
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
		cli.createBlockChain(*addr2GetReward, nodeId)
	}
	if createWalletSubCmd.Parsed() {
		cli.createWallet(nodeId)
	}
	if listAddrSubCmd.Parsed() {
		cli.listAddrs(nodeId)
	}
	if printChainSubCmd.Parsed() {
		cli.printChain(nodeId)
	}
	if printTxSubCmd.Parsed() {
		if blockIdx == nil || txIdx == nil {
			printTxSubCmd.Usage()
			os.Exit(1)
		}
		cli.printTx(nodeId, *blockIdx, *txIdx)
	}
	if printAllTxsSubCmd.Parsed() {
		cli.printAllTxs(nodeId)
	}
	if getBlockNumSubCmd.Parsed() {
		cli.getBlockNum(nodeId)
	}
	if sendSubCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmt <= 0 {
			sendSubCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmt, nodeId, *sendMine)
	}
	if getBalanceSubCmd.Parsed() {
		if *addr2QueryBalance == "" {
			getBalanceSubCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*addr2QueryBalance, nodeId)
	}
	if rebuildUTXOSubCmd.Parsed() {
		cli.rebuildUTXO(nodeId)
	}
	if startNodeSubCmd.Parsed() {
		cli.startNode(nodeId, *nodeMinerAddr)
	}
}
