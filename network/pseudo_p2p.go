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

/*
This file implements a pseudo p2p network. It's pseudo because it is not a really p2p scenario.
In this network, we have:
	- a central node (address is hardcoded as "localhost:23333"): It is the default "seed node" a newly added
		node connected to (In bitcoin, seed nodes are chosen by DNS server). In out simplified case, when a new
		node is added to the blockchain network, it connects to "localhost:23333", and downloads (synchronizes) the
		latest lightChain from it. The central node creates lightChain (it can mine if -mine is set when a tx is launched).

	- a miner node: this node plays the role of miner. It has a transaction pool, where created-but-not-packed
		transactions are collected. When the pool has enough transactions, this node will pack them into
		a block and mine the block through PoW.

	- a wallet node: this node is used to generate wallets and make transactions between those wallets.
		Different from SPV (simplified payment verification) node, this node maintains a full copy of lightChain.

Besides, we use ports to simulate nodes.
*/

package network

import (
	`bytes`
	`encoding/gob`
	`encoding/hex`
	`fmt`
	`io`
	`io/ioutil`
	`lightChain/core`
	`lightChain/utils`
	`log`
	`net`
)

const (
	protocol     = "tcp"             // we use tcp to establish connection between nodes
	nodeVersion  = 1                 // lightChain version
	cmdLen       = 12                // the length of command transferred between nodes
	CentralNode  = "localhost:23333" // the address of the central node
	txNum4Mining = 2                 // if the txPool has more than txNum4Mining txs, the miner node starts packing and mining
)

// KnownNodes plays the role of connection to DNS server, which is responsible for node register and discovery.
var KnownNodes = []string{CentralNode}

// nodeIPAddress plays the role of "current node". It is set at StartNode function.
var nodeIPAddress string

// miningWalletAddress is only set on a miner node (if -miner is set, the node is a miner node).
var miningWalletAddress string

// A local pool for collecting known transactions, used for packing to a new block. Only the miner node can visit & modify this var.
var txPool = make(map[string]core.Transaction)

var blocksInTransit [][]byte

/*
The following defines the request communicated between nodes. In general, request consists of two parts:
command (the first 12 bytes) and content (the left bytes).
	- command: version, addr, inv, getblocks, getdata, block, tx
	- content: sVersion, sAddr, sInventory, sGetBlocks, sGetData, sBlock, sTx
All the contents are defined as structs as follows.
*/

// sVersion is used to find a newer blockchain copy from the server node for the client node whose address is SenderAddr.
type sVersion struct {
	Version    int    // current version of client's lightChain
	Height     int    // current height (#blocks) of client's lightChain
	SenderAddr string // the address of client node who sends this
}

// sAddr is used to make the addresses in AddrList discoverable to all blockchain nodes.
type sAddr struct {
	AddrList []string
}

// sInventory is used to show the client node whose address is SenderAddr what the server node have.
type sInventory struct {
	SenderAddr string   // the address of client node who sends this
	Kind       string   // "block" (core.Block) or "tx" (core.Transaction)
	Items      [][]byte // detailed inventory items (the hashes of all "block" or all "tx")
}

// sGetBlocks is used to construct a request from the client node whose address is SenderAddr to the server node.
// The request asks the server to show what blocks it have.
type sGetBlocks struct {
	SenderAddr string // the address of client node who sends this
}

// sGetData is used to construct a request from the client node whose address is SenderAddr to the server node.
// The request asks the server to show the block or transaction whose identity is Id.
type sGetData struct {
	SenderAddr string // the address of client node who sends this
	Kind       string // "block" (core.Block) or "tx" (core.Transaction)
	Id         []byte
}

// sBlock is used to send block from the server node to the client node whose address is SenderAddr.
type sBlock struct {
	SenderAddr string // the address of client node who sends this
	Block      []byte
}

// sTx is used to send transaction from the server node to the client node whose address is SenderAddr.
type sTx struct {
	SenderAddr  string // the address of client node who sends this
	Transaction []byte
}

/* The following code defines the server-side functions (starts with "handle") for each p2p node. */

// StartNode starts a new node as a tcp server.
// When starting, this node firstly requests a full copy of current version of lightChain from the central node.
// Then, the node will listen a port, waits for connection, and processes the connection. The new node' address is
// generated with nodeId. minerAddr gives the address of wallet to receive the coinbase and mining reward.
func StartNode(nodeId, minerAddr string) {
	nodeIPAddress = fmt.Sprintf("localhost:%s", nodeId)
	miningWalletAddress = minerAddr

	// open for connection
	listener, err := net.Listen(protocol, nodeIPAddress)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	// request and make a local copy of current lightChain from the whole network (actually the central node in our case)
	chain := core.NewBlockChain(nodeId)
	if nodeIPAddress != CentralNode {
		// if this node is not the central node, it should query the central node whether the blockchain it copied is outdated
		sendVersion(CentralNode, chain)
	}

	// as a server, wait, establish and handle each connection from clients
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConn(conn, chain)
	}
}

// handleConn reads message from conn, extracts command from the message and call corresponding function
// to process the command. Note that chain is from the server node.
func handleConn(conn net.Conn, chain *core.BlockChain) {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	cmd := bytes2Cmd(request[:cmdLen])
	fmt.Printf("Recevie command: %s\n", cmd)

	switch cmd {
	case "version":
		handleVersion(request, chain)
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, chain)
	case "inv":
		handleInv(request)
	case "getblocks":
		handleGetBlocks(request, chain)
	case "getdata":
		handleGetData(request, chain)
	case "tx":
		handleTx(request, chain)
	default:
		fmt.Println("Unknown command!")
	}

	err = conn.Close()
	if err != nil {
		log.Println(err)
	}
}

// handleVersion handles the "version" request received from the client. If the server has a highest lightChain (which
// means it has a newer lightChain copy), it will response to the client with sendVersion message. Otherwise, the server
// will response to the client with sendGetBlocks message. Note that chain is from the server node.
func handleVersion(request []byte, chain *core.BlockChain) {
	// extract the sVersion instance from the request
	var buf bytes.Buffer
	var payload sVersion

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	// according to the height of local (server) chain and client chain, response with different message
	localHeight := chain.GetChainHeight()
	externalHeight := payload.Height
	if localHeight < externalHeight {
		sendGetBlocks(payload.SenderAddr)
	} else if localHeight > externalHeight {
		sendVersion(payload.SenderAddr, chain)
	}

	// if the client's address is not known beforehand, make it discoverable for all blockchain nodes
	// this is actually a simulation of the DNS server's operation
	senderAddrIsKnown := false
	for _, node := range KnownNodes {
		if node == payload.SenderAddr {
			senderAddrIsKnown = true
			break
		}
	}
	if !senderAddrIsKnown {
		KnownNodes = append(KnownNodes, payload.SenderAddr)
	}
}

// TODO: this func may not used. The content of this func is included in handleVersion.
func handleAddr(request []byte) {
	var buf bytes.Buffer
	var payload sAddr

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	KnownNodes = append(KnownNodes, payload.AddrList...)
	fmt.Printf("#KnownNodes: %d\n", len(KnownNodes))
	requestBlocks()
}

// requestBlocks sends nodeIPAddress to all known nodes.
func requestBlocks() {
	for _, node := range KnownNodes {
		sendGetBlocks(node)
	}
}

// handleInv handles the received sInventory instance from the client. If the inventory is block, this server will save
// all received blocks' hash in blocksInTransit and call sendGetData to the client to get a block.
// If the inventory is transaction and this server does not have this transaction, it will call sendGetData to the client
// to get a tx.
func handleInv(request []byte) {
	// extract the inventory instance from request
	var buf bytes.Buffer
	var payload sInventory

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Receive inventory with %d %ss\n", len(payload.Items), payload.Kind)

	if payload.Kind == "block" {
		blocksInTransit = payload.Items
		blockHash := payload.Items[0]
		sendGetData(payload.SenderAddr, "block", blockHash)

		// reset blocksInTransit
		var newInTransit [][]byte
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Kind == "tx" {
		txId := payload.Items[0]
		if txPool[hex.EncodeToString(txId)].Id == nil {
			sendGetData(payload.SenderAddr, "tx", txId)
		}
	}
}

// handleGetBlocks handles the "getblocks" request received from the client. The server node sends all blocks' hash
// it have to the client node. Note that chain is from the server node.
func handleGetBlocks(request []byte, chain *core.BlockChain) {
	// extract sGetBlocks instance from the request
	var buf bytes.Buffer
	var payload sGetBlocks

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	// send all blocks' hash from the server node to the client node
	blockHashes := chain.GetAllBlocksHashes()
	sendInv(payload.SenderAddr, "block", blockHashes)
}

// handleGetData handles the "getdata" request received from the client. If the client requires block, this server sends
// the specific block to the client by calling sendBlock. If the client requires tx, this server sends the specific tx
// to the client by calling SendTx. Note that chain is from the server node.
// TODO: we do not check whether the server node has the block or the tx. Fix this!
func handleGetData(request []byte, chain *core.BlockChain) {
	var buf bytes.Buffer
	var payload sGetData

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Kind == "block" {
		block, err := chain.GetBlock(payload.Id)
		if err != nil {
			log.Panic(err)
		}

		sendBlock(payload.SenderAddr, block)
	}

	if payload.Kind == "tx" {
		txId := hex.EncodeToString(payload.Id)
		tx := txPool[txId]

		SendTx(payload.SenderAddr, &tx)
	}
}

// handleBlock handles the received block from the client node. Note that chain is from the server node.
func handleBlock(request []byte, chain *core.BlockChain) {
	var buf bytes.Buffer
	var payload sBlock

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	block := core.DeserializeBlock(payload.Block)
	fmt.Printf("Receive a new block!\n")
	chain.AddBlock(block)
	fmt.Printf("Added this block successfully! Its hash: %x\n", block.Hash)

	// if this server finds that it has more blocks to download, just send request the same client for next block
	// until all blocks are downloaded
	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.SenderAddr, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	} else {
		utxoSet := core.UTXOSet{BlockChain: chain}
		utxoSet.Rebuild()
	}
}

// handleTx handles the received tx from the client node. Note that chain is from the server node.
func handleTx(request []byte, chain *core.BlockChain) {
	// extract the tx from the client and put it into txPool
	var buf bytes.Buffer
	var payload sTx

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	tx := core.DeserializeTx(payload.Transaction)
	txPool[hex.EncodeToString(tx.Id)] = tx

	// CentralNode does not mining. Just broadcast this tx to every known nodes
	if nodeIPAddress == CentralNode {
		for _, node := range KnownNodes {
			if node != nodeIPAddress && node != payload.SenderAddr {
				sendInv(node, "tx", [][]byte{tx.Id})
			}
		}
	} else {
		if len(txPool) >= txNum4Mining && len(miningWalletAddress) > 0 {
		MineTxs:
			var verifiedTxs []*core.Transaction
			for txIdInPool := range txPool {
				txInPool := txPool[txIdInPool]
				if chain.VerifyTx(&txInPool) {
					verifiedTxs = append(verifiedTxs, &txInPool)
				}
			}

			if len(verifiedTxs) == 0 {
				fmt.Printf("No transaction is valid. Waiting for new transactions...\n")
				return
			}

			coinbaseTx := core.NewCoinbaseTx(miningWalletAddress, "", chain.CoinbaseReward)
			// verifiedTxs = append([]*core.Transaction{coinbaseTx}, verifiedTxs...)
			verifiedTxs = append(verifiedTxs, coinbaseTx)

			// pack into a new block
			newBlock := chain.MineBlock(verifiedTxs)
			utxoSet := core.UTXOSet{BlockChain: chain}
			utxoSet.Rebuild()
			fmt.Printf("New block is successfully mined!\n")

			// remove the already packed transactions from pool
			for _, tx := range verifiedTxs {
				delete(txPool, hex.EncodeToString(tx.Id))
			}

			// broadcast this newly mined block to all known nodes
			for _, node := range KnownNodes {
				if node != nodeIPAddress {
					sendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}

			if len(txPool) > 0 {
				goto MineTxs
			}
		}
	}
}

/* The following code defines the client-side functions (starts with "send") for each p2p node. */

// sendBlock sends block b to dstAddr.
func sendBlock(dstAddr string, b *core.Block) {
	block := sBlock{
		SenderAddr: nodeIPAddress,
		Block:      b.SerializeBlock(),
	}

	payload := utils.GobEncode(block)
	request := append(cmd2Bytes("block"), payload...)

	send(dstAddr, request)
}

// sendInv sends a sInventory instance constructed by nodeIPAddress, kind, and items to dstAddr.
func sendInv(dstAddr, kind string, items [][]byte) {
	inv := sInventory{
		SenderAddr: nodeIPAddress,
		Kind:       kind,
		Items:      items,
	}

	payload := utils.GobEncode(inv)
	request := append(cmd2Bytes("inv"), payload...)

	send(dstAddr, request)
}

// SendTx sends a sTx instance constructed by nodeIPAddress and transaction to dstAddr.
func SendTx(dstAddr string, transaction *core.Transaction) {
	tx := sTx{
		SenderAddr:  nodeIPAddress,
		Transaction: transaction.SerializeTx(),
	}

	payload := utils.GobEncode(tx)
	request := append(cmd2Bytes("tx"), payload...)

	send(dstAddr, request)
}

// sendVersion sends a sVersion instance constructed by chain, nodeVersion, and nodeIPAddress to dstAddr.
func sendVersion(dstAddr string, chain *core.BlockChain) {
	ver := sVersion{
		Version:    nodeVersion,
		Height:     chain.GetChainHeight(),
		SenderAddr: nodeIPAddress,
	}

	payload := utils.GobEncode(ver)
	request := append(cmd2Bytes("version"), payload...)

	send(dstAddr, request)
}

// sendGetBlocks sends nodeIPAddress to dstAddr.
func sendGetBlocks(dstAddr string) {
	getBlocks := sGetBlocks{
		SenderAddr: nodeIPAddress,
	}

	payload := utils.GobEncode(getBlocks)
	request := append(cmd2Bytes("getblocks"), payload...)

	send(dstAddr, request)
}

// sendGetData sends a sGetData instance to dstAddr.
func sendGetData(dstAddr, kind string, id []byte) {
	getData := sGetData{
		SenderAddr: nodeIPAddress,
		Kind:       kind,
		Id:         id,
	}

	payload := utils.GobEncode(getData)
	request := append(cmd2Bytes("getdata"), payload...)

	send(dstAddr, request)
}

// send sends data to dstAddr through TCP.
func send(dstAddr string, data []byte) {
	// establish connection to dstAddr
	conn, err := net.Dial(protocol, dstAddr)
	if err != nil {
		// if dstAddr is not reachable, remove it from KnownNodes
		fmt.Printf("%s is not available\n", dstAddr)
		var updatedNodes []string
		for _, node := range KnownNodes {
			if node != dstAddr {
				updatedNodes = append(updatedNodes, node)
			}
		}
		KnownNodes = updatedNodes
		return
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	// copy data to the connection
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

/* The following defines several auxiliary functions. */

// cmd2Bytes converts the cmd string into a byte slice.
func cmd2Bytes(cmd string) []byte {
	var byteChars [cmdLen]byte
	for idx, ch := range cmd {
		byteChars[idx] = byte(ch)
	}
	return byteChars[:]
}

// bytes2Cmd converts the byte slice back to a cmd string.
func bytes2Cmd(byteChars []byte) string {
	var cmd []byte
	for _, b := range byteChars {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}
	return fmt.Sprintf("%s", cmd)
}
