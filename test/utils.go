package test

import (
	"github.com/gagarinchain/network"
	"github.com/gagarinchain/network/mocks"
	"github.com/stretchr/testify/mock"
)

func SoftStorageMock() gagarinchain.Storage {
	storage := &mocks.Storage{}

	storage.On("Put", mock.AnythingOfType("gagarinchain.ResourceType"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return(nil)
	storage.On("Get", mock.AnythingOfType("gagarinchain.ResourceType"), mock.AnythingOfType("[]uint8")).Return(nil, nil)
	storage.On("Contains", mock.AnythingOfType("gagarinchain.ResourceType"), mock.AnythingOfType("[]uint8")).Return(false)
	storage.On("Delete", mock.AnythingOfType("gagarinchain.ResourceType"), mock.AnythingOfType("[]uint8")).Return(nil)
	storage.On("Keys", mock.AnythingOfType("gagarinchain.ResourceType"), mock.AnythingOfType("[]uint8")).Return(nil)

	return storage
}
