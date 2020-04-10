package crypto

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptySignature(t *testing.T) {
	proto := EmptySignature().ToStorageProto()

	spew.Dump(proto)

	storage := SignatureFromStorage(proto)

	assert.True(t, storage.IsEmpty())
}
