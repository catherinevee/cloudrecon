//go:build performance
// +build performance

package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/cloudrecon/cloudrecon/internal/storage"
)

func BenchmarkDiscoveryOrchestrator(b *testing.B) {
	// Initialize storage
	sqliteStorage, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer sqliteStorage.Close()

	// Initialize discovery orchestrator
	providers := make(map[string]core.CloudProvider)
	opts := core.DiscoveryOptions{
		Mode:        core.StandardMode,
		MaxParallel: 10,
		Timeout:     30 * time.Minute,
	}

	orchestrator := core.NewDiscoveryOrchestrator(providers, sqliteStorage, opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := orchestrator.Discover(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStorageOperations(b *testing.B) {
	// Initialize storage
	sqliteStorage, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer sqliteStorage.Close()

	ctx := context.Background()

	// Create test resources
	resources := make([]core.Resource, 1000)
	for i := 0; i < 1000; i++ {
		resources[i] = core.Resource{
			ID:              fmt.Sprintf("test-resource-%d", i),
			Provider:        "aws",
			AccountID:       "123456789012",
			Region:          "us-east-1",
			Service:         "ec2",
			Type:            "instance",
			Name:            fmt.Sprintf("test-instance-%d", i),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test storing resources
		err := sqliteStorage.StoreResources(ctx, resources)
		if err != nil {
			b.Fatal(err)
		}

		// Test retrieving resources
		_, err = sqliteStorage.GetResources("SELECT * FROM resources WHERE provider = ?", "aws")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResourceCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource := core.Resource{
			ID:              fmt.Sprintf("test-resource-%d", i),
			Provider:        "aws",
			AccountID:       "123456789012",
			Region:          "us-east-1",
			Service:         "ec2",
			Type:            "instance",
			Name:            fmt.Sprintf("test-instance-%d", i),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
			Tags: map[string]string{
				"Environment": "test",
				"Project":     "cloudrecon",
			},
		}
		_ = resource // Prevent optimization
	}
}
