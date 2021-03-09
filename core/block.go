package core

import (
	`bytes`
	`crypto/sha256`
	`strconv`
	`time`
)

type Block struct {
	// block header
	TimeStamp     int64
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	// block body
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
// Miner needs to run the Mine() function while validator needs to run the validate() function.
func NewBlock(data string, prevBlockHash []byte) *Block {
	var block = &Block{
		TimeStamp:     time.Now().Unix(),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Nonce:         0,
		Data:          []byte(data)}
	proof := NewPoW(block)
	nonce, hash := proof.Mine()

	block.Hash = hash
	block.Nonce = nonce
	// block.SetHash()

	return block
}

// NewGenesisBlock: generate the very first block of the chain.
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
