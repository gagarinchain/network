package trie

import (
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/op/go-logging"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")

	// emptyState is the known hash of an empty state trie entry.
	emptyState = crypto.Keccak256Hash(nil)
)
var log = logging.MustGetLogger("hotstuff")

// As we have fixed key length we will benefit several simplifications
// - No mixed nodes, we have leaf or routing Node only
type FixedLengthHexKeyMerkleTrie struct {
	root Node
}

func New() *FixedLengthHexKeyMerkleTrie {
	trie := FixedLengthHexKeyMerkleTrie{root: &nilNode{}}
	return &trie
}

func (t *FixedLengthHexKeyMerkleTrie) Get(key []byte) (val []byte, found bool) {
	n := find(t.root, key)
	if n == nil {
		return nil, false
	}
	if curNode, ok := n.(*valueNode); ok {
		return curNode.value, true
	} else {
		log.Error("error while looking for Node")
		return nil, false
	}

}

func (t *FixedLengthHexKeyMerkleTrie) Proof() common.Hash {
	if t.root == nil {
		return emptyState
	}
	return t.root.Hash()
}

func find(n Node, key []byte) Node {
	if n == nil {
		return nil
	}

	prefix := CommonPrefix(n.Key(), key)
	postfix := key[len(prefix):]

	if Equal(prefix, key) {
		return n
	} else {
		switch n.(type) {
		case *routingNode:
			curNode := n.(*routingNode)
			ind := findIndex(postfix[0])
			nextNode := curNode.Children[ind]
			return find(nextNode, postfix)
		case *valueNode:
			curNode := n.(*valueNode)
			if Equal(prefix, postfix) {
				return curNode
			} else {
				return nil
			}
		}
	}
	return nil
}

//key is a hex encoded byte key
func (t *FixedLengthHexKeyMerkleTrie) InsertOrUpdate(key, value []byte) {
	if t.root == nil { //Node not found, can be only if trie is empty
		t.root = &valueNode{key: key, value: value}
		t.root.CalcHash()
		return
	}

	n := insert(t.root, key, value)

	if n != nil {
		t.root = n
	}

}

func (t *FixedLengthHexKeyMerkleTrie) Copy() *FixedLengthHexKeyMerkleTrie {
	tcopy := t.root.Copy()

	return &FixedLengthHexKeyMerkleTrie{root: tcopy}
}

func insert(n Node, key []byte, value []byte) (newNode Node) {
	if n == nil {
		vNode := &valueNode{key: key, value: value}
		vNode.CalcHash()
		return vNode
	}

	prefix := CommonPrefix(n.Key(), key)
	postfix := key[len(prefix):]

	split := len(prefix) < len(n.Key())

	switch n.(type) {
	case *nilNode:
		vNode := &valueNode{key: key, value: value}
		vNode.CalcHash()
		return vNode
	case *routingNode:
		current := n.(*routingNode)
		if split {
			postfixS := n.Key()[len(prefix):]
			current.key = postfixS

			vNode := &valueNode{key: postfix, value: value}
			vNode.CalcHash()

			rNode := &routingNode{key: prefix}
			ind := findIndex(postfix[0])
			indS := findIndex(postfixS[0])
			rNode.Children[ind] = vNode
			rNode.Children[indS] = current
			rNode.CalcHash()
			return rNode

		} else {
			ind := findIndex(postfix[0])
			nextNode := current.Children[ind]
			newNode2 := insert(nextNode, postfix, value)
			if newNode2 != nil {
				current.Children[ind] = newNode2
				current.CalcHash()
				return current
			}
		}
	case *valueNode:
		//since we have fixed length trie, we will never have splits for value nodes
		current := n.(*valueNode)
		if Equal(prefix, key) { //exact match, update value
			current.value = value
			current.CalcHash()
			return current
		} else { //split Node
			rNode := &routingNode{key: prefix}
			postfixS := n.Key()[len(prefix):]

			ind := findIndex(postfix[0])
			indS := findIndex(postfixS[0])
			vNode := &valueNode{key: postfix, value: value}
			vNode.CalcHash()
			rNode.Children[ind] = vNode
			current.key = postfixS
			rNode.Children[indS] = current
			rNode.CalcHash()
			return rNode
		}
	default:
		panic("wrong Node type")
	}

	return nil
}

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func CommonPrefix(a, b []byte) (c []byte) {
	for i, v := range a {
		if v == b[i] {
			c = append(c, v)
		} else {
			return
		}
	}
	return
}

//Ordered array made with left-right-root walk
func (t *FixedLengthHexKeyMerkleTrie) Values() (values [][]byte) {

	return aggregate(values, t.root)
}

type Entry struct {
	Key   []byte
	Value []byte
}

func (t *FixedLengthHexKeyMerkleTrie) Entries() (entries []Entry) {
	return aggregateEntries(nil, entries, t.root)
}

func aggregateEntries(keyPrefix []byte, source []Entry, n Node) []Entry {
	if n == nil {
		return source
	}
	switch n.(type) {
	case *nilNode:
		return source
	case *valueNode:
		return append(source, Entry{
			Key:   append(keyPrefix, n.(*valueNode).key...),
			Value: n.(*valueNode).value,
		})
	case *routingNode:
		node := n.(*routingNode)
		for _, child := range node.Children {
			bytes := append(keyPrefix, node.key...)
			res := make([]byte, len(bytes))
			copy(res, bytes)
			source = aggregateEntries(res, source, child)
		}
		return source
	}

	return nil
}

func aggregate(source [][]byte, n Node) [][]byte {
	if n == nil {
		return source
	}
	switch n.(type) {
	case *nilNode:
		return source
	case *valueNode:
		return append(source, n.(*valueNode).value)
	case *routingNode:
		for _, child := range n.(*routingNode).Children {
			source = aggregate(source, child)
		}
		return source
	}

	return nil
}
