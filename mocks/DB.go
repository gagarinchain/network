// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import common "github.com/gagarinchain/network/common/eth/common"
import mock "github.com/stretchr/testify/mock"
import state "github.com/gagarinchain/network/blockchain/state"

// DB is an autogenerated mock type for the DB type
type DB struct {
	mock.Mock
}

// Commit provides a mock function with given fields: parent, pending
func (_m *DB) Commit(parent common.Hash, pending common.Hash) (*state.Snapshot, error) {
	ret := _m.Called(parent, pending)

	var r0 *state.Snapshot
	if rf, ok := ret.Get(0).(func(common.Hash, common.Hash) *state.Snapshot); ok {
		r0 = rf(parent, pending)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.Snapshot)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash, common.Hash) error); ok {
		r1 = rf(parent, pending)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Create provides a mock function with given fields: parent, proposer
func (_m *DB) Create(parent common.Hash, proposer common.Address) (*state.Snapshot, error) {
	ret := _m.Called(parent, proposer)

	var r0 *state.Snapshot
	if rf, ok := ret.Get(0).(func(common.Hash, common.Address) *state.Snapshot); ok {
		r0 = rf(parent, proposer)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.Snapshot)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash, common.Address) error); ok {
		r1 = rf(parent, proposer)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: hash
func (_m *DB) Get(hash common.Hash) (*state.Snapshot, bool) {
	ret := _m.Called(hash)

	var r0 *state.Snapshot
	if rf, ok := ret.Get(0).(func(common.Hash) *state.Snapshot); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.Snapshot)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(common.Hash) bool); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// Init provides a mock function with given fields: hash, seed
func (_m *DB) Init(hash common.Hash, seed *state.Snapshot) error {
	ret := _m.Called(hash, seed)

	var r0 error
	if rf, ok := ret.Get(0).(func(common.Hash, *state.Snapshot) error); ok {
		r0 = rf(hash, seed)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Release provides a mock function with given fields: blockHash
func (_m *DB) Release(blockHash common.Hash) error {
	ret := _m.Called(blockHash)

	var r0 error
	if rf, ok := ret.Get(0).(func(common.Hash) error); ok {
		r0 = rf(blockHash)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
