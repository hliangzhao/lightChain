package core

type BlockChain struct {
	Blocks []*Block
}

// NewBlockChain: generate the blockchain.
func NewBlockChain() *BlockChain {
	return &BlockChain{[]*Block{NewGenesisBlock()}}
}

// AppendBlock: Append a new block with data as content to the blockchain.
func (chain *BlockChain) AppendBlock(data string) {
	prevBlock := chain.Blocks[len(chain.Blocks) - 1]
	newBlock := NewBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, newBlock)
}