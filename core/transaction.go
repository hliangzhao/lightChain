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
var coinbaseReward = 666.0

// Transaction consists of its Id, a collection of TxInput, and a collection of output TxOutput.
type Transaction struct {
	Id   []byte
	Vin  []TxInput
	Vout []TxOutput
}

// String formalizes the output style of a Transaction.
func (tx Transaction) String() string {
	var outStr []string
	outStr = append(outStr, fmt.Sprintf("TxId: %x", tx.Id))
	for txInputIdx, txInput := range tx.Vin {
		outStr = append(outStr, fmt.Sprintf("--input #%d", txInputIdx))
		outStr = append(outStr, fmt.Sprintf("----TxId: %x", txInput.TxId))
		outStr = append(outStr, fmt.Sprintf("----OutIdx: %x", txInput.VoutIdx))
		outStr = append(outStr, fmt.Sprintf("----Signature: %x", txInput.Signature))
		outStr = append(outStr, fmt.Sprintf("----PubKey: %x", txInput.PubKey))
	}
	for txOutputIdx, txOutput := range tx.Vout {
		outStr = append(outStr, fmt.Sprintf("--output #%d", txOutputIdx))
		outStr = append(outStr, fmt.Sprintf("----Value: %f", txOutput.Value))
		outStr = append(outStr, fmt.Sprintf("----PubKeyHash: %x", txOutput.PubKeyHash))
	}
	return strings.Join(outStr, "\n")
}

/* The following defines the data structure of TxInput and operations on it. */

// TxInput includes all information required for the input of a Transaction: TxId, VoutIdx, Signature, and PubKey.
// Wherein, TxId is the Id of some previous Transaction, at least one output of which is pointed by current Transaction's some input.
// VoutIdx is the index of the pointed output of the previous Transaction.
// Signature is the data bytes signed with sender's private key.
// PubKey is the public key of sender.  // TODO: or pubKeyHash?
type TxInput struct {
	TxId      []byte
	VoutIdx   int
	Signature []byte
	PubKey    []byte
}

// UseKey checks whether the owner of pubKeyHash can initialize the Transaction whose Vin contains txInput.
func (txInput *TxInput) UseKey(pubKeyHash []byte) bool {
	lockingHash := HashingPubKey(txInput.PubKey)
	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

/* The following defines the data structure of TxOutput and operations on it. */

// TxOutput includes all information required for the output of a Transaction: Value and PubKeyHash.
// Wherein, Value is the quantity of the coin LIG involved in the corresponding tx.
// PubKeyHash is the base58 encoding of the address of the receiver, which is a lock with the receiver's address.
type TxOutput struct {
	Value      float64
	PubKeyHash []byte
}

// Lock signs txOutput with the receiver's address addr.
func (txOutput *TxOutput) Lock(addr string) {
	fullPayload := utils.Base58Decoding([]byte(addr))
	pubKeyHash := fullPayload[1 : len(fullPayload)-4]
	txOutput.PubKeyHash = pubKeyHash
}

// IsLockedWithKey checks whether txOutput belongs to the owner of pubKeyHash.
func (txOutput *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(txOutput.PubKeyHash, pubKeyHash) == 0
}

// NewTxOutput creates a new TxOutput instance and returns the pointer to it.
func NewTxOutput(value float64, addr string) *TxOutput {
	txOutput := &TxOutput{value, nil}
	txOutput.Lock(addr)
	return txOutput
}

// TxOutputs is a collection of TxOutput.
type TxOutputs struct {
	Outputs []TxOutput
}

// SerializeOutputs returns encoded bytes for the input txOutputs.
func (txOutputs TxOutputs) SerializeOutputs() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(txOutputs)
	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}

// DeserializeOutputs returns a TxOutputs instance decoded from the serialized data encodedData.
func DeserializeOutputs(encodedData []byte) TxOutputs {
	var txOutputs TxOutputs
	decoder := gob.NewDecoder(bytes.NewReader(encodedData))

	err := decoder.Decode(&txOutputs)
	if err != nil {
		log.Panic(err)
	}

	return txOutputs
}

/* The following defines the operations on Transaction. */

// NewCoinbaseTx returns a pointer to a newly created coinbase transaction. dstAddr is the address of wallet who does
// this creation (also the address to accept reward).
func NewCoinbaseTx(dstAddr, data string) *Transaction {
	if data == "" {
		// TODO: in bitcoin, these data are used to sample nonce.
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}
		data = fmt.Sprintf("%x", randData)
	}
	// txIn is from nowhere, thus its PubKey is set by data
	txIn := TxInput{[]byte{}, -1, nil, []byte(data)}
	// TODO: decrease coinbase reward before construct the coinbaseTx.
	txOut := NewTxOutput(coinbaseReward, dstAddr)
	tx := Transaction{nil, []TxInput{txIn}, []TxOutput{*txOut}}
	tx.Id = tx.Hashing()
	return &tx
}

// IsCoinbaseTx judges whether the caller is a coinbase Transaction, i.e. the transaction for
// generating new coins (as the transaction fee for the successful miner).
func (tx *Transaction) IsCoinbaseTx() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxId) == 0 && tx.Vin[0].VoutIdx == -1
}

// NewUTXOTx returns a pointer to a newly created UTXO transaction. When creating an UTXO transaction,
// firstly, we need to find the wallet of sender according to srcAddr; then, we need to check whether this
// wallet has enough coins to support this tx. If yes, construct Vin (with src wallet's PubKey) and Vout.
// Finally, sign this tx with src wallet's private key.
func NewUTXOTx(srcAddr, dstAddr string, amount float64, utxoSet *UTXOSet) *Transaction {
	var vin []TxInput
	var vout []TxOutput

	// get the sender's wallet's pubKeyHash
	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	senderWallet, err := wallets.GetWallet(srcAddr)
	if err != nil {
		log.Panic(err)
	}
	pubKeyHash := HashingPubKey(senderWallet.PubKey)

	// find enough unspent outputs to support this tx
	accumulated, unspentOutputs := utxoSet.FindSpendableOutputs(pubKeyHash, amount)
	if accumulated < amount {
		log.Panic("Error: the sender does not have enough coins to support this transaction")
	}
	for txId, outputIndices := range unspentOutputs {
		decodedTxId, err := hex.DecodeString(txId)
		if err != nil {
			log.Panic(err)
		}
		for _, outputIdx := range outputIndices {
			vin = append(vin, TxInput{decodedTxId, outputIdx, nil, senderWallet.PubKey})
		}
	}

	vout = append(vout, *NewTxOutput(amount, dstAddr))
	if accumulated > amount {
		// generate the change transaction
		// TODO: support new addr generation.
		vout = append(vout, *NewTxOutput(accumulated-amount, srcAddr))
	}

	tx := Transaction{nil, vin, vout}
	tx.Id = tx.Hashing()

	// sign each input of this transaction with sender's privateKey
	utxoSet.BlockChain.SignTx(&tx, senderWallet.PrivateKey)
	return &tx
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
		// TODO: the following line is not fully understood.
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

// Verify checks whether all the inputs of Transaction tx are legal. Wherein, this function checks whether the inputs
// of tx are tampered by some evil guys. If yes, the signature is incorrect.
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

		x, y := big.Int{}, big.Int{}
		keyLength := len(txInput.PubKey)
		x.SetBytes(txInput.PubKey[:(keyLength / 2)])
		y.SetBytes(txInput.PubKey[(keyLength / 2):])

		r, s := big.Int{}, big.Int{}
		sigLength := len(txInput.Signature)
		r.SetBytes(txInput.Signature[:(sigLength / 2)])
		s.SetBytes(txInput.Signature[(sigLength / 2):])

		if ecdsa.Verify(&ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}, copiedTx.Id, &r, &s) == false {
			return false
		}
	}
	return true
}

// TODO: it seems like there is no need to copy tx for SerializeTx.
// Hashing returns the hashing result of input tx, which is used to set its Id.
func (tx *Transaction) Hashing() []byte {
	var hash [32]byte
	copiedTx := *tx
	copiedTx.Id = []byte{}
	hash = sha256.Sum256(copiedTx.SerializeTx())
	return hash[:]
}

// SerializeTx converts the content of tx into a serialized byte slice.
func (tx Transaction) SerializeTx() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}
