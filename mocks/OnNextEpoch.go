// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import api "github.com/gagarinchain/common/api"
import mock "github.com/stretchr/testify/mock"

// OnNextEpoch is an autogenerated mock type for the OnNextEpoch type
type OnNextEpoch struct {
	mock.Mock
}

// OnNewEpoch provides a mock function with given fields: pacer, bc, newEpoch
func (_m *OnNextEpoch) OnNewEpoch(pacer api.Pacer, bc api.Blockchain, newEpoch int32) {
	_m.Called(pacer, bc, newEpoch)
}
