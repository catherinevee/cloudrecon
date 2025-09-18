package analysis

import (
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of the Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Initialize() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) StoreDiscovery(result *core.DiscoveryResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *MockStorage) GetResources(query string, args ...interface{}) ([]core.Resource, error) {
	argsMock := m.Called(query, args)
	return argsMock.Get(0).([]core.Resource), argsMock.Error(1)
}

func (m *MockStorage) GetDiscoveryStatus() (*core.DiscoveryStatus, error) {
	args := m.Called()
	return args.Get(0).(*core.DiscoveryStatus), args.Error(1)
}

func (m *MockStorage) Query(query string, args ...interface{}) (core.Rows, error) {
	argsMock := m.Called(query, args)
	return argsMock.Get(0).(core.Rows), argsMock.Error(1)
}

func (m *MockStorage) GetResourceCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockStorage) GetResourceSummary() (*core.ResourceSummary, error) {
	args := m.Called()
	return args.Get(0).(*core.ResourceSummary), args.Error(1)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}
