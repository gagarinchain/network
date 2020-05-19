package network

import (
	"crypto/sha256"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type TopicCid struct {
	name string
}

func NewTopicCid(name string) *TopicCid {
	return &TopicCid{name: name}
}

func (t *TopicCid) CID() (id *cid.Cid, e error) {
	h := sha256.Sum256([]byte(t.name))
	encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
	if err != nil {
		return nil, err
	}

	mh, err := multihash.Cast(encoded)
	if err != nil {
		return nil, err
	}

	v1 := cid.NewCidV1(cid.Raw, mh)
	return &v1, nil
}
