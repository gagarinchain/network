// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import message "github.com/gagarinchain/network/common/message"
import mock "github.com/stretchr/testify/mock"

// Dispatcher is an autogenerated mock type for the Dispatcher type
type Dispatcher struct {
	mock.Mock
}

// Dispatch provides a mock function with given fields: msg
func (_m *Dispatcher) Dispatch(msg *message.Message) {
	_m.Called(msg)
}
