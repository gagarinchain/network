package web3

import (
	"context"
	"fmt"
	"github.com/gagarinchain/common/eth/common"
	"github.com/gagarinchain/common/eth/common/hexutil"
	"math/big"
	"time"
)

type ApiSrv struct {
}

// ChainId returns the chainID value for transaction replay protection.
func (s *ApiSrv) ChainId() *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(12))
}

// BlockNumber returns the block number of the chain head.
func (s *ApiSrv) BlockNumber() hexutil.Uint64 {
	return hexutil.Uint64(66)
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The web3.LatestBlockNumber and web3.PendingBlockNumber meta
// block numbers are also allowed.
func (s *ApiSrv) GetBalance(ctx context.Context, address common.Address, blockNrOrHash BlockNumberOrHash) (*hexutil.Big, error) {
	return (*hexutil.Big)(big.NewInt(158972490234375000)), nil
}

// GasPrice returns a suggestion for a gas price.
func (s *ApiSrv) GasPrice(ctx context.Context) (*hexutil.Big, error) {

	return (*hexutil.Big)(big.NewInt(100)), nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *ApiSrv) GetCode(ctx context.Context, address common.Address, blockNrOrHash BlockNumberOrHash) (hexutil.Bytes, error) {
	return common.Hex2Bytes("0x600160008035811a818181146012578301005b601b6001356025565b8060005260206000f25b600060078202905091905056"), nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *ApiSrv) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash BlockNumberOrHash) (*hexutil.Uint64, error) {
	// Ask transaction pool for the nonce which includes pending transactions
	// Resolve block number and use its state to ask for the nonce
	u := uint64(10)
	return (*hexutil.Uint64)(&u), nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *ApiSrv) SendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	return common.Hash{}, nil
}

// GetBlockByNumber returns the requested canonical block.
// * When blockNr is -1 the chain head is returned.
// * When blockNr is -2 the pending chain head is returned.
// * When fullTx is true all transactions in the block are returned, otherwise
//   only the transaction hash is returned.
func (s *ApiSrv) GetBlockByNumber(ctx context.Context, number BlockNumber, fullTx bool) (map[string]interface{}, error) {
	//type Header struct {
	//	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	//	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	//	Coinbase    common.Address `json:"miner"            gencodec:"required"`
	//	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	//	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	//	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	//	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	//	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	//	Number      *big.Int       `json:"number"           gencodec:"required"`
	//	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	//	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	//	Time        uint64         `json:"timestamp"        gencodec:"required"`
	//	Extra       []byte         `json:"extraData"        gencodec:"required"`
	//	MixDigest   common.Hash    `json:"mixHash"`
	//	Nonce       BlockNonce     `json:"nonce"`
	//}
	block := map[string]interface{}{
		"number":           (*hexutil.Big)(big.NewInt(number.Int64())),
		"hash":             common.Hash{},
		"parentHash":       common.Hash{},
		"nonce":            100,
		"mixHash":          common.Hash{},
		"sha3Uncles":       common.Hash{},
		"logsBloom":        [2048]byte{},
		"stateRoot":        common.Hash{},
		"miner":            common.Address{},
		"difficulty":       (*hexutil.Big)(big.NewInt(20)),
		"extraData":        hexutil.Bytes{},
		"size":             hexutil.Uint64(0),
		"gasLimit":         hexutil.Uint64(100),
		"gasUsed":          hexutil.Uint64(123),
		"timestamp":        hexutil.Uint64(time.Now().Unix()),
		"transactionsRoot": common.Hash{},
		"receiptsRoot":     common.Hash{},
	}
	block["size"] = 0

	return block, nil

}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *ApiSrv) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	fields := map[string]interface{}{
		"blockHash":         common.Hash{},
		"blockNumber":       hexutil.Uint64(0),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(0),
		"from":              common.Address{},
		"to":                common.Address{},
		"gasUsed":           hexutil.Uint64(0),
		"cumulativeGasUsed": hexutil.Uint64(0),
		"contractAddress":   nil,
		"logs":              nil,
		"logsBloom":         nil,
	}

	return fields, nil
}

// Version returns the current ethereum protocol version.
func (s *ApiSrv) Version() string {
	return fmt.Sprintf("%d", 1)
}

//// rpcMarshalBlock uses the generalized output filler, then adds the total difficulty field, which requires
//// a `PublicBlockchainAPI`.
//func (s *ApiSrv) rpcMarshalBlock(ctx context.Context, b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
//	fields, err := RPCMarshalBlock(b, inclTx, fullTx)
//	if err != nil {
//		return nil, err
//	}
//	if inclTx {
//		fields["totalDifficulty"] = (*hexutil.Big)(s.b.GetTd(ctx, b.Hash()))
//	}
//	return fields, err
//}
//
//// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
//// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
//// transaction hashes.
//func RPCMarshalBlock(block *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
//	fields := RPCMarshalHeader(block.Header())
//	fields["size"] = hexutil.Uint64(block.Size())
//
//	if inclTx {
//		formatTx := func(tx *types.Transaction) (interface{}, error) {
//			return tx.Hash(), nil
//		}
//		if fullTx {
//			formatTx = func(tx *types.Transaction) (interface{}, error) {
//				return newRPCTransactionFromBlockHash(block, tx.Hash()), nil
//			}
//		}
//		txs := block.Transactions()
//		transactions := make([]interface{}, len(txs))
//		var err error
//		for i, tx := range txs {
//			if transactions[i], err = formatTx(tx); err != nil {
//				return nil, err
//			}
//		}
//		fields["transactions"] = transactions
//	}
//	uncles := block.Uncles()
//	uncleHashes := make([]common.Hash, len(uncles))
//	for i, uncle := range uncles {
//		uncleHashes[i] = uncle.Hash()
//	}
//	fields["uncles"] = uncleHashes
//
//	return fields, nil
//}
//
//// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
//func newRPCTransactionFromBlockHash(b *types.Block, hash common.Hash) *RPCTransaction {
//	for idx, tx := range b.Transactions() {
//		if tx.Hash() == hash {
//			return newRPCTransactionFromBlockIndex(b, uint64(idx))
//		}
//	}
//	return nil
//}
//
//// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
//func newRPCTransactionFromBlockIndex(b *types.Block, index uint64) *RPCTransaction {
//	txs := b.Transactions()
//	if index >= uint64(len(txs)) {
//		return nil
//	}
//	return newRPCTransaction(txs[index], b.Hash(), b.NumberU64(), index)
//}
//
//// newRPCTransaction returns a transaction that will serialize to the RPC
//// representation, with the given location metadata set (if available).
//func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCTransaction {
//	var signer types.Signer = types.FrontierSigner{}
//	if tx.Protected() {
//		signer = types.NewEIP155Signer(tx.ChainId())
//	}
//	from, _ := types.Sender(signer, tx)
//	v, r, s := tx.RawSignatureValues()
//
//	result := &RPCTransaction{
//		From:     from,
//		Gas:      hexutil.Uint64(tx.Gas()),
//		GasPrice: (*hexutil.Big)(tx.GasPrice()),
//		Hash:     tx.Hash(),
//		Input:    hexutil.Bytes(tx.Data()),
//		Nonce:    hexutil.Uint64(tx.Nonce()),
//		To:       tx.To(),
//		Value:    (*hexutil.Big)(tx.Value()),
//		V:        (*hexutil.Big)(v),
//		R:        (*hexutil.Big)(r),
//		S:        (*hexutil.Big)(s),
//	}
//	if blockHash != (common.Hash{}) {
//		result.BlockHash = &blockHash
//		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
//		result.TransactionIndex = (*hexutil.Uint64)(&index)
//	}
//	return result
//}
//
//
//// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
//type RPCTransaction struct {
//	BlockHash        *common.Hash    `json:"blockHash"`
//	BlockNumber      *hexutil.Big    `json:"blockNumber"`
//	From             common.Address  `json:"from"`
//	Gas              hexutil.Uint64  `json:"gas"`
//	GasPrice         *hexutil.Big    `json:"gasPrice"`
//	Hash             common.Hash     `json:"hash"`
//	Input            hexutil.Bytes   `json:"input"`
//	Nonce            hexutil.Uint64  `json:"nonce"`
//	To               *common.Address `json:"to"`
//	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
//	Value            *hexutil.Big    `json:"value"`
//	V                *hexutil.Big    `json:"v"`
//	R                *hexutil.Big    `json:"r"`
//	S                *hexutil.Big    `json:"s"`
//}
//
//
//// RPCMarshalHeader converts the given header to the RPC output .
//func RPCMarshalHeader(head *types.Header) map[string]interface{} {
//	return map[string]interface{}{
//		"number":           (*hexutil.Big)(head.Number),
//		"hash":             head.Hash(),
//		"parentHash":       head.ParentHash,
//		"nonce":            head.Nonce,
//		"mixHash":          head.MixDigest,
//		"sha3Uncles":       head.UncleHash,
//		"logsBloom":        head.Bloom,
//		"stateRoot":        head.Root,
//		"miner":            head.Coinbase,
//		"difficulty":       (*hexutil.Big)(head.Difficulty),
//		"extraData":        hexutil.Bytes(head.Extra),
//		"size":             hexutil.Uint64(head.Size()),
//		"gasLimit":         hexutil.Uint64(head.GasLimit),
//		"gasUsed":          hexutil.Uint64(head.GasUsed),
//		"timestamp":        hexutil.Uint64(head.Time),
//		"transactionsRoot": head.TxHash,
//		"receiptsRoot":     head.ReceiptHash,
//	}
//}
