package test

import (
	"context"
	"errors"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gagarinchain/common/api"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/crypto"
	"github.com/gagarinchain/network/blockchain"
	"github.com/gagarinchain/network/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

//simply test chain loading
func TestSyncRequestBlocksSimple(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	headers, blocks := createChain(genesis, 25)

	for _, b := range blocks[1:] {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}
	for leftLimit := 0; leftLimit < 25; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 25 {
			rightLimit = 25
		}
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit), int32(rightLimit), mock.AnythingOfType("*common.Peer")).Return(headers[leftLimit+1:rightLimit+1], nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	_ = toTest.LoadBlocks(background, 0, 25, nil)

	assert.Equal(t, 26, len(bc.blocks))
}

//test loading fork we don't have lowest height block, so we query lower blocks until find common with blockchain
func TestSyncRequestBlocksWithNoHead(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	_, blocksLoaded := createChain(genesis, 20)
	headers, blocks := createChain(blocksLoaded[15], 10)

	for _, b := range blocksLoaded {
		bc.AddBlock(b)
	}
	for _, b := range blocks {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}

	for leftLimit := 0; leftLimit < 10; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 10 {
			rightLimit = 10
		}
		chunk := headers[leftLimit+1 : rightLimit+1]
		res := make([]api.Header, len(chunk))
		copy(res, chunk)
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit+15), int32(rightLimit+15), mock.AnythingOfType("*common.Peer")).Return(res, nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	_ = toTest.LoadBlocks(background, 19, 25, nil)

	for _, b := range bc.blocks {
		log.Debugf("%v", b.Height())
	}
	assert.Equal(t, 31, len(bc.blocks))
}

//we load fork but hit loading depth and fail
func TestSyncRequestBlocksWithNoHeadExceedDepthLimit(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}

	_, blocksLoaded := createChain(genesis, 20)
	headers, blocks := createChain(genesis, 25)

	for _, b := range blocksLoaded {
		bc.AddBlock(b)
	}
	for _, b := range blocks {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}
	for i := 23; i > 0; i -= headersLimit {
		rightLimit := i
		leftLimit := i - headersLimit

		if leftLimit < 0 {
			leftLimit = 0
		}

		chunk := headers[leftLimit:rightLimit]
		res := make([]api.Header, len(chunk))
		copy(res, chunk)
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit), int32(rightLimit), mock.AnythingOfType("*common.Peer")).Return(res, nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, 3, 20)

	background := context.Background()
	_ = toTest.LoadBlocks(background, 19, 25, nil)

	for _, b := range bc.blocks {
		log.Debugf("%v", b.Height())
	}
	assert.Equal(t, 21, len(bc.blocks))
}

// fail to load 16-th block
func TestSyncRequestBlocksNoBlockFound(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	headers, blocks := createChain(genesis, 25)

	for _, b := range blocks[1:] {
		if b.Height() == 16 {
			bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
				b.Header().Hash(),
				mock.AnythingOfType("*common.Peer")).Return(nil, errors.New("failed"))
			continue
		}
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}
	for leftLimit := 0; leftLimit < 25; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 25 {
			rightLimit = 25
		}
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit), int32(rightLimit), mock.AnythingOfType("*common.Peer")).Return(headers[leftLimit+1:rightLimit+1], nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	_ = toTest.LoadBlocks(background, 0, 25, nil)

	assert.Equal(t, 16, len(bc.blocks))
}

//fail to load 12-th header
func TestSyncRequestBlocksNoHeaderFound(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	headers, blocks := createChain(genesis, 25)

	for _, b := range blocks[1:] {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}
	for leftLimit := 0; leftLimit < 25; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 25 {
			rightLimit = 25
		}

		if leftLimit == 12 {
			bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
				int32(leftLimit), int32(rightLimit), mock.AnythingOfType("*common.Peer")).Return(nil, errors.New("failed"))
			continue
		}
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit), int32(rightLimit), mock.AnythingOfType("*common.Peer")).Return(headers[leftLimit+1:rightLimit+1], nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	_ = toTest.LoadBlocks(background, 0, 25, nil)

	assert.Equal(t, 13, len(bc.blocks))
}

//fail to load distinct header
func TestSyncRequestBlocksWithForkNoHeaderFound(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	_, blocksLoaded := createChain(genesis, 20)
	headers, blocks := createChain(blocksLoaded[15], 10)

	for _, b := range blocksLoaded {
		bc.AddBlock(b)
	}
	for _, b := range blocks {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}

	for leftLimit := 0; leftLimit < 10; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 10 {
			rightLimit = 10
		}
		chunk := headers[leftLimit+1 : rightLimit+1]
		res := make([]api.Header, len(chunk))
		copy(res, chunk)

		if leftLimit == 4 {
			bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
				int32(leftLimit+15), int32(rightLimit+15), mock.AnythingOfType("*common.Peer")).Return(nil, errors.New("failed"))
			continue
		}
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit+15), int32(rightLimit+15), mock.AnythingOfType("*common.Peer")).Return(res, nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	_ = toTest.LoadBlocks(background, 19, 25, nil)

	assert.Equal(t, 21, len(bc.blocks))
}

//simply load fork
func TestSyncRequestFork(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	_, blocksLoaded := createChain(genesis, 20)
	headers, blocks := createChain(blocksLoaded[17], 10)

	for _, b := range blocksLoaded {
		bc.AddBlock(b)
	}
	for _, b := range blocks {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}

	for leftLimit := 0; leftLimit < 10; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 10 {
			rightLimit = 10
		}
		chunk := headers[leftLimit+1 : rightLimit+1]
		res := make([]api.Header, len(chunk))
		copy(res, chunk)
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit+17), int32(rightLimit+17), mock.AnythingOfType("*common.Peer")).Return(res, nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	_ = toTest.LoadFork(background, headers[10].Height(), headers[10].Hash(), nil)

	for _, b := range bc.blocks {
		log.Debugf("%v", b.Height())
	}
	assert.Equal(t, 31, len(bc.blocks))
}

//load fork but don't find exact head, we load all blocks up to it and return error
func TestSyncRequestForkNoHead(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}
	_, blocksLoaded := createChain(genesis, 20)
	headers, blocks := createChain(blocksLoaded[17], 10)
	block27 := blockchain.CreateBlockWithParent(blocks[9])

	for _, b := range blocksLoaded {
		bc.AddBlock(b)
	}
	for _, b := range blocks {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}
	bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
		block27.Header().Hash(),
		mock.AnythingOfType("*common.Peer")).Return(block27, nil)

	for leftLimit := 0; leftLimit < 10; leftLimit += headersLimit {
		rightLimit := leftLimit + headersLimit
		if rightLimit > 10 {
			rightLimit = 10
		}
		chunk := headers[leftLimit+1 : rightLimit+1]
		if rightLimit == 10 {
			chunk2 := make([]api.Header, len(chunk))
			copy(chunk2, chunk)
			chunk2[len(chunk2)-1] = block27.Header()
			chunk = chunk2
		}
		res := make([]api.Header, len(chunk))
		copy(res, chunk)
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit+17), int32(rightLimit+17), mock.AnythingOfType("*common.Peer")).Return(res, nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, -1, 20)

	background := context.Background()
	e := toTest.LoadFork(background, headers[10].Height(), headers[10].Hash(), nil)

	for _, b := range bc.blocks {
		log.Debugf("%v", b.Height())
	}
	assert.Equal(t, 29, len(bc.blocks))
	assert.Error(t, e)
}

//load fork without common blocks
func TestSyncRequestForkNoCommonBlock(t *testing.T) {
	bsrv := &mocks.BlockService{}

	headersLimit := 4
	genesis := blockchain.CreateGenesisBlock()
	bc := &MockBlockchain{map[common.Hash]api.Block{genesis.Header().Hash(): genesis}}

	_, blocksLoaded := createChain(genesis, 20)
	headers, blocks := createChain(genesis, 25)

	for _, b := range blocksLoaded {
		bc.AddBlock(b)
	}
	for _, b := range blocks {
		bsrv.On("RequestBlock", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			b.Header().Hash(),
			mock.AnythingOfType("*common.Peer")).Return(b, nil)
	}
	for i := 21; i > 0; i -= headersLimit {
		rightLimit := i
		leftLimit := i - headersLimit

		if leftLimit < 0 {
			leftLimit = 0
		}

		chunk := headers[leftLimit:rightLimit]
		res := make([]api.Header, len(chunk))
		copy(res, chunk)
		bsrv.On("RequestHeaders", mock.MatchedBy(func(ctx context.Context) bool { return true }),
			int32(leftLimit), int32(rightLimit), mock.AnythingOfType("*common.Peer")).Return(res, nil)
	}

	toTest := blockchain.CreateSynchronizer(nil, bsrv, bc, -1, int32(headersLimit),
		3, 3, 3, 20)

	background := context.Background()
	_ = toTest.LoadFork(background, headers[21].Height(), headers[21].Hash(), nil)

	for _, b := range bc.blocks {
		log.Debugf("%v", b.Height())
	}
	assert.Equal(t, 21, len(bc.blocks))
}

func createChain(root api.Block, n int) ([]api.Header, []api.Block) {
	blocks := []api.Block{root}
	headers := []api.Header{root.Header()}
	for i := 1; i <= n; i++ {
		block := blockchain.CreateBlockWithParent(blocks[i-1])
		blocks = append(blocks, block)
		headers = append(headers, block.Header())
	}

	return headers, blocks
}

type MockBlockchain struct {
	blocks map[common.Hash]api.Block
}

func (m *MockBlockchain) GetBlockByHash(hash common.Hash) (block api.Block) {
	return m.blocks[hash]
}

func (m *MockBlockchain) GetBlockByHeight(height int32) (res []api.Block) {
	for _, b := range m.blocks {
		if b.Height() == height {
			res = append(res, b)
		}
	}

	return res
}

func (m *MockBlockchain) GetFork(height int32, headHash common.Hash) (res []api.Block) {
	panic("implement me")
}

func (m *MockBlockchain) GetBlockByHashOrLoad(ctx context.Context, hash common.Hash) (b api.Block, loaded bool) {
	b, loaded = m.blocks[hash]
	return b, loaded
}

func (m *MockBlockchain) LoadBlock(ctx context.Context, hash common.Hash) api.Block {
	panic("implement me")
}

func (m *MockBlockchain) Contains(hash common.Hash) bool {
	_, f := m.blocks[hash]
	return f
}

func (m *MockBlockchain) GetThreeChain(twoHash common.Hash) (zero api.Block, one api.Block, two api.Block) {
	panic("implement me")
}

func (m *MockBlockchain) OnCommit(b api.Block) (toCommit []api.Block, orphans *treemap.Map, err error) {
	panic("implement me")
}

func (m *MockBlockchain) GetHead() (res api.Block) {
	max := int32(0)

	for _, b := range m.blocks {
		if b.Height() > max {
			max = b.Height()
			res = b
		}
	}
	return res
}

func (m *MockBlockchain) GetHeadRecord() api.Record {
	panic("implement me")
}

func (m *MockBlockchain) GetTopHeight() int32 {
	return m.GetHead().Height()
}

func (m *MockBlockchain) GetTopHeightBlocks() (res []api.Block) {
	top := m.GetTopHeight()

	for _, b := range m.blocks {
		if b.Height() == top {
			res = append(res, b)
		}
	}
	return res
}

func (m *MockBlockchain) AddBlock(block api.Block) ([]api.Receipt, error) {
	_, f := m.blocks[block.Header().Parent()]
	if !f {
		return nil, errors.New("error")
	}
	m.blocks[block.Header().Hash()] = block
	return nil, nil
}

func (m *MockBlockchain) RemoveBlock(block api.Block) error {
	delete(m.blocks, block.Header().Hash())
	return nil
}

func (m *MockBlockchain) GetGenesisBlock() api.Block {
	panic("implement me")
}

func (m *MockBlockchain) GetGenesisCert() api.QuorumCertificate {
	panic("implement me")
}

func (m *MockBlockchain) IsSibling(sibling api.Header, ancestor api.Header) bool {
	panic("implement me")
}

func (m *MockBlockchain) NewBlock(parent api.Block, qc api.QuorumCertificate, data []byte) (api.Block, error) {
	panic("implement me")
}

func (m *MockBlockchain) PadEmptyBlock(head api.Block, qc api.QuorumCertificate) (api.Block, error) {
	panic("implement me")
}

func (m *MockBlockchain) GetGenesisBlockSignedHash(key *crypto.PrivateKey) *crypto.Signature {
	panic("implement me")
}

func (m *MockBlockchain) ValidateGenesisBlockSignature(signature *crypto.Signature, address common.Address) bool {
	panic("implement me")
}

func (m *MockBlockchain) GetTopCommittedBlock() api.Block {
	committedHeight := m.GetTopHeight() - 3

	if committedHeight < 0 {
		committedHeight = 0
	}
	return m.GetBlockByHeight(committedHeight)[0]
}

func (m *MockBlockchain) UpdateGenesisBlockQC(certificate api.QuorumCertificate) {
	panic("implement me")
}

func (m *MockBlockchain) SetProposerGetter(proposerGetter api.ProposerForHeight) {
	panic("implement me")
}
