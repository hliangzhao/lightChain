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

package utils

import (
	`bytes`
	`encoding/gob`
	`log`
)

// TODO: use this function to replace all the Serialize() operations.
// Serialize returns the encoded bytes for the input e.
func Serialize(e interface{}) []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	err := encoder.Encode(e)
	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}

// TODO: use this function to replace all the Deserialize() operations. May use type assertion!
// Deserialize returns the decoded data to e.
func Deserialize(encodedData []byte) interface{} {
	var e interface{}
	decoder := gob.NewDecoder(bytes.NewReader(encodedData))

	err := decoder.Decode(&e)
	if err != nil {
		log.Panic(err)
	}
	return e
}