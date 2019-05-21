package trie

import (
	"github.com/poslibp2p/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptyTrie(t *testing.T) {
	trie := New()
	trie.InsertOrUpdate([]byte("00000"), []byte("hello"))

	val, found := trie.Get([]byte("00000"))

	assert.True(t, found)
	assert.Equal(t, []byte("hello"), val)

}

func TestRichTrie(t *testing.T) {
	trie := New()
	trie.InsertOrUpdate([]byte("00000"), []byte("hello"))
	trie.InsertOrUpdate([]byte("00200"), []byte("hi"))
	trie.InsertOrUpdate([]byte("00300"), []byte("hello"))
	trie.InsertOrUpdate([]byte("00220"), []byte("hello"))
	trie.InsertOrUpdate([]byte("00000"), []byte("hi"))

	val, found := trie.Get([]byte("00000"))

	assert.True(t, found)
	assert.Equal(t, []byte("hi"), val)

}
func TestMerklelize(t *testing.T) {
	trie := New()
	trie.InsertOrUpdate([]byte("00000"), []byte("hello"))
	trie.InsertOrUpdate([]byte("00200"), []byte("hi"))
	trie.InsertOrUpdate([]byte("00300"), []byte("hello"))
	trie.InsertOrUpdate([]byte("00220"), []byte("hello"))
	trie.InsertOrUpdate([]byte("00000"), []byte("hi"))

	hash_hi := crypto.Keccak256Hash([]byte("hi"))
	hash_hello := crypto.Keccak256Hash([]byte("hello"))
	hex00200_00220 := crypto.Keccak256(append(hash_hi.Bytes(), hash_hello.Bytes()...))
	h := crypto.Keccak256Hash(append(append(hash_hi.Bytes(), hex00200_00220...), hash_hello.Bytes()...))

	assert.Equal(t, h.Hex(), trie.root.Hash().Hex())
}

func TestFixedLengthHexKeyMerkleTrie_Values(t *testing.T) {
	trie := New()

	trie.InsertOrUpdate([]byte("00000"), []byte("1"))
	trie.InsertOrUpdate([]byte("00200"), []byte("2"))
	trie.InsertOrUpdate([]byte("00300"), []byte("3"))
	trie.InsertOrUpdate([]byte("00220"), []byte("4"))

	values := trie.Values()

	for _, v := range values {
		log.Debug(string(v))
	}
}
