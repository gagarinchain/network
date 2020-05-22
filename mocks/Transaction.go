// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import api "github.com/gagarinchain/common/api"
import big "math/big"
import common "github.com/gagarinchain/common/eth/common"
import crypto "github.com/gagarinchain/common/eth/crypto"
import mock "github.com/stretchr/testify/mock"
import pb "github.com/gagarinchain/common/protobuff"

// Transaction is an autogenerated mock type for the Transaction type
type Transaction struct {
	mock.Mock
}

// CreateProof provides a mock function with given fields: pk
func (_m *Transaction) CreateProof(pk *crypto.PrivateKey) error {
	ret := _m.Called(pk)

	var r0 error
	if rf, ok := ret.Get(0).(func(*crypto.PrivateKey) error); ok {
		r0 = rf(pk)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Data provides a mock function with given fields:
func (_m *Transaction) Data() []byte {
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

// DropSignature provides a mock function with given fields:
func (_m *Transaction) DropSignature() {
	_m.Called()
}

// Fee provides a mock function with given fields:
func (_m *Transaction) Fee() *big.Int {
	ret := _m.Called()

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func() *big.Int); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	return r0
}

// From provides a mock function with given fields:
func (_m *Transaction) From() common.Address {
	ret := _m.Called()

	var r0 common.Address
	if rf, ok := ret.Get(0).(func() common.Address); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Address)
		}
	}

	return r0
}

// GetMessage provides a mock function with given fields:
func (_m *Transaction) GetMessage() *pb.Transaction {
	ret := _m.Called()

	var r0 *pb.Transaction
	if rf, ok := ret.Get(0).(func() *pb.Transaction); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Transaction)
		}
	}

	return r0
}

// Hash provides a mock function with given fields:
func (_m *Transaction) Hash() common.Hash {
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

// Nonce provides a mock function with given fields:
func (_m *Transaction) Nonce() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// RecoverProver provides a mock function with given fields:
func (_m *Transaction) RecoverProver() (*crypto.SignatureAggregate, error) {
	ret := _m.Called()

	var r0 *crypto.SignatureAggregate
	if rf, ok := ret.Get(0).(func() *crypto.SignatureAggregate); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*crypto.SignatureAggregate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Serialized provides a mock function with given fields:
func (_m *Transaction) Serialized() []byte {
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

// SetFrom provides a mock function with given fields: from
func (_m *Transaction) SetFrom(from common.Address) {
	_m.Called(from)
}

// SetTo provides a mock function with given fields: to
func (_m *Transaction) SetTo(to common.Address) {
	_m.Called(to)
}

// Sign provides a mock function with given fields: key
func (_m *Transaction) Sign(key *crypto.PrivateKey) {
	_m.Called(key)
}

// Signature provides a mock function with given fields:
func (_m *Transaction) Signature() *crypto.Signature {
	ret := _m.Called()

	var r0 *crypto.Signature
	if rf, ok := ret.Get(0).(func() *crypto.Signature); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*crypto.Signature)
		}
	}

	return r0
}

// To provides a mock function with given fields:
func (_m *Transaction) To() common.Address {
	ret := _m.Called()

	var r0 common.Address
	if rf, ok := ret.Get(0).(func() common.Address); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Address)
		}
	}

	return r0
}

// ToStorageProto provides a mock function with given fields:
func (_m *Transaction) ToStorageProto() *pb.TransactionS {
	ret := _m.Called()

	var r0 *pb.TransactionS
	if rf, ok := ret.Get(0).(func() *pb.TransactionS); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.TransactionS)
		}
	}

	return r0
}

// TxType provides a mock function with given fields:
func (_m *Transaction) TxType() api.Type {
	ret := _m.Called()

	var r0 api.Type
	if rf, ok := ret.Get(0).(func() api.Type); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(api.Type)
	}

	return r0
}

// Value provides a mock function with given fields:
func (_m *Transaction) Value() *big.Int {
	ret := _m.Called()

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func() *big.Int); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	return r0
}
