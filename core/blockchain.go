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
	`crypto/ecdsa`
	`encoding/hex`
	`errors`
	`fmt`
	`github.com/boltdb/bolt`
	`lightChain/utils`
	`log`
	`os`
	`time`
)

const (
	dbFile             = "./db/lightChain_%s.db" // A key-value db created by boltdb. The key is block hash, the value is block body.
	blocksBucket       = "Blocks"                // The db has two buckets. One is blocksBucket (for blocks), another is utxoBucket (for UTXO).
	initCoinbaseReward = 666.0                   // The initial reward to the miner who successfully mined a block.
	rewardDecayNum     = 2016                    // Every rewardDecayNum blocks added to lightChain, halve the coinbase reward.
)

var genesisCoinbaseData = fmt.Sprintf("The genesis block of lightChain is created at %v", time.Now().Local())

// BlockChain is a list of Block linked by hash pointers. It only saves the newest block hash and the pointer
// to the local db file.
type BlockChain struct {
	Tip            []byte   // the newest block' hash
	Db             *bolt.DB // the pointer-to-db where the chain stored
	CoinbaseReward float64  // the coinbase reward value (decided by the chain length), this is the only way to generate new coins
}

// CreateBlockChain creates the lightChain across the whole network. The node whose Id is nodeId (actually network.CentralNode)
// does this creation. addr is its wallet address to receive the coinbase reward.
func CreateBlockChain(addr, nodeId string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeId)
	if ok, _ := utils.FileExists(dbFile); ok {
		fmt.Println("lightChain is found in the whole network. You should not create it again.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0644, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(
		func(tx *bolt.Tx) error {
			// create a bucket
			bucket, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				log.Panic(err)
			}

			// create a coinbase tx ---> create the genesis block
			coinbaseTx := NewCoinbaseTx(addr, genesisCoinbaseData, initCoinbaseReward)
			genesisBlock := NewGenesisBlock(coinbaseTx)

			// add the genesis block to the blockchain
			err = bucket.Put(genesisBlock.Hash, genesisBlock.SerializeBlock())
			if err != nil {
				log.Panic(err)
			}

			// the key []byte("l") always points to the newest block' hash
			err = bucket.Put([]byte("l"), genesisBlock.Hash)
			if err != nil {
				log.Panic(err)
			}
			tip = genesisBlock.Hash

			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return &BlockChain{tip, db, initCoinbaseReward}
}

// NewBlockChain requests lightChain from the whole network for the owner of nodeId and create a local db to save it.
// It returns a pointer to local copied BlockChain. NOTE: Before calling this function, the node with nodeId should have
// already copied the chain to its local storage.
func NewBlockChain(nodeId string) *BlockChain {
	dbFile := fmt.Sprintf(dbFile, nodeId)
	if ok, _ := utils.FileExists(dbFile); !ok {
		fmt.Println("No existing lightChain found across the whole network. Create one first.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0644, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			tip = bucket.Get([]byte("l"))
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	var chain = BlockChain{tip, db, initCoinbaseReward}
	chain.DecCoinbaseReward()
	return &chain
}

// AddBlock adds block to chain by writing it to db.
func (chain *BlockChain) AddBlock(block *Block) {
	err := chain.Db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))

			// if this block has been put into blockchain beforehand, just return
			blockInDb := bucket.Get(block.Hash)
			if blockInDb != nil {
				return nil
			}

			// otherwise just put it into blockchain
			err := bucket.Put(block.Hash, block.SerializeBlock())
			if err != nil {
				log.Panic(err)
			}

			// modify tip to the newest block
			lastHash := bucket.Get([]byte("l"))
			lastBlockData := bucket.Get(lastHash)
			lastBlock := DeserializeBlock(lastBlockData)
			if block.Height > lastBlock.Height { // the if-not condition could happen (when receives an already have block)
				err = bucket.Put([]byte("l"), block.Hash)
				if err != nil {
					log.Panic(err)
				}
				chain.Tip = block.Hash
			}

			return nil
		})
	if err != nil {
		log.Panic(err)
	}
}

// GetChainHeight returns the most recent block's height of chain.
func (chain *BlockChain) GetChainHeight() int {
	var lastBlock *Block
	err := chain.Db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			lastHash := bucket.Get([]byte("l"))
			lastBlockData := bucket.Get(lastHash)
			lastBlock = DeserializeBlock(lastBlockData)

			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return lastBlock.Height
}

// GetBlocksNum returns the number of blocks in current BlockChain.
func (chain *BlockChain) GetBlocksNum() int {
	iter := chain.Iterator()
	numBlocks := 0
	for {
		block := iter.Next()
		numBlocks++
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return numBlocks
}

// ValidBlockChain checks whether chain is legal.
func (chain *BlockChain) ValidBlockChain() bool {
	return chain.GetBlocksNum() == chain.GetChainHeight()+1
}

// GetTx returns the specific Transaction denoted by blockIdx and txIdx.
func (chain *BlockChain) GetTx(blockIdx, txIdx int) (*Transaction, error) {
	iter := chain.Iterator()
	numIdx := 0
	for {
		block := iter.Next()
		numIdx++
		if numIdx == blockIdx {
			return block.Transactions[txIdx], nil
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return nil, errors.New("transaction not found")
}

// DecCoinbaseReward decreases the coinbase reward every rewardDecayNum blocks.
func (chain *BlockChain) DecCoinbaseReward() {
	decayTimes := chain.GetBlocksNum() / rewardDecayNum
	for i := 0; i < decayTimes; i++ {
		chain.CoinbaseReward /= 2
	}
}

// GetBlock returns the pointer to the block whose hash is blockHash.
func (chain *BlockChain) GetBlock(blockHash []byte) (*Block, error) {
	var block *Block
	err := chain.Db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			blockData := bucket.Get(blockHash)
			if blockData == nil {
				return errors.New("block not found")
			}
			block = DeserializeBlock(blockData)

			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return block, nil
}

// GetAllBlocksHashes returns a slice of hashes, each for a block.
func (chain *BlockChain) GetAllBlocksHashes() [][]byte {
	var allHashes [][]byte
	iter := chain.Iterator()

	for {
		block := iter.Next()
		allHashes = append(allHashes, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return allHashes
}

// MineBlock appends a new block where txs are packed to chain through mining. Each new block is mined through PoW and
// the key-value pair (block hash, serialized block data) will be stored into the db. Before mining, each transaction
// packed in the block should be legal.
func (chain *BlockChain) MineBlock(txs []*Transaction) *Block {
	// verify all tx in txs
	for _, tx := range txs {
		if chain.VerifyTx(tx) != true {
			log.Panic("Error: invalid transaction found!")
		}
	}

	// get the last block' hash for generating the new block
	var lastHash []byte
	var height int
	err := chain.Db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			lastHash = bucket.Get([]byte("l"))
			blockData := bucket.Get(lastHash)
			block := DeserializeBlock(blockData)
			height = block.Height

			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	// construct a new block with height++ and store it into db
	newBlock := NewBlock(txs, lastHash, height+1)
	err = chain.Db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			err := bucket.Put(newBlock.Hash, newBlock.SerializeBlock())
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
	if err != nil {
		log.Panic(err)
	}

	return newBlock
}

// FindTx returns a Transaction according to the Transaction Id, i.e. txId.
func (chain *BlockChain) FindTx(txId []byte) (Transaction, error) {
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			if bytes.Compare(tx.Id, txId) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction not found")
}

// FindUTXO returns all the unspent outputs (a map: {key: txId, value: unspent outputs in this tx}).
func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	utxo := make(map[string]TxOutputs)
	spentTxOutputs := make(map[string][]int)
	iter := chain.Iterator()

	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.Id)

		Outputs:
			for txOutputIdx, txOutput := range tx.Vout {
				if spentTxOutputs[txId] != nil {
					// at least one txOutput of tx whose Id is txId is spent out
					for _, spentOutIdx := range spentTxOutputs[txId] {
						if txOutputIdx == spentOutIdx {
							// this txOutput has been spent, goto the next txOutput
							continue Outputs
						}
					}
				}
				// this txOutput is not spent out, add it to utxo
				txOutputs := utxo[txId]
				txOutputs.Outputs = append(txOutputs.Outputs, txOutput)
				utxo[txId] = txOutputs
			}

			// as the input of tx, it must be spent
			// thus directly append the input tx' id and the corresponding txOutput idx to spentTxOutputs
			if !tx.IsCoinbaseTx() {
				for _, txInput := range tx.Vin {
					inTxId := hex.EncodeToString(txInput.TxId)
					spentTxOutputs[inTxId] = append(spentTxOutputs[inTxId], txInput.VoutIdx)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return utxo
}

/* The following two functions are wrappers to tx.Sign and tx.Verify. */

// SignTx signs on the inputs of Transaction tx with the sender's private key.
func (chain *BlockChain) SignTx(tx *Transaction, privateKey ecdsa.PrivateKey) {
	tx.Sign(privateKey, chain.getPrevTxs(tx))
}

// VerifyTx verifies the input's signature of the Transaction tx.
func (chain *BlockChain) VerifyTx(tx *Transaction) bool {
	// this is where the bug occurs! I just fix this. :-)
	if tx.IsCoinbaseTx() {
		return true
	}
	return tx.Verify(chain.getPrevTxs(tx))
}

// getPrevTxs returns a map of transactions whose output is pointed by some input of tx.
// In the returned map, the key is string of transaction's Id, the value is the tx itself.
func (chain *BlockChain) getPrevTxs(tx *Transaction) map[string]Transaction {
	prevTxs := make(map[string]Transaction)
	for _, txInput := range tx.Vin {
		prevTx, err := chain.FindTx(txInput.TxId)
		if err != nil {
			log.Panic(err)
		}
		prevTxs[hex.EncodeToString(prevTx.Id)] = prevTx
	}
	return prevTxs
}

// IterOnChain is an iterator on the blockchain.
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
			block = DeserializeBlock(encodedBlock)
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	iter.curBlockHash = block.PrevBlockHash
	return block
}
