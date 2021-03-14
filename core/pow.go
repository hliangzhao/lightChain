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
	`fmt`
	`lightChain/utils`
	`math`
	`math/big`
)

// number of 0 bits at the beginning of the hash for PoW, tuned for changing difficulty
const targetBits = 4 // larger this number, more difficult the mining
// the trial of nonce ranging from 0 to maxNonce
const maxNonce = math.MaxInt64

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

// NewPoW defines the PoW for each block.
func NewPoW(block *Block) *ProofOfWork {
	// set the target as 1 << (256 - targetBits)
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	return &ProofOfWork{block, target}
}

// prepareData joins the existing data into a byte slice, for the purpose of hashing.
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	return bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.HashingAllTxs(),
			utils.Int2Hex(pow.block.TimeStamp),
			utils.Int2Hex(int64(targetBits)),
			utils.Int2Hex(int64(nonce))},
		[]byte{},
	)
}

// Run finds the satisfied hash of data by trying different nonce.
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Println("Start to mine a new block...")
	// iteration over each possible nonce util find a nonce that satisfies "sha256(data) < target"
	for nonce < maxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	return nonce, hash[:]
}

// Validate the mining result (nonce).
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	return -1 == hashInt.Cmp(pow.target)
}
