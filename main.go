package main

import (
	`fmt`
	`lightChain/core`
)

func main() {
	lightChain := core.NewBlockChain()
	lightChain.AppendBlock("Send 1 LIG to hliangzhao")
	lightChain.AppendBlock("Send 2 LIG to hliangzhao")

	for _, block := range lightChain.Blocks {
		fmt.Printf("Current block generated time: %d\n", block.TimeStamp)
		fmt.Printf("Previous block's hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n\n", block.Hash)
	}
}