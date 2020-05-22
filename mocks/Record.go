// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import api "github.com/gagarinchain/common/api"
import common "github.com/gagarinchain/common/eth/common"
import mock "github.com/stretchr/testify/mock"
import sparse "github.com/gagarinchain/common/trie/sparse"

// Record is an autogenerated mock type for the Record type
type Record struct {
	mock.Mock
}

// AddSibling provides a mock function with given fields: record
func (_m *Record) AddSibling(record api.Record) {
	_m.Called(record)
}

// ApplyTransaction provides a mock function with given fields: t
func (_m *Record) ApplyTransaction(t api.Transaction) error {
	ret := _m.Called(t)

	var r0 error
	if rf, ok := ret.Get(0).(func(api.Transaction) error); ok {
		r0 = rf(t)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: address
func (_m *Record) Get(address common.Address) (api.Account, bool) {
	ret := _m.Called(address)

	var r0 api.Account
	if rf, ok := ret.Get(0).(func(common.Address) api.Account); ok {
		r0 = rf(address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.Account)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(common.Address) bool); ok {
		r1 = rf(address)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// GetForUpdate provides a mock function with given fields: address
func (_m *Record) GetForUpdate(address common.Address) (api.Account, bool) {
	ret := _m.Called(address)

	var r0 api.Account
	if rf, ok := ret.Get(0).(func(common.Address) api.Account); ok {
		r0 = rf(address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.Account)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(common.Address) bool); ok {
		r1 = rf(address)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// Hash provides a mock function with given fields:
func (_m *Record) Hash() common.Hash {
	ret := _m.Called()

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func() common.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	return r0
}

// NewPendingRecord provides a mock function with given fields: proposer
func (_m *Record) NewPendingRecord(proposer common.Address) api.Record {
	ret := _m.Called(proposer)

	var r0 api.Record
	if rf, ok := ret.Get(0).(func(common.Address) api.Record); ok {
		r0 = rf(proposer)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.Record)
		}
	}

	return r0
}

// Parent provides a mock function with given fields:
func (_m *Record) Parent() api.Record {
	ret := _m.Called()

	var r0 api.Record
	if rf, ok := ret.Get(0).(func() api.Record); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.Record)
		}
	}

	return r0
}

// Pending provides a mock function with given fields:
func (_m *Record) Pending() api.Record {
	ret := _m.Called()

	var r0 api.Record
	if rf, ok := ret.Get(0).(func() api.Record); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.Record)
		}
	}

	return r0
}

// Proof provides a mock function with given fields: address
func (_m *Record) Proof(address common.Address) *sparse.Proof {
	ret := _m.Called(address)

	var r0 *sparse.Proof
	if rf, ok := ret.Get(0).(func(common.Address) *sparse.Proof); ok {
		r0 = rf(address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sparse.Proof)
		}
	}

	return r0
}

// RootProof provides a mock function with given fields:
func (_m *Record) RootProof() common.Hash {
	ret := _m.Called()

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func() common.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	return r0
}

// Serialize provides a mock function with given fields:
func (_m *Record) Serialize() []byte {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// SetHash provides a mock function with given fields: pending
func (_m *Record) SetHash(pending common.Hash) {
	_m.Called(pending)
}

// SetParent provides a mock function with given fields: parent
func (_m *Record) SetParent(parent api.Record) {
	_m.Called(parent)
}

// SetPending provides a mock function with given fields: pending
func (_m *Record) SetPending(pending api.Record) {
	_m.Called(pending)
}

// Siblings provides a mock function with given fields:
func (_m *Record) Siblings() []api.Record {
	ret := _m.Called()

	var r0 []api.Record
	if rf, ok := ret.Get(0).(func() []api.Record); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]api.Record)
		}
	}

	return r0
}

// Trie provides a mock function with given fields:
func (_m *Record) Trie() *sparse.SMT {
	ret := _m.Called()

	var r0 *sparse.SMT
	if rf, ok := ret.Get(0).(func() *sparse.SMT); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sparse.SMT)
		}
	}

	return r0
}
