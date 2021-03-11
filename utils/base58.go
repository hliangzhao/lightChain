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

/* This file gives the encoding and decoding methods of base58, which are used to generate the wallet addresses. */
package utils

import (
	`bytes`
	`math/big`
)

var alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
var length = int64(len(alphabet))

// Base58Encoding returns the base58 encoding result for input.
func Base58Encoding(input []byte) []byte {
	var encoded []byte
	x := big.NewInt(0).SetBytes(input)
	base := big.NewInt(length)
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		// x <--- x / 58, mod <--- x - x / 58
		x.DivMod(x, base, mod)
		encoded = append(encoded, alphabet[mod.Int64()])
	}
	ReverseBytes(encoded)
	for b := range input {
		if b == 0x00 {
			encoded = append([]byte{alphabet[0]}, encoded...)
		}
	}

	return encoded
}

// Base58Decoding returns the decoding result from the base58 encoded input.
func Base58Decoding(input []byte) []byte {
	tmp := big.NewInt(0)
	zeroBytes := 0

	for b := range input {
		if b == 0x00 {
			zeroBytes++
		}
	}
	payload := input[zeroBytes:]
	for _, b := range payload {
		byteIdx := bytes.IndexByte(alphabet, b)
		tmp.Mul(tmp, big.NewInt(length))
		tmp.Add(tmp, big.NewInt(int64(byteIdx)))
	}

	decoded := tmp.Bytes()
	decoded = append(bytes.Repeat([]byte{byte(0x00)}, zeroBytes), decoded...)
	return decoded
}
