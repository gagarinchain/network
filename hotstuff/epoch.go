package hotstuff

import (
	"fmt"
	"github.com/gagarinchain/network/blockchain"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type Epoch struct {
	qc               *blockchain.QuorumCertificate
	genesisSignature []byte
	sender           *comm.Peer
	number           int32
}

func (ep *Epoch) Qc() *blockchain.QuorumCertificate {
	return ep.qc
}

func (ep *Epoch) Number() int32 {
	return ep.number
}

func (ep *Epoch) Sender() *comm.Peer {
	return ep.sender
}

func (ep *Epoch) GenesisSignature() []byte {
	return ep.genesisSignature
}

func CreateEpoch(sender *comm.Peer, number int32, qc *blockchain.QuorumCertificate, genesisSignature []byte) *Epoch {
	return &Epoch{qc, genesisSignature, sender, number}
}

func CreateEpochFromMessage(msg *msg.Message) (*Epoch, error) {
	if msg.Type != pb.Message_EPOCH_START {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_EPOCH_START.String(), msg.Type))
	}

	p := &pb.EpochStartPayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, p); err != nil {
		return nil, err
	}

	var ep *Epoch
	if cert := p.GetCert(); cert != nil {
		ep = CreateEpoch(msg.Source(), p.EpochNumber, blockchain.CreateQuorumCertificateFromMessage(cert), p.GetGenesisSignature())

	} else {
		ep = CreateEpoch(msg.Source(), p.EpochNumber, nil, p.GetGenesisSignature())
	}

	hash, e := CalculateHash(ep.createPayload())

	pub, e := crypto.SigToPub(hash.Bytes(), p.Signature)
	if e != nil {
		return nil, errors.New("bad signature")
	}
	a := crypto.PubkeyToAddress(*pub)
	msg.Source().SetAddress(a)

	return ep, nil
}

func CalculateHash(ep *pb.EpochStartPayload) (common.Hash, error) {
	payload := &pb.EpochStartPayload{EpochNumber: ep.EpochNumber, Body: &pb.EpochStartPayload_GenesisSignature{GenesisSignature: ep.Signature}}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return common.Hash{}, errors.Errorf("error while marshalling Payload", e)
	}
	hashbytes := common.BytesToHash(crypto.Keccak256(any.GetValue()))

	return hashbytes, e
}

func (ep *Epoch) GetMessage() (*msg.Message, error) {
	payload := ep.createPayload()
	hash, err := CalculateHash(payload)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(hash.Bytes(), ep.sender.GetPrivateKey())
	if err != nil {
		return nil, errors.Errorf("can'T sign sync message", err)
	}

	payload.Signature = sig
	any2, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, errors.Errorf("error while marshalling Payload", e)
	}

	return msg.CreateMessage(pb.Message_EPOCH_START, any2, ep.sender), nil
}

func (ep *Epoch) createPayload() *pb.EpochStartPayload {
	var payload *pb.EpochStartPayload
	if ep.genesisSignature != nil {
		payload = &pb.EpochStartPayload{
			Body:        &pb.EpochStartPayload_GenesisSignature{GenesisSignature: ep.genesisSignature},
			EpochNumber: ep.number,
		}
	} else {
		payload = &pb.EpochStartPayload{
			Body:        &pb.EpochStartPayload_Cert{Cert: ep.qc.GetMessage()},
			EpochNumber: ep.number,
		}
	}
	return payload
}
