// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import api "github.com/gagarinchain/network/common/api"
import mock "github.com/stretchr/testify/mock"
import treemap "github.com/emirpasic/gods/maps/treemap"

// OnBlockCommit is an autogenerated mock type for the OnBlockCommit type
type OnBlockCommit struct {
	mock.Mock
}

// OnBlockCommit provides a mock function with given fields: bc, block, orphans
func (_m *OnBlockCommit) OnBlockCommit(bc api.Blockchain, block api.Block, orphans *treemap.Map) {
	_m.Called(bc, block, orphans)
}
