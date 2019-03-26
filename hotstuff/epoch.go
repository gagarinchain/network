package hotstuff

import (
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"github.com/poslibp2p/blockchain"
	"github.com/poslibp2p/eth/common"
	"github.com/poslibp2p/eth/crypto"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
)

type Epoch struct {
	qc     *blockchain.QuorumCertificate
	sender *msg.Peer
	number int32
}

func CreateEpoch(sender *msg.Peer, number int32, qc *blockchain.QuorumCertificate) *Epoch {
	return &Epoch{qc, sender, number}
}

func CreateEpochFromMessage(msg *msg.Message, sender *msg.Peer) (*Epoch, error) {
	if msg.Type != pb.Message_EPOCH_START {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_EPOCH_START.String(), msg.Type))
	}

	ep := &pb.EpochStartPayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, ep); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}

	hashbytes, e := getHash(ep)

	pub, e := crypto.SigToPub(hashbytes, ep.Signature)
	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := common.BytesToAddress(crypto.FromECDSAPub(pub))
	sender.SetAddress(a)

	return CreateEpoch(sender, ep.EpochNumber, blockchain.CreateQuorumCertificateFromMessage(ep.Cert)), nil
}

func getHash(ep *pb.EpochStartPayload) ([]byte, error) {
	payload := &pb.EpochStartPayload{EpochNumber: ep.EpochNumber}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, errors.Errorf("error while marshalling payload", e)
	}
	hashbytes := crypto.Keccak256(any.GetValue())
	return hashbytes, e
}

func (ep *Epoch) GetMessage() (*msg.Message, error) {
	payload := &pb.EpochStartPayload{
		Cert:        ep.qc.GetMessage(),
		EpochNumber: ep.number,
	}

	hashbytes, err := getHash(payload)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(hashbytes, ep.sender.GetPrivateKey())
	if err != nil {
		return nil, errors.Errorf("can't sign sync message", err)
	}

	payload.Signature = sig
	any2, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, errors.Errorf("error while marshalling payload", e)
	}

	return msg.CreateMessage(pb.Message_EPOCH_START, any2), nil
}
