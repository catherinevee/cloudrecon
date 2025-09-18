package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPerformanceOptimizedDependencyAnalyzer_AnalyzeDependenciesOptimized(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	analyzer := NewPerformanceOptimizedDependencyAnalyzer(mockStorage, config)

	// Mock resources
	testResources := []core.Resource{
		{
			ID:       "resource1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "instance",
			Region:   "us-east-1",
		},
		{
			ID:       "resource2",
			Provider: "aws",
			Service:  "rds",
			Type:     "db-instance",
			Region:   "us-east-1",
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	graph, err := analyzer.AnalyzeDependenciesOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, 2, graph.Stats.TotalResources)
	assert.Equal(t, 0, graph.Stats.TotalDependencies) // No dependencies in test data

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedSecurityAnalyzer_AnalyzeSecurityOptimized(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	analyzer := NewPerformanceOptimizedSecurityAnalyzer(mockStorage, config)

	// Mock resources
	testResources := []core.Resource{
		{
			ID:           "resource1",
			Provider:     "aws",
			Service:      "ec2",
			Type:         "instance",
			Region:       "us-east-1",
			PublicAccess: true,
			Encrypted:    false,
		},
		{
			ID:           "resource2",
			Provider:     "aws",
			Service:      "s3",
			Type:         "bucket",
			Region:       "us-east-1",
			PublicAccess: true,
			Encrypted:    false,
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	report, err := analyzer.AnalyzeSecurityOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.True(t, len(report.Findings) > 0)          // Should find security issues
	assert.True(t, report.Summary.TotalFindings >= 2) // Should find at least 2 security issues

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedCostAnalyzer_AnalyzeCostOptimized(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	analyzer := NewPerformanceOptimizedCostAnalyzer(mockStorage, config)

	// Mock resources
	testResources := []core.Resource{
		{
			ID:            "resource1",
			Provider:      "aws",
			Service:       "ec2",
			Type:          "instance",
			Region:        "us-east-1",
			Configuration: []byte(`{"InstanceType": "t3.micro"}`),
		},
		{
			ID:            "resource2",
			Provider:      "aws",
			Service:       "rds",
			Type:          "db-instance",
			Region:        "us-east-1",
			Configuration: []byte(`{"DBInstanceClass": "db.t3.small"}`),
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	report, err := analyzer.AnalyzeCostOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 2, len(report.CostEstimates))
	assert.True(t, report.Summary.AverageMonthlyCost > 0)

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedAnalysisOrchestrator_AnalyzeAllOptimized(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	orchestrator := NewPerformanceOptimizedAnalysisOrchestrator(mockStorage, config)

	// Mock resources
	testResources := []core.Resource{
		{
			ID:            "resource1",
			Provider:      "aws",
			Service:       "ec2",
			Type:          "instance",
			Region:        "us-east-1",
			PublicAccess:  true,
			Encrypted:     false,
			Configuration: []byte(`{"InstanceType": "t3.micro"}`),
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	report, err := orchestrator.AnalyzeAllOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 1, report.Summary.TotalResources)
	assert.NotNil(t, report.Dependencies)
	assert.NotNil(t, report.Security)
	assert.NotNil(t, report.Cost)

	mockStorage.AssertExpectations(t)
}

func TestPerformanceConfig_Default(t *testing.T) {
	config := DefaultPerformanceConfig()

	assert.True(t, config.MaxWorkers > 0) // Should be runtime.NumCPU()
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 5*time.Minute, config.CacheTimeout)
	assert.True(t, config.EnableParallel)
	assert.True(t, config.MemoryOptimize)
	assert.False(t, config.EnableProfiling)
}

func TestPerformanceOptimizedDependencyAnalyzer_Cache(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	analyzer := NewPerformanceOptimizedDependencyAnalyzer(mockStorage, config)

	// Test cache stats
	stats := analyzer.GetCacheStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["cache_size"])

	// Test cache clear
	analyzer.ClearCache()
	stats = analyzer.GetCacheStats()
	assert.Equal(t, 0, stats["cache_size"])
}

func TestPerformanceOptimizedSecurityAnalyzer_Cache(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	analyzer := NewPerformanceOptimizedSecurityAnalyzer(mockStorage, config)

	// Test cache stats
	stats := analyzer.GetCacheStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["cache_size"])

	// Test cache clear
	analyzer.ClearCache()
	stats = analyzer.GetCacheStats()
	assert.Equal(t, 0, stats["cache_size"])
}

func TestPerformanceOptimizedCostAnalyzer_Cache(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	analyzer := NewPerformanceOptimizedCostAnalyzer(mockStorage, config)

	// Test cache stats
	stats := analyzer.GetCacheStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["cache_size"])

	// Test cache clear
	analyzer.ClearCache()
	stats = analyzer.GetCacheStats()
	assert.Equal(t, 0, stats["cache_size"])
}

func TestPerformanceOptimizedAnalysisOrchestrator_Cache(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	orchestrator := NewPerformanceOptimizedAnalysisOrchestrator(mockStorage, config)

	// Test cache stats
	stats := orchestrator.GetCacheStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["cache_size"])

	// Test cache clear
	orchestrator.ClearCache()
	stats = orchestrator.GetCacheStats()
	assert.Equal(t, 0, stats["cache_size"])

	// Test performance metrics
	metrics := orchestrator.GetPerformanceMetrics()
	assert.NotNil(t, metrics)
	assert.True(t, metrics["max_workers"].(int) > 0)
	assert.Equal(t, 100, metrics["batch_size"])
	assert.True(t, metrics["parallel_enabled"].(bool))
}

func TestPerformanceOptimizedDependencyAnalyzer_EmptyResources(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	analyzer := NewPerformanceOptimizedDependencyAnalyzer(mockStorage, config)

	// Mock empty resources
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return([]core.Resource{}, nil)

	ctx := context.Background()
	graph, err := analyzer.AnalyzeDependenciesOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, 0, graph.Stats.TotalResources)
	assert.Equal(t, 0, graph.Stats.TotalDependencies)

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedSecurityAnalyzer_EmptyResources(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	analyzer := NewPerformanceOptimizedSecurityAnalyzer(mockStorage, config)

	// Mock empty resources
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return([]core.Resource{}, nil)

	ctx := context.Background()
	report, err := analyzer.AnalyzeSecurityOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 0, report.Summary.TotalFindings)
	assert.Equal(t, 100.0, report.ComplianceScore)
	assert.Equal(t, 0.0, report.RiskScore)

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedCostAnalyzer_EmptyResources(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	analyzer := NewPerformanceOptimizedCostAnalyzer(mockStorage, config)

	// Mock empty resources
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return([]core.Resource{}, nil)

	ctx := context.Background()
	report, err := analyzer.AnalyzeCostOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 0, len(report.CostEstimates))
	assert.Equal(t, 0.0, report.Summary.AverageMonthlyCost)

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedAnalysisOrchestrator_EmptyResources(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	orchestrator := NewPerformanceOptimizedAnalysisOrchestrator(mockStorage, config)

	// Mock empty resources
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return([]core.Resource{}, nil)

	ctx := context.Background()
	report, err := orchestrator.AnalyzeAllOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 0, report.Summary.TotalResources)

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedDependencyAnalyzer_MultipleProviders(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	analyzer := NewPerformanceOptimizedDependencyAnalyzer(mockStorage, config)

	// Mock resources from multiple providers
	testResources := []core.Resource{
		{
			ID:       "aws-resource1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "instance",
			Region:   "us-east-1",
		},
		{
			ID:       "azure-resource1",
			Provider: "azure",
			Service:  "compute",
			Type:     "vm",
			Region:   "eastus",
		},
		{
			ID:       "gcp-resource1",
			Provider: "gcp",
			Service:  "compute",
			Type:     "instance",
			Region:   "us-central1",
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	graph, err := analyzer.AnalyzeDependenciesOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, 3, graph.Stats.TotalResources)
	assert.Equal(t, 0, graph.Stats.TotalDependencies) // No dependencies in test data

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedSecurityAnalyzer_MultipleProviders(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	analyzer := NewPerformanceOptimizedSecurityAnalyzer(mockStorage, config)

	// Mock resources from multiple providers
	testResources := []core.Resource{
		{
			ID:           "aws-resource1",
			Provider:     "aws",
			Service:      "ec2",
			Type:         "instance",
			Region:       "us-east-1",
			PublicAccess: true,
			Encrypted:    false,
		},
		{
			ID:           "azure-resource1",
			Provider:     "azure",
			Service:      "compute",
			Type:         "vm",
			Region:       "eastus",
			PublicAccess: true,
			Encrypted:    false,
		},
		{
			ID:           "gcp-resource1",
			Provider:     "gcp",
			Service:      "compute",
			Type:         "instance",
			Region:       "us-central1",
			PublicAccess: true,
			Encrypted:    false,
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	report, err := analyzer.AnalyzeSecurityOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.True(t, len(report.Findings) > 0)          // Should find security issues
	assert.True(t, report.Summary.TotalFindings >= 2) // Should find at least 2 security issues

	mockStorage.AssertExpectations(t)
}

func TestPerformanceOptimizedCostAnalyzer_MultipleProviders(t *testing.T) {
	mockStorage := &MockStorage{}
	config := DefaultPerformanceConfig()
	config.MaxWorkers = 2
	config.BatchSize = 10

	analyzer := NewPerformanceOptimizedCostAnalyzer(mockStorage, config)

	// Mock resources from multiple providers
	testResources := []core.Resource{
		{
			ID:            "aws-resource1",
			Provider:      "aws",
			Service:       "ec2",
			Type:          "instance",
			Region:        "us-east-1",
			Configuration: []byte(`{"InstanceType": "t3.micro"}`),
		},
		{
			ID:            "azure-resource1",
			Provider:      "azure",
			Service:       "compute",
			Type:          "vm",
			Region:        "eastus",
			Configuration: []byte(`{"Size": "Standard_B1s"}`),
		},
		{
			ID:            "gcp-resource1",
			Provider:      "gcp",
			Service:       "compute",
			Type:          "instance",
			Region:        "us-central1",
			Configuration: []byte(`{"MachineType": "e2-micro"}`),
		},
	}

	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	ctx := context.Background()
	report, err := analyzer.AnalyzeCostOptimized(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 3, len(report.CostEstimates))
	assert.True(t, report.Summary.AverageMonthlyCost > 0)

	mockStorage.AssertExpectations(t)
}
