// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import tx "github.com/poslibp2p/common/tx"

// TransactionPool is an autogenerated mock type for the TransactionPool type
type TransactionPool struct {
	mock.Mock
}

// Add provides a mock function with given fields: _a0
func (_m *TransactionPool) Add(_a0 *tx.Transaction) {
	_m.Called(_a0)
}

// Iterator provides a mock function with given fields:
func (_m *TransactionPool) Iterator() tx.Iterator {
	ret := _m.Called()

	var r0 tx.Iterator
	if rf, ok := ret.Get(0).(func() tx.Iterator); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(tx.Iterator)
		}
	}

	return r0
}

// Remove provides a mock function with given fields: transaction
func (_m *TransactionPool) Remove(transaction *tx.Transaction) {
	_m.Called(transaction)
}

// RemoveAll provides a mock function with given fields: transactions
func (_m *TransactionPool) RemoveAll(transactions ...*tx.Transaction) {
	_va := make([]interface{}, len(transactions))
	for _i := range transactions {
		_va[_i] = transactions[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}