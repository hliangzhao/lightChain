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
const targetBits = 2			// larger this number, more difficult the mining
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
func (proof *ProofOfWork) prepareData(nonce int) []byte {
	return bytes.Join(
		[][]byte{
			proof.block.PrevBlockHash,
			proof.block.Data,
			utils.Int2Hex(proof.block.TimeStamp),
			utils.Int2Hex(int64(targetBits)),
			utils.Int2Hex(int64(nonce))},
		[]byte{},
	)
}

// Mine finds the satisfied hash of data by trying different nonce.
func (proof *ProofOfWork) Mine() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining the block containing data: \"%s\"...\n", proof.block.Data)
	// iteration over each possible nonce util find a nonce that satisfies "sha256(data) < target"
	for nonce < maxNonce {
		data := proof.prepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(proof.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	return nonce, hash[:]
}

// Validate the mining result (nonce).
func (proof *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := proof.prepareData(proof.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	return -1 == hashInt.Cmp(proof.target)
}
