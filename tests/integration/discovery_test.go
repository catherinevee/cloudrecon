//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestDiscoveryIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use mock storage for integration test
	mockStorage := &MockStorage{}

	// Initialize discovery orchestrator
	providers := make(map[string]core.CloudProvider)
	opts := core.DiscoveryOptions{
		Mode:        core.StandardMode,
		MaxParallel: 5,
		Timeout:     5 * time.Minute,
	}

	orchestrator := core.NewDiscoveryOrchestrator(providers, mockStorage, opts)

	// Test discovery
	ctx := context.Background()
	result, err := orchestrator.Discover(ctx)

	// Should not error even with no providers
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStorageIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use mock storage for integration test
	mockStorage := &MockStorage{}

	ctx := context.Background()

	// Test storing resources
	resources := []core.Resource{
		{
			ID:              "test-resource-1",
			Provider:        "aws",
			AccountID:       "123456789012",
			Region:          "us-east-1",
			Service:         "ec2",
			Type:            "instance",
			Name:            "test-instance",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
		},
	}

	err := mockStorage.StoreResources(ctx, resources)
	assert.NoError(t, err)

	// Test retrieving resources
	retrievedResources, err := mockStorage.GetResources("SELECT * FROM resources WHERE provider = ?", "aws")
	assert.NoError(t, err)
	assert.Len(t, retrievedResources, 0) // Mock returns empty slice
}

// MockStorage for testing
type MockStorage struct{}

func (m *MockStorage) Initialize() error {
	return nil
}

func (m *MockStorage) StoreResources(ctx context.Context, resources []core.Resource) error {
	return nil
}

func (m *MockStorage) StoreDiscovery(result *core.DiscoveryResult) error {
	return nil
}

func (m *MockStorage) GetResources(query string, args ...interface{}) ([]core.Resource, error) {
	return []core.Resource{}, nil
}

func (m *MockStorage) Query(query string, args ...interface{}) (core.Rows, error) {
	return nil, nil
}

func (m *MockStorage) GetResource(ctx context.Context, id string) (*core.Resource, error) {
	return nil, nil
}

func (m *MockStorage) UpdateResource(ctx context.Context, resource core.Resource) error {
	return nil
}

func (m *MockStorage) DeleteResource(ctx context.Context, id string) error {
	return nil
}

func (m *MockStorage) GetResourceCount() (int, error) {
	return 0, nil
}

func (m *MockStorage) GetResourceSummary() (*core.ResourceSummary, error) {
	return &core.ResourceSummary{}, nil
}

func (m *MockStorage) GetDiscoveryStatus() (*core.DiscoveryStatus, error) {
	return &core.DiscoveryStatus{}, nil
}

func (m *MockStorage) Close() error {
	return nil
}
