package rpc

import (
	"context"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	pb "github.com/gagarinchain/common/protobuff"
	"github.com/gagarinchain/network/blockchain/state"
	"github.com/op/go-logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
)

var log = logging.MustGetLogger("rpc")

//TODO make pure libp2p service implementation
type Service struct {
	pb.CommonServiceServer
	bc    api.Blockchain
	pacer api.Pacer
	db    state.DB
}

type Config struct {
	Address              string
	MaxConcurrentStreams uint32
}

func (s *Service) GetBlockByHash(ctx context.Context, in *pb.GetBlockByHashRequest) (*pb.GetBlockByHashResponse, error) {
	block := s.bc.GetBlockByHash(common.BytesToHash(in.Hash))
	if block == nil {
		return nil, status.Error(codes.NotFound, "no block is found")
	}

	return &pb.GetBlockByHashResponse{
		Block: block.ToStorageProto(),
	}, nil
}

func (s *Service) GetBlocksByHeight(ctx context.Context, in *pb.GetBlockByHeightRequest) (*pb.GetBlockByHeightResponse, error) {
	blocks := s.bc.GetBlockByHeight(in.Height)
	return &pb.GetBlockByHeightResponse{
		Blocks: getProtoBlocks(blocks),
	}, nil
}

func getProtoBlocks(blocks []api.Block) []*pb.BlockS {
	var res []*pb.BlockS
	for _, b := range blocks {
		res = append(res, b.ToStorageProto())
	}
	return res
}

func (s *Service) GetFork(ctx context.Context, in *pb.GetForkRequest) (*pb.GetForkResponse, error) {
	blocks := s.bc.GetFork(in.Height, common.BytesToHash(in.HeadHash))
	return &pb.GetForkResponse{
		Blocks: getProtoBlocks(blocks),
	}, nil
}

func (s *Service) Contains(ctx context.Context, in *pb.ContainsRequest) (*pb.ContainsResponse, error) {
	contains := s.bc.Contains(common.BytesToHash(in.Hash))
	return &pb.ContainsResponse{
		Res: contains,
	}, nil
}

func (s *Service) GetThreeChain(ctx context.Context, in *pb.GetThreeChainRequest) (*pb.GetThreeChainResponse, error) {
	zero, one, two := s.bc.GetThreeChain(common.BytesToHash(in.Hash))
	var zeropb, onepb, twopb *pb.BlockS
	if zero != nil {
		zeropb = zero.ToStorageProto()
	}
	if one != nil {
		onepb = one.ToStorageProto()
	}
	if two != nil {
		twopb = two.ToStorageProto()
	}
	return &pb.GetThreeChainResponse{
		Zero: zeropb,
		One:  onepb,
		Two:  twopb,
	}, nil
}

func (s *Service) GetHead(ctx context.Context, in *pb.GetHeadRequest) (*pb.GetHeadResponse, error) {
	head := s.bc.GetHead()

	if head == nil {
		return nil, status.Error(codes.NotFound, "no head is found")
	}

	return &pb.GetHeadResponse{
		Block: head.ToStorageProto(),
	}, nil
}

func (s *Service) GetTopHeight(ctx context.Context, in *pb.GetTopHeightRequest) (*pb.GetTopHeightResponse, error) {
	topHeight := s.bc.GetTopHeight()

	return &pb.GetTopHeightResponse{
		Height: topHeight,
	}, nil
}

func (s *Service) GetTopHeightBlock(ctx context.Context, in *pb.GetTopHeightBlockRequest) (*pb.GetTopHeightBlockResponse, error) {
	head := s.bc.GetTopHeightBlocks()

	if len(head) > 1 {
		log.Warning("We can't have more than one block on top height, probably still loading from db or have huge protocol failure")
	} else if len(head) == 0 {
		return nil, status.Error(codes.NotFound, "no top height block is found")
	}
	return &pb.GetTopHeightBlockResponse{
		Block: head[0].ToStorageProto(),
	}, nil
}

func (s *Service) GetGenesisBlock(ctx context.Context, in *pb.GetGenesisBlockRequest) (*pb.GetGenesisBlockResponse, error) {
	genesis := s.bc.GetGenesisBlock()

	if genesis == nil {
		return nil, status.Error(codes.NotFound, "no genesis block is found")
	}

	return &pb.GetGenesisBlockResponse{
		Block: genesis.ToStorageProto(),
	}, nil
}

func (s *Service) IsSibling(ctx context.Context, in *pb.IsSiblingRequest) (*pb.IsSiblingResponse, error) {
	sibl := s.bc.GetBlockByHash(common.BytesToHash(in.SiblingHash))
	anc := s.bc.GetBlockByHash(common.BytesToHash(in.AncestorHash))
	if sibl == nil {
		return nil, status.Error(codes.NotFound, "no sibling block is found")
	}
	if anc == nil {
		return nil, status.Error(codes.NotFound, "no ancestor block is found")
	}

	sibling := s.bc.IsSibling(sibl.Header(), anc.Header())
	return &pb.IsSiblingResponse{
		Res: sibling,
	}, nil

}

func (s *Service) GetAccount(ctx context.Context, in *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	var version common.Hash
	if in.Hash == nil || len(in.Hash) == 0 {
		version = s.bc.GetTopCommittedBlock().Header().Hash()
	} else {
		version = common.BytesToHash(in.Hash)
	}

	get, f := s.db.Get(version)
	if !f {
		return nil, status.Error(codes.NotFound, "no record for given hash is found")
	}
	address := common.BytesToAddress(in.Address)
	acc, found := get.Get(address)
	if !found {
		return nil, status.Error(codes.NotFound, "no account found for given hash")
	}

	return &pb.GetAccountResponse{
		Account: acc.ToStorageProto(),
	}, nil

}

//TODO implement me
func (s *Service) GetTransaction(ctx context.Context, in *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not now")
}

func (s *Service) GetCommittee(ctx context.Context, in *pb.GetCommitteeRequest) (*pb.GetCommitteeResponse, error) {
	var peers []*pb.Peer
	for _, p := range s.pacer.GetPeers() {
		peers = append(peers, p.ToStorageProto())
	}

	return &pb.GetCommitteeResponse{
		Peer: peers,
	}, nil
}

func (s *Service) GetProposerForView(ctx context.Context, in *pb.GetProposerForViewRequest) (*pb.GetProposerForViewResponse, error) {
	proposer := s.pacer.ProposerForHeight(in.View)
	if proposer == nil {
		return nil, status.Error(codes.NotFound, "proposer is not found")
	}
	return &pb.GetProposerForViewResponse{
		Peer: proposer.ToStorageProto(),
	}, nil
}

func (s *Service) GetCurrentView(context.Context, *pb.GetCurrentViewRequest) (*pb.GetCurrentViewResponse, error) {
	return &pb.GetCurrentViewResponse{
		View: s.pacer.GetCurrentView(),
	}, nil
}

func (s *Service) GetCurrentEpoch(context.Context, *pb.GetCurrentEpochRequest) (*pb.GetCurrentEpochResponse, error) {
	return &pb.GetCurrentEpochResponse{
		Epoch: s.pacer.GetCurrentEpoch(),
	}, nil
}

func (s *Service) GetTopCommittedBlock(context.Context, *pb.GetTopCommittedBlockRequest) (*pb.GetTopCommittedBlockResponse, error) {
	block := s.bc.GetTopCommittedBlock()
	if block == nil {
		return nil, status.Error(codes.NotFound, "no committed block found")
	}

	return &pb.GetTopCommittedBlockResponse{
		Block: block.ToStorageProto(),
	}, nil
}

func NewService(bc api.Blockchain, pacer api.Pacer) *Service {
	return &Service{bc: bc, pacer: pacer}
}

func (s *Service) Bootstrap(cfg Config) error {
	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{grpc.MaxConcurrentStreams(cfg.MaxConcurrentStreams)}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterCommonServiceServer(grpcServer, s)
	if err := grpcServer.Serve(lis); err != nil {
		return err
	}
	return nil
}
