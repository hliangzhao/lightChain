package main

import (
	`lightChain/core`
	`log`
)

func main() {
	lightChain := core.NewBlockChain()
	defer func() {
		err := lightChain.Db.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	cli := CLI{lightChain}
	cli.Run()
}