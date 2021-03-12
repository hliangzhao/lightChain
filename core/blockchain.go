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

// TODO: the db file path should be outside the project root path (for example, /var/db/).
/* I use the key-value database boltdb to save lightChain. Here the key is each block' hash, and the corresponding
value is the serialized data bytes of the block. */
const dbFile = "lightChain.db"
const blocksBucket = "lightChain"

var genesisCoinbaseData = fmt.Sprintf("The genesis block of lightChain is created at %v", time.Now().Local())

// BlockChain is a list linked by hash pointers. Thus this data structure only saves the newest
// block hash and the pointer to the local db file.
type BlockChain struct {
	Tip []byte   // the newest block' hash
	Db  *bolt.DB // the pointer-to-db where the chain stored
}

// CreateBlockChain creates the first lightChain across the whole network.
// addr is the address of the node who do the creation operation.
func CreateBlockChain(addr string) *BlockChain {
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
			coinbaseTx := NewCoinbaseTx(addr, genesisCoinbaseData)
			genesisBlock := NewGenesisBlock(coinbaseTx)

			// add the genesis block to the blockchain
			err = bucket.Put(genesisBlock.Hash, genesisBlock.Serialize())
			if err != nil {
				log.Panic(err)
			}

			// the key []byte("l") always points to the last block' hash
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

	return &BlockChain{tip, db}
}

// NewBlockChain requests lightChain from the whole network and create a local boltdb to save it.
// It returns a pointer to local copied BlockChain.
func NewBlockChain() *BlockChain {
	// TODO: implement p2p network to request the whole lightChain data (not just the tip) from other nodes.
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

	return &BlockChain{tip, db}
}

// MineBlock appends a new block to the blockchain through mining. Each new block is mined through PoW and
// the key-value pair (block hash, serialized block data) will be stored into the db.
// Before mining, each transaction packed in a block should be legal.
func (chain *BlockChain) MineBlock(txs []*Transaction) {
	// get the last block' hash for generating the new block
	var lastHash []byte
	for _, tx := range txs {
		if !chain.VerifyTx(tx) {
			log.Panic("Error: invalid transaction found!")
		}
	}

	err := chain.Db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(blocksBucket))
			lastHash = bucket.Get([]byte("l"))
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	// store the new block into db
	newBlock := NewBlock(txs, lastHash)
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
	if err != nil {
		log.Panic(err)
	}
}

// FindUnspentTxs returns a slice of Transaction for the node addr. For each transaction of this slice, at least one
// output is not spent out.
// TODO: this function may have bugs.
func (chain *BlockChain) FindUnspentTxs(pubKeyHash []byte) []Transaction {
	var unspentTxs []Transaction
	spentTxOutputs := make(map[string][]int)
	iter := chain.Iterator()

	// remember that the iteration direction is from the newest to the oldest block
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.Id)

		Outputs:
			for txOutputIdx, txOutput := range tx.Vout {
				if spentTxOutputs[txId] != nil {
					for _, spentOutIdx := range spentTxOutputs[txId] {
						if spentOutIdx == txOutputIdx {
							// this txOutput has been spent, goto the next txOutput
							continue Outputs
						}
					}
				}
				// only if no spent out matches txOutput, txOutput can be append to unspentTxs
				if txOutput.IsLockedWithKey(pubKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			// as the input of tx, it must be spent
			// thus directly append the input tx' id and the corresponding txOutput idx to spentTxOutputs
			if !tx.IsCoinbaseTx() {
				for _, input := range tx.Vin {
					if input.UseKey(pubKeyHash) {
						inTxId := hex.EncodeToString(input.TxId)
						spentTxOutputs[inTxId] = append(spentTxOutputs[inTxId], input.VoutIdx)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unspentTxs
}

// FindUTXO returns the slice of all the outputs which are not spent over the whole blockchain for node addr.
func (chain *BlockChain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXO []TxOutput
	unspentTxs := chain.FindUnspentTxs(pubKeyHash)
	for _, tx := range unspentTxs {
		for _, txOutput := range tx.Vout {
			if txOutput.IsLockedWithKey(pubKeyHash) {
				UTXO = append(UTXO, txOutput)
			}
		}
	}
	return UTXO
}

// FindSpendableOutputs returns the coin quantity (the sum of legal output's value) and the corresponding slice of
// unspent transactions' outputs (UTXO) for the node addr, where the coin quantity is expected to not less than amount.
func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTxs := chain.FindUnspentTxs(pubKeyHash)
	accumulated := 0

Search:
	for _, tx := range unspentTxs {
		txId := hex.EncodeToString(tx.Id)
		for txOutputIdx, txOutput := range tx.Vout {
			if txOutput.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += txOutput.Value
				unspentOutputs[txId] = append(unspentOutputs[txId], txOutputIdx)
				if accumulated >= amount {
					break Search
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

// FindTx returns a Transaction according to the Id provided.
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

// getPrevTxs returns a map of transactions whose output is the input of tx.
func (chain *BlockChain) getPrevTxs(tx *Transaction) map[string]Transaction {
	prevTxs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx, err := chain.FindTx(vin.TxId)
		if err != nil {
			log.Panic(err)
		}
		prevTxs[hex.EncodeToString(prevTx.Id)] = prevTx
	}
	return prevTxs
}

// SignTx signs on the inputs of one transaction with the sender's private key.
func (chain *BlockChain) SignTx(tx *Transaction, privateKey ecdsa.PrivateKey) {
	tx.Sign(privateKey, chain.getPrevTxs(tx))
}

// VerifyTx verifies the input's signature of the Transaction tx.
func (chain *BlockChain) VerifyTx(tx *Transaction) bool {
	return tx.Verify(chain.getPrevTxs(tx))
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
			block = Deserialize(encodedBlock)
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	iter.curBlockHash = block.PrevBlockHash
	return block
}
