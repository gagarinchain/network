// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import pb "github.com/poslibp2p/message/protobuff"

// Validator is an autogenerated mock type for the Validator type
type Validator struct {
	mock.Mock
}

// Supported provides a mock function with given fields: mType
func (_m *Validator) Supported(mType *pb.Message_MessageType) bool {
	ret := _m.Called(mType)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*pb.Message_MessageType) bool); ok {
		r0 = rf(mType)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Validate provides a mock function with given fields: payload
func (_m *Validator) Validate(payload *pb.Message) bool {
	ret := _m.Called(payload)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*pb.Message) bool); ok {
		r0 = rf(payload)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
