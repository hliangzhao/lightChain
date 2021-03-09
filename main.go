package main

import (
	`fmt`
	`lightChain/core`
	`strconv`
)

func main() {
	fmt.Println("======== Updating the blockchain ========")
	lightChain := core.NewBlockChain()
	lightChain.AppendBlock("Send 1 LIG to hliangzhao")
	lightChain.AppendBlock("Send 2 LIG to hliangzhao")

	fmt.Println("\n======== Print detailed information ========")
	for _, block := range lightChain.Blocks {
		fmt.Printf("Current block generated time: %d\n", block.TimeStamp)
		fmt.Printf("Previous block's hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)

		// new a validator with the mined block to examine the nonce
		proof := core.NewPoW(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(proof.Validate()))
	}
}