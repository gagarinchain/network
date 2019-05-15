package trie

import (
	"github.com/poslibp2p/common/eth/common"
	"github.com/poslibp2p/common/eth/crypto"
	"strings"
)

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}

type Node interface {
	Key() []byte
	Hash() common.Hash
	CalcHash() common.Hash
	Copy() Node
}

type routingNode struct {
	key      []byte
	hash     common.Hash
	Children [16]Node
}

func (r *routingNode) Key() []byte {
	return r.key
}

func (r *routingNode) Hash() common.Hash {
	return r.hash
}

func (r *routingNode) Copy() Node {
	var keyCopy []byte
	copy(keyCopy, r.key)
	cpy := &routingNode{hash: r.hash, key: keyCopy}

	for i, n := range r.Children {
		cpy.Children[i] = n.Copy()
	}

	return cpy
}

func (r *routingNode) CalcHash() common.Hash {
	var aggregate []byte
	for _, child := range r.Children {
		if child == nil {
			continue
		}
		aggregate = append(aggregate, child.Hash().Bytes()...)
	}
	h := crypto.Keccak256Hash(aggregate)
	r.hash = h
	return h
}

type valueNode struct {
	key   []byte
	hash  common.Hash
	value []byte
}

func (v *valueNode) Key() []byte {
	return v.key
}

func (v *valueNode) Hash() common.Hash {
	return v.hash
}

func (v *valueNode) Copy() Node {
	var keyCopy []byte
	copy(keyCopy, v.key)

	var valueCopy []byte
	copy(valueCopy, v.key)

	return &valueNode{key: keyCopy, value: valueCopy, hash: v.hash}
}

func (v *valueNode) CalcHash() common.Hash {
	h := crypto.Keccak256Hash(v.value)
	v.hash = h
	return h
}

type nilNode struct {
}

func (n *nilNode) Key() []byte {
	return []byte("")
}

func (n *nilNode) Hash() common.Hash {
	return emptyState
}

func (n *nilNode) Copy() Node {
	return n
}

func (n *nilNode) CalcHash() common.Hash {
	return emptyState
}

func findIndex(letter byte) int {
	for i, v := range indices {
		if v == strings.ToLower(string(letter)) {
			return i
		}
	}

	log.Error("no index found for %v", letter)
	return -1
}
