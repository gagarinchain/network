package trie

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/gagarinchain/network/common/eth/crypto"
	"github.com/stretchr/testify/assert"
	strings2 "strings"
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
func TestFixedLengthHexKeyMerkleTrie_Values2(t *testing.T) {
	trie := New()

	strings := []string{
		strings2.ToLower("0xA8de0B33D7b815aC5c5B2524626A241bAf0FBA1f"),
		strings2.ToLower("0xA7bAf0EA7c4a76e33070231464841B206198467D"),
		strings2.ToLower("0x7C4bA1e923dEcDcC8C88beE16bBAB7fb305B0901"),
		strings2.ToLower("0xd9750628EbC61828649601FEBbAa71567C21635B"),
	}

	trie.InsertOrUpdate([]byte(strings[0]), []byte("1"))
	trie.InsertOrUpdate([]byte(strings[1]), []byte("2"))
	trie.InsertOrUpdate([]byte(strings[2]), []byte("3"))
	trie.InsertOrUpdate([]byte(strings[3]), []byte("4"))

	e := trie.Entries()
	spew.Dump(e)

	assert.Equal(t, trie.root.Key(), []byte("0x"))

}
