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

package core

import (
	`bytes`
	`encoding/gob`
	`log`
	`time`
)

// Block consists of the block header and the block body.
type Block struct {
	// block header
	TimeStamp     int64
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	Height        int

	// block body (a collection of transactions)
	Transactions []*Transaction
}

// NewBlock generates a new block with slice of Transaction and previous block hash.
// Miners need to run the Run function while validators need to run the Validate function.
func NewBlock(txs []*Transaction, prevBlockHash []byte, height int) *Block {
	var block = &Block{
		TimeStamp:     time.Now().Unix(),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0,
		Height:        height,
		Transactions:  txs}

	pow := NewPoW(block)
	nonce, hash := pow.Run()
	block.Hash = hash
	block.Nonce = nonce

	return block
}

// NewGenesisBlock generates the very first block of the chain with only one Transaction,
// i.e. the coinbase transaction.
func NewGenesisBlock(coinbaseTx *Transaction) *Block {
	return NewBlock([]*Transaction{coinbaseTx}, []byte{}, 0)
}

// SerializeBlock converts the block's content into a serialized byte slice.
func (block *Block) SerializeBlock() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(block)
	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}

// DeserializeBlock returns a block pointer decoded from the serialized data encodedData.
func DeserializeBlock(encodedData []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(encodedData))

	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

// HashingAllTxs returns the hashing result of all the transactions in block.
// The hashing is based on the Merkle tree structure.
func (block *Block) HashingAllTxs() []byte {
	var serializedTxData [][]byte
	for _, tx := range block.Transactions {
		serializedTxData = append(serializedTxData, tx.SerializeTx())
	}
	merkleTree := NewMerkleTree(serializedTxData)
	return merkleTree.RootNode.Data
}
