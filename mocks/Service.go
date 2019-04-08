// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import context "context"
import message "github.com/poslibp2p/message"
import mock "github.com/stretchr/testify/mock"

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// Broadcast provides a mock function with given fields: ctx, msg
func (_m *Service) Broadcast(ctx context.Context, msg *message.Message) {
	_m.Called(ctx, msg)
}

// SendMessage provides a mock function with given fields: ctx, peer, msg
func (_m *Service) SendMessage(ctx context.Context, peer *message.Peer, msg *message.Message) (chan *message.Message, chan error) {
	ret := _m.Called(ctx, peer, msg)

	var r0 chan *message.Message
	if rf, ok := ret.Get(0).(func(context.Context, *message.Peer, *message.Message) chan *message.Message); ok {
		r0 = rf(ctx, peer, msg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan *message.Message)
		}
	}

	var r1 chan error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Peer, *message.Message) chan error); ok {
		r1 = rf(ctx, peer, msg)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(chan error)
		}
	}

	return r0, r1
}

// SendMessageTriggered provides a mock function with given fields: ctx, peer, msg, trigger
func (_m *Service) SendMessageTriggered(ctx context.Context, peer *message.Peer, msg *message.Message, trigger chan interface{}) {
	_m.Called(ctx, peer, msg, trigger)
}

// SendRequestToRandomPeer provides a mock function with given fields: ctx, req
func (_m *Service) SendRequestToRandomPeer(ctx context.Context, req *message.Message) (chan *message.Message, chan error) {
	ret := _m.Called(ctx, req)

	var r0 chan *message.Message
	if rf, ok := ret.Get(0).(func(context.Context, *message.Message) chan *message.Message); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan *message.Message)
		}
	}

	var r1 chan error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Message) chan error); ok {
		r1 = rf(ctx, req)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(chan error)
		}
	}

	return r0, r1
}
