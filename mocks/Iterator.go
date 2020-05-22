// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import api "github.com/gagarinchain/common/api"
import mock "github.com/stretchr/testify/mock"

// Iterator is an autogenerated mock type for the Iterator type
type Iterator struct {
	mock.Mock
}

// HasNext provides a mock function with given fields:
func (_m *Iterator) HasNext() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Next provides a mock function with given fields:
func (_m *Iterator) Next() api.Transaction {
	ret := _m.Called()

	var r0 api.Transaction
	if rf, ok := ret.Get(0).(func() api.Transaction); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.Transaction)
		}
	}

	return r0
}
