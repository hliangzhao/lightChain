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
	`crypto/sha256`
	`encoding/gob`
	`encoding/hex`
	`fmt`
	`log`
)

// coinbaseReward is the reward to the miner who successfully mined a block.
// This value is saved in the coinbase transaction and this is the only way to generate new LIG coins.
// TODO: add a function to decrease the value of coinbaseReward after specified quantity of blocks.
const coinbaseReward = 666

// Transaction consists of its Id, the slice of TxInput, and the slice of output TxOutput.
type Transaction struct {
	Id   []byte
	Vin  []TxInput
	Vout []TxOutput
}

// TxInput includes all information required for the input of a Transaction: TxId, VoutIdx, and ScriptSig.
// Wherein, TxId is the Id of some previous Transaction, one output of which is pointed by current Transaction's input.
// VoutIdx is the index of the pointed output of the previous Transaction.
// ScriptSig is the signature to unlock the data of TxId: VoutIdx.
type TxInput struct {
	TxId      []byte
	VoutIdx   int
	ScriptSig string
}

// TxOutput includes all information required for the output of a Transaction: Value and ScriptPubKey.
// Wherein, Value is the quantity of the coin LIG involved, ScriptPubKey is used to lock this TxOutput.
type TxOutput struct {
	Value        int
	ScriptPubKey string
}

// IsCoinbaseTx judges whether the caller is a coinbase Transaction, i.e. the transaction for
// generating new coins (as the transaction fee for the successful miner).
func (tx *Transaction) IsCoinbaseTx() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxId) == 0 && tx.Vin[0].VoutIdx == -1
}

// NewCoinbaseTx returns a pointer to a newly created coinbase transaction.
func NewCoinbaseTx(dstAddr, signaturedData string) *Transaction {
	if signaturedData == "" {
		signaturedData = fmt.Sprintf("Reward to '%s'", dstAddr)
	}
	txIn := TxInput{[]byte{}, -1, signaturedData}
	txOut := TxOutput{coinbaseReward, dstAddr}
	tx := Transaction{nil, []TxInput{txIn}, []TxOutput{txOut}}
	tx.SetId()

	return &tx
}

// NewUTXOTx returns a pointer to a newly created UTXO transaction.
func NewUTXOTx(srcAddr, dstAddr string, amount int, chain *BlockChain) *Transaction {
	var vin []TxInput
	var vout []TxOutput
	accumulated, unspentOutputs := chain.FindSpendableOutputs(srcAddr, amount)
	if accumulated < amount {
		log.Panic("Error: src does not have enough coins")
	}

	for txId, outputIndices := range unspentOutputs {
		decodedTxId, err := hex.DecodeString(txId)
		if err != nil {
			log.Panic(err)
		}
		for _, outputIdx := range outputIndices {
			vin = append(vin, TxInput{decodedTxId, outputIdx, srcAddr})
		}
	}

	vout = append(vout, TxOutput{amount, dstAddr})
	if accumulated > amount {
		// generate the change transaction
		vout = append(vout, TxOutput{accumulated - amount, srcAddr})
	}

	tx := Transaction{nil, vin, vout}
	tx.SetId()
	return &tx
}

// SetId sets the Id of the caller Transaction based on the Transaction content and sha256 algorithm.
func (tx *Transaction) SetId() {
	var buf bytes.Buffer
	var hash [32]byte
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	hash = sha256.Sum256(buf.Bytes())
	tx.Id = hash[:]
}

// CanUnlockOutputWith judges whether the input string can unlock the caller TxInput by matching its
// ScriptSig.
func (in *TxInput) CanUnlockOutputWith(unlockingData string) bool {
	return in.ScriptSig == unlockingData
}

// CanBeUnlockedWith judges whether the caller TxOutput can be unlocked by matching its
// ScriptPubKey.
func (out *TxOutput) CanBeUnlockedWith(unlockingData string) bool {
	return out.ScriptPubKey == unlockingData
}
