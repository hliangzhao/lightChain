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
	`crypto/sha256`
	`log`
)

// MerkleNode is a node in Merkle tree. Data is the hashed (serialized) Transaction.
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// NewMerkleNode returns a pointer two a newly created Merkle tree node.
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}
	if left != nil && right != nil {
		// as an internal node
		prevHashes := append(left.Data, right.Data...)
		hashedData := sha256.Sum256(prevHashes)
		node.Data = hashedData[:]
	} else if left == nil && right == nil {
		// as a leaf node
		hashedData := sha256.Sum256(data)
		node.Data = hashedData[:]
	} else {
		log.Panic("Error: left Merkle node and right Merkle node are not at the same level")
	}
	node.Left, node.Right = left, right
	return &node
}

// TODO: add SortedMerkleTree.

// MerkleTree organizes all the Transaction in a block to a tree structure.
type MerkleTree struct {
	RootNode *MerkleNode
}

// NewMerkleTree creates a Merkle tree and returns the pointer to the root.
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode
	// should have odd leaf nodes
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	// set all the leaf nodes
	for _, d := range data {
		node := NewMerkleNode(nil, nil, d)
		nodes = append(nodes, *node)
	}

	// set all the internal nodes
	for depth := 0; depth < len(data)/2; depth++ {
		var sameDepthNodes []MerkleNode
		for j := 0; j < len(nodes); j += 2 {
			sameDepthNodes = append(sameDepthNodes, *NewMerkleNode(&nodes[j], &nodes[j+1], nil))
		}
		nodes = sameDepthNodes
	}

	if len(nodes) != 0 {
		return &MerkleTree{RootNode: &nodes[0]}
	} else {
		// if this if-condition holds, error happened!
		return &MerkleTree{}
	}
}
