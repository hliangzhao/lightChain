package core

import (
	`bytes`
	`crypto/sha256`
	`strconv`
	`time`
)

type Block struct {
	TimeStamp int64
	PrevBlockHash []byte
	Hash []byte
	Data []byte
}

// SetHash: set the hash value of current block by SHA256(PrevBlockHash + TimeStamp + Data).
func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.TimeStamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.Data, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

// NewBlock generates a new block with data and previous block hash.
func NewBlock(data string, prevBlockHash []byte) *Block {
	var block = &Block{
		TimeStamp:     time.Now().Unix(),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Data:          []byte(data)}
	block.SetHash()
	return block
}

// NewGenesisBlock: generate the very first block of the chain.
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}