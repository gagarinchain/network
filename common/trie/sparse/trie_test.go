package sparse

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/magiconair/properties/assert"
	"math/big"
	"testing"
)

func TestNewSMT(t *testing.T) {
	smt := NewSMT(2)
	depth := smt.Depth()

	assert.Equal(t, depth, 2)

	assert.Equal(t, smt.Root(), NewNodeId(big.NewInt(0), big.NewInt(0b10)))
}

func TestIsLeft(t *testing.T) {

	id1 := NewNodeId(big.NewInt(0b100), big.NewInt(0b101))
	assert.Equal(t, id1.IsLeft(), true)

	id2 := NewNodeId(big.NewInt(0b110), big.NewInt(0b111))
	assert.Equal(t, id2.IsLeft(), false)

	id3 := NewNodeId(big.NewInt(0b010), big.NewInt(0b011))
	assert.Equal(t, id3.IsLeft(), false)

	id4 := NewNodeId(big.NewInt(0b010), big.NewInt(0b010))
	assert.Equal(t, id4.IsLeft(), true)
	id5 := NewNodeId(big.NewInt(0b101), big.NewInt(0b101))
	assert.Equal(t, id5.IsLeft(), false)
	id6 := NewNodeId(big.NewInt(0b100), big.NewInt(0b110))
	assert.Equal(t, id6.IsLeft(), false)

	assert.Equal(t, 0, id1.GetLvl())
	assert.Equal(t, 0, id2.GetLvl())
	assert.Equal(t, 0, id3.GetLvl())
	assert.Equal(t, -1, id4.GetLvl())
	assert.Equal(t, -1, id5.GetLvl())
	assert.Equal(t, 1, id6.GetLvl())

	assert.Equal(t, comparator(id1.GetRelative(), NewNodeId(big.NewInt(0b110), big.NewInt(0b111))), 0)
	assert.Equal(t, comparator(id2.GetRelative(), NewNodeId(big.NewInt(0b100), big.NewInt(0b101))), 0)
	assert.Equal(t, comparator(id3.GetRelative(), NewNodeId(big.NewInt(0b000), big.NewInt(0b001))), 0)
	assert.Equal(t, comparator(id4.GetRelative(), NewNodeId(big.NewInt(0b011), big.NewInt(0b011))), 0)
	assert.Equal(t, comparator(id5.GetRelative(), NewNodeId(big.NewInt(0b100), big.NewInt(0b100))), 0)
	assert.Equal(t, comparator(id6.GetRelative(), NewNodeId(big.NewInt(0b000), big.NewInt(0b010))), 0)
}

func TestGetPath(t *testing.T) {
	node := big.NewInt(0b00110000)
	//id := NewNodeId(node, node)

	path := GetPath(node, 8)

	relatives := GetRelativePath(path)

	spew.Dump(path)
	spew.Dump(relatives)

}

func TestProof(t *testing.T) {
	hashbig := big.NewInt(0b00110000)
	hashbig2 := big.NewInt(0b11001100)
	hashbig3 := big.NewInt(0b11011100)
	smt := NewSMT(8)
	smt.Add(hashbig, []byte("kuku"))
	smt.Add(hashbig2, []byte("kuku2"))
	smt.Add(hashbig3, []byte("kuku3"))

	spew.Dump(smt.nodes.Keys())
	spew.Dump(smt.nodes.Keys())

	for _, key := range smt.nodes.Keys() {
		spew.Dump(key)
	}

	assert.Equal(t, smt.nodes.Size(), 22)

	id1 := NewNodeId(big.NewInt(0b00110000), big.NewInt(0b00110000))
	hash1, _ := smt.nodes.Get(id1)
	assert.Equal(t, crypto.Keccak256([]byte("kuku")), hash1)
}
