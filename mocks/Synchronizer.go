// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import blockchain "github.com/poslibp2p/blockchain"
import common "github.com/poslibp2p/eth/common"
import mock "github.com/stretchr/testify/mock"

// Synchronizer is an autogenerated mock type for the Synchronizer type
type Synchronizer struct {
	mock.Mock
}

// Bootstrap provides a mock function with given fields:
func (_m *Synchronizer) Bootstrap() {
	_m.Called()
}

// RequestBlock provides a mock function with given fields: hash, respChan
func (_m *Synchronizer) RequestBlock(hash common.Hash, respChan chan<- *blockchain.Block) {
	_m.Called(hash, respChan)
}

// RequestBlockWithParent provides a mock function with given fields: header
func (_m *Synchronizer) RequestBlockWithParent(header *blockchain.Header) {
	_m.Called(header)
}

// RequestBlocks provides a mock function with given fields: low, high
func (_m *Synchronizer) RequestBlocks(low int32, high int32) {
	_m.Called(low, high)
}

// RequestBlocksAtHeight provides a mock function with given fields: height, respChan
func (_m *Synchronizer) RequestBlocksAtHeight(height int32, respChan chan<- *blockchain.Block) {
	_m.Called(height, respChan)
}
