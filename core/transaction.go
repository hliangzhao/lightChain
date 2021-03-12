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
	`crypto/elliptic`
	`crypto/rand`
	`crypto/sha256`
	`encoding/gob`
	`encoding/hex`
	`fmt`
	`lightChain/utils`
	`log`
	`math/big`
	`strings`
)

// coinbaseReward is the reward to the miner who successfully mined a block.
// This value is saved in the coinbase transaction and this is the only way to generate new LIG coins.
// TODO: add a function to decrease the value of coinbaseReward after specified quantity of blocks.
var coinbaseReward = 666

// Transaction consists of its Id, a collection of TxInput, and a collection of output TxOutput.
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
	lockingHash := HashingPubKey(txInput.PubKey)
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
	pubKeyHash := utils.Base58Decoding(addr)
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

// Sign signs each input of the Transaction tx.
func (tx *Transaction) Sign(privateKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbaseTx() {
		return
	}

	for _, txInput := range tx.Vin {
		if prevTxs[hex.EncodeToString(txInput.TxId)].Id == nil {
			log.Panic("Error: previous transaction is not correct")
		}
	}

	copiedTx := tx.Copy()
	for txInputIdx, txInput := range copiedTx.Vin {
		prevTx := prevTxs[hex.EncodeToString(txInput.TxId)]
		copiedTx.Vin[txInputIdx].Signature = nil
		copiedTx.Vin[txInputIdx].PubKey = prevTx.Vout[txInput.VoutIdx].PubKeyHash
		copiedTx.Id = copiedTx.Hashing()
		copiedTx.Vin[txInputIdx].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, copiedTx.Id)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Vin[txInputIdx].Signature = signature
	}
}

// String formalizes the output style of a Transaction.
func (tx Transaction) String() string {
	var outStr []string
	outStr = append(outStr, fmt.Sprintf("TxId: %x", tx.Id))
	for txInputIdx, txInput := range tx.Vin {
		outStr = append(outStr, fmt.Sprintf("--#%d", txInputIdx))
		outStr = append(outStr, fmt.Sprintf("----TxId: %x", txInput.TxId))
		outStr = append(outStr, fmt.Sprintf("----OutIdx: %x", txInput.VoutIdx))
		outStr = append(outStr, fmt.Sprintf("----Signature: %x", txInput.Signature))
		outStr = append(outStr, fmt.Sprintf("----PubKey: %x", txInput.PubKey))
	}
	for txOutputIdx, txOutput := range tx.Vout {
		outStr = append(outStr, fmt.Sprintf("--#%d", txOutputIdx))
		outStr = append(outStr, fmt.Sprintf("----Value: %x", txOutput.Value))
		outStr = append(outStr, fmt.Sprintf("----OutIdx: %x\n", txOutput.PubKeyHash))
	}
	return strings.Join(outStr, "\n")
}

// NewCoinbaseTx returns a pointer to a newly created coinbase transaction.
func NewCoinbaseTx(dstAddr, signaturedData string) *Transaction {
	if signaturedData == "" {
		signaturedData = fmt.Sprintf("Reward to '%s'", dstAddr)
	}
	txIn := TxInput{[]byte{}, -1, nil, []byte(signaturedData)}
	txOut := NewTxOutput(coinbaseReward, dstAddr)
	tx := Transaction{nil, []TxInput{txIn}, []TxOutput{*txOut}}
	tx.Id = tx.Hashing()

	return &tx
}

// NewUTXOTx returns a pointer to a newly created UTXO transaction.
func NewUTXOTx(srcAddr, dstAddr string, amount int, chain *BlockChain) *Transaction {
	var vin []TxInput
	var vout []TxOutput

	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	wallet, err := wallets.GetWallet(srcAddr)
	if err != nil {
		log.Panic(err)
	}
	pubKeyHash := HashingPubKey(wallet.PubKey)
	accumulated, unspentOutputs := chain.FindSpendableOutputs(pubKeyHash, amount)
	if accumulated < amount {
		log.Panic("Error: src does not have enough coins")
	}

	for txId, outputIndices := range unspentOutputs {
		decodedTxId, err := hex.DecodeString(txId)
		if err != nil {
			log.Panic(err)
		}
		for _, outputIdx := range outputIndices {
			vin = append(vin, TxInput{decodedTxId, outputIdx, nil, wallet.PubKey})
		}
	}

	vout = append(vout, *NewTxOutput(amount, dstAddr))
	if accumulated > amount {
		// generate the change transaction
		vout = append(vout, *NewTxOutput(accumulated - amount, srcAddr))
	}

	tx := Transaction{nil, vin, vout}
	tx.Id = tx.Hashing()
	chain.SignTx(&tx, wallet.PrivateKey)
	return &tx
}

// Copy copies tx into a newly created Transaction. This Copy will copy everything of tx except the
// Signature and PubKey of txInput of tx.Vin.
func (tx *Transaction) Copy() Transaction {
	var vin []TxInput
	var vout []TxOutput
	for _, txInput := range tx.Vin {
		vin = append(vin, TxInput{
			TxId:      txInput.TxId,
			VoutIdx:   txInput.VoutIdx,
			Signature: nil,
			PubKey:    nil,
		})
	}
	for _, txOutput := range tx.Vout {
		vout = append(vout, TxOutput{
			Value:      txOutput.Value,
			PubKeyHash: txOutput.PubKeyHash,
		})
	}
	return Transaction{tx.Id, vin, vout}
}

// Verify checks whether all the inputs of Transaction tx are legal.
func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbaseTx() {
		return true
	}
	for _, txInput := range tx.Vin {
		if prevTxs[hex.EncodeToString(txInput.TxId)].Id == nil {
			log.Panic("Error: previous transaction is not correct")
		}
	}
	copiedTx := tx.Copy()
	curve := elliptic.P256()
	for txInputIdx, txInput := range tx.Vin {
		prevTx := prevTxs[hex.EncodeToString(txInput.TxId)]
		copiedTx.Vin[txInputIdx].Signature = nil
		copiedTx.Vin[txInputIdx].PubKey = prevTx.Vout[txInput.VoutIdx].PubKeyHash
		copiedTx.Id = copiedTx.Hashing()
		copiedTx.Vin[txInputIdx].PubKey = nil

		r, s := big.Int{}, big.Int{}
		sigLength := len(txInput.Signature)
		r.SetBytes(txInput.Signature[: (sigLength / 2)])
		s.SetBytes(txInput.Signature[(sigLength / 2): ])

		x, y := big.Int{}, big.Int{}
		keyLength := len(txInput.PubKey)
		x.SetBytes(txInput.PubKey[: (keyLength / 2)])
		y.SetBytes(txInput.PubKey[(keyLength / 2): ])

		if ecdsa.Verify(&ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}, copiedTx.Id, &r, &s) == false {
			return false
		}
	}
	return true
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
