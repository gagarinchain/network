package sparse

import (
	"bytes"
	"fmt"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	"math/big"
	"strings"
)

type NodeId struct {
	minor *big.Int
	major *big.Int
}

func (n NodeId) String() string {
	return fmt.Sprintf("%v:%v", stringBits(n.minor, 256), stringBits(n.major, 256))
}

func NewNodeId(minor *big.Int, major *big.Int) *NodeId {
	return &NodeId{minor: minor, major: major}
}

func comparator(a, b interface{}) int {
	aid := a.(*NodeId)
	bid := b.(*NodeId)

	diffa := aid.major.Cmp(bid.major)
	if diffa == 0 {
		return aid.minor.Cmp(bid.minor)
	}
	return diffa
}
func comparatorBig(a, b interface{}) int {
	ai := a.(*big.Int)
	bi := b.(*big.Int)

	return ai.Cmp(bi)
}

//simple Sparse Merkle Tree implementation described here https://eprint.iacr.org/2016/683.pdf
type SMT struct {
	root   *NodeId
	nodes  *treemap.Map
	leaves *treemap.Map
}

type Proof struct {
	//todo return bitmap here
	NodeIds []*NodeId //todo probably change to bitmap
	Value   []common.Hash
}

func (smt *SMT) Root() *NodeId {
	return smt.root
}

func NewSMT(depth int) *SMT {
	root := NewNodeId(big.NewInt(0), big.NewInt(0).SetBit(big.NewInt(0), depth-1, 1))

	smt := &SMT{root: root}
	smt.nodes = treemap.NewWith(comparator)
	smt.leaves = treemap.NewWith(comparatorBig)
	smt.nodes.Put(root, []byte{0})
	return smt
}

func (smt *SMT) Add(key *big.Int, val []byte) {
	smt.leaves.Put(key, val)
	ids := GetPath(key, smt.Depth())
	relatives := GetRelativePath(ids)

	smt.leaves.Put(key, val)                              //add leaf
	smt.nodes.Put(ids[len(ids)-1], crypto.Keccak256(val)) //add leaf hash
	if len(ids) > 1 {                                     // if we are deeper than 1 element
		for i := len(ids) - 2; i >= 0; i-- { //take next to leaf node from bottom, and go up the tree
			val := smt.getValueSafe(ids[i+1])          // child node
			relVal := smt.getValueSafe(relatives[i+1]) // another child node (relative)

			if comparator(ids[i+1], relatives[i+1]) < 0 { // if node is left and relative is right
				smt.nodes.Put(ids[i], crypto.Keccak256(append(val, relVal...)))
			} else { //id node is right and relative is left
				smt.nodes.Put(ids[i], crypto.Keccak256(append(relVal, val...)))
			}
			//println ("put ", ids[i].String())
			//keys := smt.nodes.Keys()
			//spew.Dump(keys)
		}
	}

}

func (smt *SMT) Proof(key *big.Int) (proof *Proof, found bool) {
	leaf := NewNodeId(key, key)
	_, found = smt.leaves.Get(leaf)
	if !found {
		return
	}

	proof = &Proof{}
	for comparator(leaf, smt.root) != 0 {
		relative := leaf.GetRelative()
		hash := smt.getValueSafe(relative)

		if bytes.Compare(hash, []byte{0}) == 0 {
			proof.NodeIds = append(proof.NodeIds, nil)
		} else {
			proof.NodeIds = append(proof.NodeIds, relative)
		}
		proof.Value = append(proof.Value, common.BytesToHash(hash))
		leaf = leaf.GetParent()
	}

	return
}

func (smt *SMT) getValueSafe(key *NodeId) []byte {
	value, found := smt.nodes.Get(key)
	if !found {
		return []byte{0}
	} else {
		return value.([]byte)
	}
}

func (smt *SMT) Get(key *big.Int) (val []byte, found bool) {
	value, found := smt.leaves.Get(key)
	if !found || value == nil {
		return
	}
	val = value.([]byte)
	return
}

type Entry struct {
	Key *big.Int
	Val []byte
}

func (smt *SMT) Entries() (entries []*Entry) {
	iterator := smt.leaves.Iterator()
	for iterator.Next() {
		entries = append(entries, &Entry{
			Key: iterator.Key().(*big.Int),
			Val: iterator.Value().([]byte),
		})
	}

	return
}

func (smt *SMT) Depth() int {
	return smt.root.GetLvl() + 1
}

func (smt *SMT) GetById(id *NodeId) ([]byte, bool) {
	value, found := smt.nodes.Get(id)
	if !found {
		return nil, false
	}
	return value.([]byte), true

}

func (smt *SMT) PutNode(id *NodeId, val []byte) {
	smt.nodes.Put(id, val)
}

//Returns node ids top down, the first element is root
func GetPath(key *big.Int, length int) (ids []*NodeId) {
	for i := length - 1; i >= 0; i-- {
		rsh := big.NewInt(0).Rsh(key, uint(i+1))
		leftmost := big.NewInt(0).Lsh(rsh, uint(i+1))
		rightleftmost := big.NewInt(0).SetBit(leftmost, i, 1)
		ids = append(ids, NewNodeId(leftmost, rightleftmost))
	}
	ids = append(ids, NewNodeId(key, key))
	return ids
}

//Returns relative nodes in the same order as path ordered,
//Path always contains root
func GetRelativePath(path []*NodeId) (relatives []*NodeId) {
	for _, n := range path {
		relatives = append(relatives, n.GetRelative())

	}
	return relatives
}

//returns relative of current node, node that has the same parent and placed on the same tree level.
//it should be called sister/brother if we had single word for such a relatives
//note: relative for root is root itself, but this method do not check whether correct node is passed, so client have to check
func (n *NodeId) GetRelative() *NodeId {
	pos := n.GetLvl() + 1
	relative := NewNodeId(big.NewInt(0).SetBit(n.minor, pos, n.minor.Bit(pos)^1), big.NewInt(0).SetBit(n.major, pos, n.major.Bit(pos)^1))
	return relative
}

func (n *NodeId) GetLvl() int {
	xor := big.NewInt(0).Xor(n.minor, n.major)
	return xor.BitLen() - 1
}

func (n *NodeId) IsLeft() bool {
	pos := big.NewInt(0).Xor(n.minor, n.major).BitLen()
	if pos >= n.minor.BitLen() && pos >= n.major.BitLen() { //this is root, we have different bit on top level
		return true
	}
	return n.minor.Bit(pos) == 0 //if previous to different bit is 0 then we are left subtree
}
func (n *NodeId) GetParent() (parent *NodeId) {
	pos := n.GetLvl() + 1
	if n.IsLeft() {
		return NewNodeId(n.minor, big.NewInt(0).SetBit(n.minor, pos, 1))
	} else {
		return NewNodeId(big.NewInt(0).SetBit(n.minor, pos, 0), n.minor)
	}
}

func printBits(int2 *big.Int, size int) {
	for i := size - 1; i >= 0; i-- {
		print(int2.Bit(i))
	}
}

func stringBits(int2 *big.Int, size int) string {
	b := strings.Builder{}
	for i := size - 1; i >= 0; i-- {
		b.WriteString(fmt.Sprintf("%b", int2.Bit(i)))
	}

	return b.String()
}
