// Code generated by mockery v2.0.0-alpha.2. DO NOT EDIT.

package mocks

import (
	storage "github.com/gagarinchain/network/storage"
	mock "github.com/stretchr/testify/mock"
)

// Persistable is an autogenerated mock type for the Persistable type
type Persistable struct {
	mock.Mock
}

// Persist provides a mock function with given fields: _a0
func (_m *Persistable) Persist(_a0 storage.Storage) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(storage.Storage) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
