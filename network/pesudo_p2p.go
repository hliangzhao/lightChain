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

package network

import (
	`bytes`
	`encoding/gob`
	`fmt`
	`io`
	`lightChain/core`
	`lightChain/utils`
	`log`
	`net`
)

const (
	protocol    = "tcp"
	nodeVersion = 1
	cmdLen      = 12
)

var (
	nodeAddress     string          // used as src address
	miningAddress   string          //
	blocksInTransit [][]byte{}      //
)

var knownNodes = []string{"localhost:3000"}
var memPool = make(map[string]core.Transaction)

/* In the following we define several structs (all of them start with 's'), which are used to transit data in network. */

type sAddr struct {
	AddrList []string
}

// sendAddr sends nodeAddress and all known nodes in knownNodes to dstAddr.
func sendAddr(dstAddr string) {
	addrs := sAddr{
		AddrList: knownNodes,
	}
	addrs.AddrList = append(addrs.AddrList, nodeAddress)

	payload := utils.GobEncode(addrs)
	request := append(cmd2Bytes("addr"), payload...)

	send(dstAddr, request)
}

func handleAddr(request []byte) {
	var buf bytes.Buffer
	var payload sAddr

	buf.Write(request[cmdLen:])
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("#knownNodes: %d\n", len(knownNodes))
	requestBlocks()
}

type sBlock struct {
	AddrFrom string
	Block    []byte
}

// sendBlock sends block b to dstAddr.
func sendBlock(dstAddr string, b *core.Block) {
	block := sBlock{
		AddrFrom: nodeAddress,
		Block:    b.SerializeBlock(),
	}

	payload := utils.GobEncode(block)
	request := append(cmd2Bytes("block"), payload...)

	send(dstAddr, request)
}

// TODO: we are here!
func handleBlock(request []byte, chain *core.BlockChain) {
}

// sInventory is used to show other nodes what information (blocks & transactions) this node has.
type sInventory struct {
	AddrFrom string
	Kind     string
	Items    [][]byte
}

// sendInv sends a sInventory instance constructed by nodeAddress, kind, and items to dstAddr.
func sendInv(dstAddr, kind string, items [][]byte) {
	inv := sInventory{
		AddrFrom: nodeAddress,
		Kind:     kind,
		Items:    items,
	}

	payload := utils.GobEncode(inv)
	request := append(cmd2Bytes("inv"), payload...)

	send(dstAddr, request)
}

type sTx struct {
	AddFrom     string
	Transaction []byte
}

// sendTx sends a sTx instance constructed by nodeAddress and transaction to dstAddr.
func sendTx(dstAddr string, transaction *core.Transaction) {
	tx := sTx{
		AddFrom:     nodeAddress,
		Transaction: transaction.SerializeTx(),
	}

	payload := utils.GobEncode(tx)
	request := append(cmd2Bytes("tx"), payload...)

	send(dstAddr, request)
}

type sVersion struct {
	Version    int
	BestHeight int
	AddFrom    string
}

// sendVersion sends a sVersion instance constructed by chain, nodeVersion, and nodeAddress to dstAddr.
func sendVersion(dstAddr string, chain *core.BlockChain) {
	ver := sVersion{
		Version:    nodeVersion,
		BestHeight: chain.GetChainHeight(),
		AddFrom:    nodeAddress,
	}

	payload := utils.GobEncode(ver)
	request := append(cmd2Bytes("version"), payload...)

	send(dstAddr, request)
}

type sGetBlocks struct {
	AddrFrom string
}

// TODO: replaced by broadcast.
// sendGetBlocks sends nodeAddress to dstAddr.
func sendGetBlocks(dstAddr string) {
	getBlocks := sGetBlocks{
		AddrFrom: nodeAddress,
	}

	payload := utils.GobEncode(getBlocks)
	request := append(cmd2Bytes("version"), payload...)

	send(dstAddr, request)
}

// requestBlocks sends nodeAddress to all known nodes (stored in knownNodes).
func requestBlocks() {
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

type sGetData struct {
	AddrFrom string
	Kind     string
	Id       []byte
}

func sendGetData(dstAddr, kind string, id []byte) {
	getData := sGetData{
		AddrFrom: nodeAddress,
		Kind:     kind,
		Id:       id,
	}

	payload := utils.GobEncode(getData)
	request := append(cmd2Bytes("version"), payload...)

	send(dstAddr, request)
}

// TODO: differentiate node and two kinds of address!

// send sends data to dstAddr through TCP.
func send(dstAddr string, data []byte) {
	// establish connection to dstAddr
	conn, err := net.Dial(protocol, dstAddr)
	if err != nil {
		fmt.Printf("%s is not available\n", dstAddr)

		// remove dstAddr from knownNodes
		var updatedNodes []string
		for _, node := range knownNodes {
			if node != dstAddr {
				updatedNodes = append(updatedNodes, node)
			}
		}
		knownNodes = updatedNodes
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

// cmd2Bytes converts the cmd into a byte slice.
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

func extractCmd(request []byte) []byte {
	return request[:cmdLen]
}

// nodeIsKnown checks whether addr is known to the network (whether in knownNodes).
func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}
	return false
}
