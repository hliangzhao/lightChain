package core

import (
	`fmt`
	`github.com/boltdb/bolt`
	`log`
)

const dbFile = "lightChain.db"
const blocksBucket = "lightChain"

type BlockChain struct {
	Tip []byte          // the last block' hash
	Db  *bolt.DB        // the pointer-to-db where the chain stored
}

// NewBlockChain generates the blockchain with the genesis block.
// The chain data will be saved into a k-v db when the chain is created.
func NewBlockChain() *BlockChain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			// no lightChain exists in db, just create one
			if bucket == nil {
				fmt.Println("No existing lightChain found. Creating a new one...")

				// create a bucket
				b, err := tx.CreateBucket([]byte(blocksBucket))
				if err != nil {
					log.Panic(err)
				}

				// create the genesis block and put the key-value pair
				// (block hash: serialized block data) into the bucket
				genesis := NewGenesisBlock()
				err = b.Put(genesis.Hash, genesis.Serialize())
				if err != nil {
					log.Panic(err)
				}

				// the key []byte("l") always points to the last block' hash
				err = b.Put([]byte("l"), genesis.Hash)
				if err != nil {
					log.Panic(err)
				}
				tip = genesis.Hash
			} else { // or get the hash of the last block
				tip = bucket.Get([]byte("l"))
			}

			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return &BlockChain{tip, db}
}

// AppendBlock appends a new block to the blockchain.
// Each new block is mined through PoW and will be stored into the db.
func (chain *BlockChain) AppendBlock(data string) {
	// get the last block' hash for the generation of new block
	var lastHash []byte
	err := chain.Db.View(
		func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			lastHash = b.Get([]byte("l"))
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	// store the new block into db
	newBlock := NewBlock(data, lastHash)
	err = chain.Db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))

			err := bucket.Put(newBlock.Hash, newBlock.Serialize())
			if err != nil {
				log.Panic(err)
			}

			// overwrite the value for key []byte("l")
			err = bucket.Put([]byte("l"), newBlock.Hash)
			if err != nil {
				log.Panic(err)
			}

			chain.Tip = newBlock.Hash
			return nil
		})
}

// IterOnChain is a iterator over the blockchain.
type IterOnChain struct {
	curBlockHash []byte
	db           *bolt.DB
}

// Iterator returns a pointer to IterOnChain.
func (chain *BlockChain) Iterator() *IterOnChain {
	return &IterOnChain{chain.Tip, chain.Db}
}

// Next returns the current block's pointer based on IterOnChain.
// Note that the iteration direction is from the newest block to the oldest block.
func (iter *IterOnChain) Next() *Block {
	var block *Block
	err := iter.db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			encodedBlock := bucket.Get(iter.curBlockHash)
			block = Deserialize(encodedBlock)
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	iter.curBlockHash = block.PrevBlockHash
	return block
}
