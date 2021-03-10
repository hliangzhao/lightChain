package core

import (
	`bytes`
	`crypto/sha256`
	`encoding/gob`
	`log`
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

// SetHash sets the hash value of current block by SHA256(PrevBlockHash + TimeStamp + Data).
// This function is temporarily used in early development stage, it is replaced by PoW.
func (block *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(block.TimeStamp, 10))
	headers := bytes.Join([][]byte{block.PrevBlockHash, block.Data, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	block.Hash = hash[:]
}

// NewBlock generates a new block with data and previous block hash.
// Miner needs to run the Mine() function while validator needs to run the Validate function.
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

	return block
}

// NewGenesisBlock generates the very first block of the chain.
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

// Serialize converts the block into a serialized byte slice.
func (block *Block) Serialize() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(block)
	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}

// Deserialize returns a block pointer decoded from the serialized data encodedData.
func Deserialize(encodedData []byte) *Block {
	var b Block
	decoder := gob.NewDecoder(bytes.NewReader(encodedData))

	err := decoder.Decode(&b)
	if err != nil {
		log.Panic(err)
	}

	return &b
}