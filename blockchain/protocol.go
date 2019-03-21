package blockchain

import (
	"github.com/golang/protobuf/ptypes"
	msg "github.com/poslibp2p/message"
	"github.com/poslibp2p/message/protobuff"
	"github.com/poslibp2p/network"
)

type BlockProtocol struct {
	srv       network.Service
	bc        *Blockchain
	sync      Synchronizer
	topHeight int32
}

var Version int32 = 1

func CreateBlockProtocol(srv network.Service, bc *Blockchain, sync Synchronizer, topHeight int32) *BlockProtocol {
	return &BlockProtocol{srv: srv, bc: bc, sync: sync, topHeight: topHeight}
}

func (p *BlockProtocol) Bootstrap() {
	rq := &pb.Message{Type: pb.Message_HELLO_REQUEST}
	m := &msg.Message{Message: rq}
	resp := <-p.srv.SendRequestToRandomPeer(m)

	if resp.Type != pb.Message_HELLO_RESPONSE {
		log.Error("Not expected msg type i response to Hello")
	}

	h := &pb.HelloPayload{}
	if err := ptypes.UnmarshalAny(resp.Payload, h); err != nil {
		log.Error("Couldn't unmarshal response", err)
	}
	//TODO check here different equivocations, such as very high block heights etc

	if h.GetVersion() != Version {
		log.Error("Wrong version")
	}

	if h.GetTopBlockHeight() > p.topHeight {
		p.sync.RequestBlocks(p.topHeight, h.GetTopBlockHeight())
	}
}

func (p *BlockProtocol) OnBlockRequest(req *msg.Message) {

}

func (p *BlockProtocol) OnHello(req *msg.Message) {

}
