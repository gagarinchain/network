// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import common "github.com/poslibp2p/common"
import mock "github.com/stretchr/testify/mock"

// CommitteeLoader is an autogenerated mock type for the CommitteeLoader type
type CommitteeLoader struct {
	mock.Mock
}

// LoadPeerListFromFile provides a mock function with given fields: filePath
func (_m *CommitteeLoader) LoadFromFile(filePath string) []*common.Peer {
	ret := _m.Called(filePath)

	var r0 []*common.Peer
	if rf, ok := ret.Get(0).(func(string) []*common.Peer); ok {
		r0 = rf(filePath)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*common.Peer)
		}
	}

	return r0
}
