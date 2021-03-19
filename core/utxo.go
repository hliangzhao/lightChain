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
	`encoding/hex`
	`github.com/boltdb/bolt`
	`log`
)

// The bucket for store utxo. Key: TxId, Value: Unspent outputs in that tx.
const utxoBucket = "ChainState"

type UTXOSet struct {
	BlockChain *BlockChain
}

// FindSpendableOutputs returns the coin quantity (the sum of legal output's value) and the corresponding slice of
// unspent transactions' outputs (UTXO) for the owner of pubKeyHash, where the coin quantity is expected to not less
// than amount. Since all utxos are stored in db when new tx is created, we just directly read them from db.
func (utxoSet UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount float64) (float64, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0.0
	db := utxoSet.BlockChain.Db

	err := db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(utxoBucket))
			cursor := bucket.Cursor()

			// get txOutputs of each tx
			for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
				txId := hex.EncodeToString(key)
				txOutputs := DeserializeOutputs(value)

				for txOutputIdx, txOutput := range txOutputs.Outputs {
					if txOutput.IsLockedWithKey(pubKeyHash) && accumulated < amount {
						accumulated += txOutput.Value
						unspentOutputs[txId] = append(unspentOutputs[txId], txOutputIdx)
					}
				}
			}
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs
}

// FindUTXO returns the UTXO for the owner of pubKeyHash. Since all utxos are stored in db when new tx is created,
// we just directly read them from db.
func (utxoSet UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var utxo []TxOutput
	db := utxoSet.BlockChain.Db

	err := db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(utxoBucket))
			cursor := bucket.Cursor()

			for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
				txOutputs := DeserializeOutputs(value)

				for _, txOutput := range txOutputs.Outputs {
					if txOutput.IsLockedWithKey(pubKeyHash) {
						utxo = append(utxo, txOutput)
					}
				}
			}
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return utxo
}

// CountTxs returns the number of Transaction in the UTXO set of current lightChain.
func (utxoSet UTXOSet) CountTxs() int {
	counter := 0
	db := utxoSet.BlockChain.Db

	err := db.View(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(utxoBucket))
			cursor := bucket.Cursor()

			for key, _ := cursor.First(); key != nil; key, _ = cursor.Next() {
				counter++
			}
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	return counter
}

// Rebuild rebuilds the UTXO set according to current status of lightChain.
func (utxoSet UTXOSet) Rebuild() {
	db := utxoSet.BlockChain.Db

	// delete the old utxo bucket and create a brand new one
	err := db.Update(
		func(tx *bolt.Tx) error {
			err := tx.DeleteBucket([]byte(utxoBucket))
			if err != nil && err != bolt.ErrBucketNotFound {
				log.Panic(err)
			}

			_, err = tx.CreateBucket([]byte(utxoBucket))
			if err != nil {
				log.Panic(err)
			}
			return nil
		})
	if err != nil {
		log.Panic(err)
	}

	// call BlockChain.FindUTXO to get the new utxo set, and save the content of it into the newly created bucket
	newUtxo := utxoSet.BlockChain.FindUTXO()
	err = db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(utxoBucket))

			for txId, txOutputs := range newUtxo {
				key, err := hex.DecodeString(txId)
				if err != nil {
					log.Panic(err)
				}
				err = bucket.Put(key, txOutputs.SerializeOutputs())
				if err != nil {
					log.Panic(err)
				}
			}
			return nil
		})
	if err != nil {
		log.Panic(err)
	}
}

// Update updates the utxo set according to the newly mined block. Here block must be the tip block of lightChain.
// For this reason, we just need to check each input of the pointed beforehand txs.
func (utxoSet UTXOSet) Update(block *Block) {
	db := utxoSet.BlockChain.Db

	err := db.Update(
		func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(utxoBucket))

			// according to the inputs of each tx in this block, find the beforehand txs whose outputs are the inputs of this tx.
			// for those beforehand txs, add their not spent-out outputs to utxo (if exist)
			for _, tx := range block.Transactions {
				if !tx.IsCoinbaseTx() {
					for _, vin := range tx.Vin {
						updatedOutputs := TxOutputs{}
						outs := DeserializeOutputs(bucket.Get(vin.TxId))
						for outIdx, out := range outs.Outputs {
							// note that an output can never be pointed by multiple inputs!
							// Thus, if outIdx is not vin.VoutIdx, outIdx is not pointed by any vin. Thus this out is unspent
							if outIdx != vin.VoutIdx {
								// out is not spent out in this newly mined block, add it to utxo
								updatedOutputs.Outputs = append(updatedOutputs.Outputs, out)
							}
						}
						// when rebuild utxo, we allocate a k-v pair for every tx
						// if some tx's outputs are all been spent out, just remove the corresponding k-v pair
						if len(updatedOutputs.Outputs) == 0 {
							err := bucket.Delete(vin.TxId)
							if err != nil {
								log.Panic(err)
							}
						} else {
							// otherwise, just update k-v pair
							err := bucket.Put(vin.TxId, updatedOutputs.SerializeOutputs())
							if err != nil {
								log.Panic(err)
							}
						}
					}
				}

				// of course all the outputs in the newly packed tx are unspent out, just add them to utxo
				newOutputs := TxOutputs{}
				for _, out := range tx.Vout {
					newOutputs.Outputs = append(newOutputs.Outputs, out)
				}

				err := bucket.Put(tx.Id, newOutputs.SerializeOutputs())
				if err != nil {
					log.Panic(err)
				}
			}

			return nil
		})
	if err != nil {
		log.Panic(err)
	}
}
