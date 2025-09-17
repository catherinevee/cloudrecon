//go:build loadtest
// +build loadtest

package loadtest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/cloudrecon/cloudrecon/internal/storage"
)

func TestConcurrentDiscovery(t *testing.T) {
	// Skip if not running load tests
	if testing.Short() {
		t.Skip("Skipping load test")
	}

	// Initialize storage
	sqliteStorage, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteStorage.Close()

	// Initialize discovery orchestrator
	providers := make(map[string]core.CloudProvider)
	opts := core.DiscoveryOptions{
		Mode:        core.StandardMode,
		MaxParallel: 50,
		Timeout:     30 * time.Minute,
	}

	orchestrator := core.NewDiscoveryOrchestrator(providers, sqliteStorage, opts)

	// Test concurrent discovery operations
	numGoroutines := 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx := context.Background()
			result, err := orchestrator.Discover(ctx)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d failed: %w", id, err)
				return
			}

			if result == nil {
				errors <- fmt.Errorf("goroutine %d got nil result", id)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentStorageOperations(t *testing.T) {
	// Skip if not running load tests
	if testing.Short() {
		t.Skip("Skipping load test")
	}

	// Initialize storage
	sqliteStorage, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteStorage.Close()

	ctx := context.Background()

	// Test concurrent storage operations
	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create test resources
			resources := make([]core.Resource, 100)
			for j := 0; j < 100; j++ {
				resources[j] = core.Resource{
					ID:              fmt.Sprintf("test-resource-%d-%d", id, j),
					Provider:        "aws",
					AccountID:       "123456789012",
					Region:          "us-east-1",
					Service:         "ec2",
					Type:            "instance",
					Name:            fmt.Sprintf("test-instance-%d-%d", id, j),
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
					DiscoveredAt:    time.Now(),
					DiscoveryMethod: "direct_api",
				}
			}

			// Store resources
			err := sqliteStorage.StoreResources(ctx, resources)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d store failed: %w", id, err)
				return
			}

			// Retrieve resources
			_, err = sqliteStorage.GetResources("SELECT * FROM resources WHERE provider = ?", "aws")
			if err != nil {
				errors <- fmt.Errorf("goroutine %d retrieve failed: %w", id, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

func TestMemoryUsage(t *testing.T) {
	// Skip if not running load tests
	if testing.Short() {
		t.Skip("Skipping load test")
	}

	// Initialize storage
	sqliteStorage, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteStorage.Close()

	ctx := context.Background()

	// Create a large number of resources
	numResources := 10000
	resources := make([]core.Resource, numResources)

	for i := 0; i < numResources; i++ {
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
			Tags: map[string]string{
				"Environment": "test",
				"Project":     "cloudrecon",
				"Index":       fmt.Sprintf("%d", i),
			},
		}
	}

	// Store all resources
	err = sqliteStorage.StoreResources(ctx, resources)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve all resources
	retrievedResources, err := sqliteStorage.GetResources("SELECT * FROM resources")
	if err != nil {
		t.Fatal(err)
	}

	// Verify we got all resources back
	if len(retrievedResources) != numResources {
		t.Fatalf("Expected %d resources, got %d", numResources, len(retrievedResources))
	}
}
