package hotstuff

import (
	"fmt"
	"github.com/gagarinchain/network/blockchain"
	comm "github.com/gagarinchain/network/common"
	"github.com/gagarinchain/network/common/api"
	"github.com/gagarinchain/network/common/eth/common"
	"github.com/gagarinchain/network/common/eth/crypto"
	msg "github.com/gagarinchain/network/common/message"
	"github.com/gagarinchain/network/common/protobuff"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type Epoch struct {
	qc               api.QuorumCertificate
	genesisSignature *crypto.Signature
	sender           *comm.Peer
	number           int32
}

func (ep *Epoch) Qc() api.QuorumCertificate {
	return ep.qc
}

func (ep *Epoch) Number() int32 {
	return ep.number
}

func (ep *Epoch) Sender() *comm.Peer {
	return ep.sender
}

func (ep *Epoch) GenesisSignature() *crypto.Signature {
	return ep.genesisSignature
}

func CreateEpoch(sender *comm.Peer, number int32, qc api.QuorumCertificate, genesisSignature *crypto.Signature) *Epoch {
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
		ep = CreateEpoch(msg.Source(), p.EpochNumber, blockchain.CreateQuorumCertificateFromMessage(cert), nil)

	} else {
		pbSign := p.GetGenesisSignature()
		signature := crypto.NewSignatureFromBytes(pbSign.From, pbSign.Signature)
		ep = CreateEpoch(msg.Source(), p.EpochNumber, nil, signature)
	}

	hash, e := CalculateHash(ep.createPayload())
	if e != nil {
		return nil, errors.New("bad hash")
	}
	sign := crypto.SignatureFromProto(p.Signature)
	res := crypto.Verify(hash.Bytes(), sign)
	if !res {
		return nil, errors.New("bad signature")
	}

	a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))
	msg.Source().SetPublicKey(crypto.NewPublicKey(sign.Pub()))
	msg.Source().SetAddress(a)

	return ep, nil
}

func CalculateHash(ep *pb.EpochStartPayload) (common.Hash, error) {
	var payload *pb.EpochStartPayload
	if ep.GetGenesisSignature() != nil {
		payload = &pb.EpochStartPayload{
			Body:        &pb.EpochStartPayload_GenesisSignature{GenesisSignature: ep.GetGenesisSignature()},
			EpochNumber: ep.GetEpochNumber(),
		}
	} else {
		payload = &pb.EpochStartPayload{
			Body:        &pb.EpochStartPayload_Cert{Cert: ep.GetCert()},
			EpochNumber: ep.GetEpochNumber(),
		}
	}
	any, e := ptypes.MarshalAny(payload)
	if e != nil {
		return common.Hash{}, errors.WithMessage(e, "error while marshalling Payload")
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

	sig := crypto.Sign(hash.Bytes(), ep.sender.GetPrivateKey())
	if sig == nil {
		return nil, errors.WithMessage(err, "can'T sign sync message")
	}

	payload.Signature = sig.ToProto()
	any2, e := ptypes.MarshalAny(payload)
	if e != nil {
		return nil, errors.WithMessage(e, "error while marshalling Payload")
	}

	return msg.CreateMessage(pb.Message_EPOCH_START, any2, ep.sender), nil
}

func (ep *Epoch) createPayload() *pb.EpochStartPayload {
	var payload *pb.EpochStartPayload
	if ep.genesisSignature != nil && !ep.genesisSignature.IsEmpty() {
		payload = &pb.EpochStartPayload{
			Body:        &pb.EpochStartPayload_GenesisSignature{GenesisSignature: ep.genesisSignature.ToProto()},
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
