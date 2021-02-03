package hotstuff

import (
	"fmt"
	comm "github.com/gagarinchain/common"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/crypto"
	msg "github.com/gagarinchain/common/message"
	"github.com/gagarinchain/common/protobuff"
	bc "github.com/gagarinchain/network/blockchain"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type SyncImpl struct {
	cert            api.Certificate
	height          int32
	voting          bool
	sender          *comm.Peer
	signature       *crypto.Signature
	votingSignature *crypto.Signature
}

func (s *SyncImpl) Cert() api.Certificate {
	return s.cert
}

func (s *SyncImpl) Height() int32 {
	return s.height
}

func (s *SyncImpl) Voting() bool {
	return s.voting
}

func (s *SyncImpl) Signature() *crypto.Signature {
	return s.signature
}

func (s *SyncImpl) VotingSignature() *crypto.Signature {
	return s.votingSignature
}

func (s *SyncImpl) Sender() *comm.Peer {
	return s.sender
}

func CreateSync(height int32, voting bool, cert api.Certificate, sender *comm.Peer) *SyncImpl {
	return &SyncImpl{
		cert:   cert,
		height: height,
		voting: voting,
		sender: sender,
	}
}

func CreateSyncFromMessage(msg *msg.Message) (api.Sync, error) {
	if msg.Type != pb.Message_SYNCHRONIZE {
		return nil, errors.New(fmt.Sprintf("wrong message type, expected [%v], but got [%v]",
			pb.Message_SYNCHRONIZE.String(), msg.Type))
	}

	p := &pb.SynchronizePayload{}
	if err := ptypes.UnmarshalAny(msg.Payload, p); err != nil {
		return nil, err
	}

	var cert api.Certificate
	if p.GetQc() != nil {
		cert = bc.CreateQuorumCertificateFromMessage(p.GetQc())
	}

	if p.GetSc() != nil {
		cert = bc.CreateSynchronizeCertificateFromMessage(p.GetSc())
	}

	voting := p.VotingSignature != nil

	sign := crypto.SignatureFromProto(p.Signature)

	s := CreateSync(p.Height, voting, cert, msg.Source())
	s.signature = sign

	hash := bc.CalculateSyncHash(s.height, false)
	res := crypto.Verify(hash.Bytes(), sign)
	if !res {
		return nil, errors.New("bad signature")
	}

	a := crypto.PubkeyToAddress(crypto.NewPublicKey(sign.Pub()))
	msg.Source().SetAddress(a)
	msg.Source().SetPublicKey(crypto.NewPublicKey(sign.Pub()))

	if voting {
		vsign := crypto.SignatureFromProto(p.VotingSignature)
		s.votingSignature = vsign
		if vsign != nil {
			syncHash := bc.CalculateSyncHash(s.height, true)
			res := crypto.Verify(syncHash.Bytes(), vsign)
			if !res {
				return nil, errors.New("bad voting signature")
			}
		}
	}

	return s, nil
}

func (s *SyncImpl) Sign(key *crypto.PrivateKey) {
	hash := bc.CalculateSyncHash(s.height, false)
	s.signature = crypto.Sign(hash.Bytes(), key)
	if s.voting {
		hash := bc.CalculateSyncHash(s.height, true)
		s.votingSignature = crypto.Sign(hash.Bytes(), key)
	}
}

func (s *SyncImpl) GetMessage() *pb.SynchronizePayload {
	var vs *pb.Signature

	if s.votingSignature != nil {
		vs = s.votingSignature.ToProto()
	}

	p := &pb.SynchronizePayload{
		Signature:       s.signature.ToProto(),
		Height:          s.height,
		VotingSignature: vs,
	}

	if s.cert != nil {
		switch s.cert.Type() {
		case api.SC:
			sc := s.cert.(api.SynchronizeCertificate)
			p.Cert = &pb.SynchronizePayload_Sc{Sc: sc.GetMessage()}
		case api.QRef:
			fallthrough
		case api.Empty:
			sc := s.cert.(api.QuorumCertificate)
			p.Cert = &pb.SynchronizePayload_Qc{Qc: sc.GetMessage()}
		default:
			log.Error("unknown cert type")
			return nil
		}
	}

	return p
}
