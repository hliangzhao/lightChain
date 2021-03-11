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
	`crypto/sha256`
	`encoding/gob`
	`encoding/hex`
	`fmt`
	`lightChain/utils`
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
// Signature is the data bytes signatured with sender's private key.
// PubKey is the public key of sender.
type TxInput struct {
	TxId      []byte
	VoutIdx   int
	Signature []byte
	PubKey    []byte
}

// UseKey checks whether the address pubKeyHash can initialize the transaction whose Vin contains txInput.
func (txInput *TxInput) UseKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(txInput.PubKey)
	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

// TxOutput includes all information required for the output of a Transaction: Value and PubKeyHash.
// Wherein, Value is the quantity of the coin LIG involved in the corresponding tx, PubKeyHash is
// the base58 encoding of the address of the receiver.
type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

func (txOutput *TxOutput) Lock(addr []byte) {
	pubKeyHash := utils.Base58Encoding(addr)
	pubKeyHash = pubKeyHash[1: len(pubKeyHash) - 4]
	txOutput.PubKeyHash = pubKeyHash
}

// IsLockedWithKey checks whether the node with pubKeyHash can use txOutput.
func (txOutput *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(txOutput.PubKeyHash, pubKeyHash) == 0
}

// NewTxOutput creates a new TxOutput instance and returns the pointer to it.
func NewTxOutput(value int, addr string) *TxOutput {
	txOutput := &TxOutput{value, nil}
	txOutput.Lock([]byte(addr))
	return txOutput
}

// IsCoinbaseTx judges whether the caller is a coinbase Transaction, i.e. the transaction for
// generating new coins (as the transaction fee for the successful miner).
func (tx *Transaction) IsCoinbaseTx() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxId) == 0 && tx.Vin[0].VoutIdx == -1
}

// Serialize converts tx into a serialized byte slice.
func (tx Transaction) Serialize() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}

// Hashing returns the hashing result of input tx.
func (tx *Transaction) Hashing() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.Id = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

// TODO: this function waits for changing.
func (tx *Transaction) Sign(privateKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbaseTx() {
		return
	}

	for _, txInput := range tx.Vin {
		if prevTxs[hex.EncodeToString(txInput.TxId)].Id == nil {
			log.Panic("Error: previous transaction is not correct")
		}
	}

	txCopy := *tx
	for txInputIdx, txInput := range tx.Vin {
		prevTx := prevTxs[hex.EncodeToString(txInput.TxId)]
	}
}

// TODO: this function waits for changing.
func (tx Transaction) String() string {

}

// TODO: this function waits for changing.
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

// TODO: this function waits for changing.
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

// TODO: this function waits for changing.
func (tx *Transaction) TrimmedCopy() Transaction {

}

// TODO: this function waits for changing.
func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {

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
