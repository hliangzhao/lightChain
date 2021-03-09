package utils

import (
	`bytes`
	`encoding/binary`
	`log`
)

// Int2Hex: transfer a int64 value into a byte slice.
func Int2Hex(num int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buf.Bytes()
}
