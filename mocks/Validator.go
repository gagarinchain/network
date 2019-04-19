// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import pb "github.com/poslibp2p/common/protobuff"

// Validator is an autogenerated mock type for the Validator type
type Validator struct {
	mock.Mock
}

// GetId provides a mock function with given fields:
func (_m *Validator) GetId() interface{} {
	ret := _m.Called()

	var r0 interface{}
	if rf, ok := ret.Get(0).(func() interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// IsValid provides a mock function with given fields: entity
func (_m *Validator) IsValid(entity interface{}) (bool, error) {
	ret := _m.Called(entity)

	var r0 bool
	if rf, ok := ret.Get(0).(func(interface{}) bool); ok {
		r0 = rf(entity)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(interface{}) error); ok {
		r1 = rf(entity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Supported provides a mock function with given fields: mType
func (_m *Validator) Supported(mType pb.Message_MessageType) bool {
	ret := _m.Called(mType)

	var r0 bool
	if rf, ok := ret.Get(0).(func(pb.Message_MessageType) bool); ok {
		r0 = rf(mType)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
