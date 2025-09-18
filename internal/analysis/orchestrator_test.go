package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAnalysisOrchestrator_AnalyzeAll(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:          "ec2-instance-1",
			Provider:    "aws",
			Service:     "ec2",
			Type:        "instance",
			Name:        "test-instance",
			MonthlyCost: 50.0,
		},
		{
			ID:           "s3-bucket-1",
			Provider:     "aws",
			Service:      "s3",
			Type:         "bucket",
			Name:         "test-bucket",
			PublicAccess: true,
			Encrypted:    false,
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	report, err := orchestrator.AnalyzeAll(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.DependencyGraph)
	assert.NotNil(t, report.SecurityReport)
	assert.NotNil(t, report.CostReport)
	assert.NotNil(t, report.Summary)
	assert.Greater(t, report.Summary.TotalResources, 0)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestAnalysisOrchestrator_AnalyzeDependencies(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:       "ec2-instance-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "instance",
			Name:     "test-instance",
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	graph, err := orchestrator.AnalyzeDependencies(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, 1, graph.Stats.TotalResources)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestAnalysisOrchestrator_AnalyzeSecurity(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:           "s3-bucket-1",
			Provider:     "aws",
			Service:      "s3",
			Type:         "bucket",
			Name:         "test-bucket",
			PublicAccess: true,
			Encrypted:    false,
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	report, err := orchestrator.AnalyzeSecurity(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, report.Summary.TotalFindings, 0)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestAnalysisOrchestrator_AnalyzeCost(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:          "ec2-instance-1",
			Provider:    "aws",
			Service:     "ec2",
			Type:        "instance",
			Name:        "test-instance",
			MonthlyCost: 50.0,
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	report, err := orchestrator.AnalyzeCost(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 50.0, report.TotalMonthlyCost)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestAnalysisOrchestrator_GetAnalysisInsights(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:          "ec2-instance-1",
			Provider:    "aws",
			Service:     "ec2",
			Type:        "instance",
			Name:        "test-instance",
			MonthlyCost: 50.0,
		},
		{
			ID:           "s3-bucket-1",
			Provider:     "aws",
			Service:      "s3",
			Type:         "bucket",
			Name:         "test-bucket",
			PublicAccess: true,
			Encrypted:    false,
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	insights, err := orchestrator.GetAnalysisInsights(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, insights)
	assert.Greater(t, len(insights), 0)

	// Check for expected insight types
	insightText := ""
	for _, insight := range insights {
		insightText += insight
	}

	// Should contain resource count insight
	assert.Contains(t, insightText, "Discovered")
	assert.Contains(t, insightText, "resources")

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestAnalysisOrchestrator_ExportAnalysisReport(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:          "ec2-instance-1",
			Provider:    "aws",
			Service:     "ec2",
			Type:        "instance",
			Name:        "test-instance",
			MonthlyCost: 50.0,
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Test JSON export
	ctx := context.Background()
	jsonData, err := orchestrator.ExportAnalysisReport(ctx, "json")
	assert.NoError(t, err)
	assert.NotNil(t, jsonData)
	assert.Contains(t, string(jsonData), "message")

	// Test YAML export
	yamlData, err := orchestrator.ExportAnalysisReport(ctx, "yaml")
	assert.NoError(t, err)
	assert.NotNil(t, yamlData)
	assert.Contains(t, string(yamlData), "message")

	// Test CSV export
	csvData, err := orchestrator.ExportAnalysisReport(ctx, "csv")
	assert.NoError(t, err)
	assert.NotNil(t, csvData)
	assert.Contains(t, string(csvData), "message")

	// Test unsupported format
	_, err = orchestrator.ExportAnalysisReport(ctx, "unsupported")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestAnalysisOrchestrator_CalculateAnalysisSummary(t *testing.T) {
	orchestrator := NewAnalysisOrchestrator(nil)

	// Create test data
	dependencyGraph := &DependencyGraph{
		Stats: GraphStats{
			TotalResources:    10,
			TotalDependencies: 15,
		},
	}

	securityReport := &SecurityReport{
		Summary: SecuritySummary{
			TotalFindings:    5,
			CriticalFindings: 1,
		},
		ComplianceScore: 85.0,
		RiskScore:       25.0,
	}

	costReport := &CostReport{
		TotalMonthlyCost: 1000.0,
		PotentialSavings: 200.0,
	}

	duration := 5 * time.Second

	// Execute test
	summary := orchestrator.calculateAnalysisSummary(dependencyGraph, securityReport, costReport, duration)

	// Assertions
	assert.Equal(t, "5s", summary.AnalysisDuration)
	assert.Equal(t, 10, summary.TotalResources)
	assert.Equal(t, 15, summary.TotalDependencies)
	assert.Equal(t, 5, summary.SecurityFindings)
	assert.Equal(t, 1, summary.CriticalFindings)
	assert.Equal(t, 85.0, summary.ComplianceScore)
	assert.Equal(t, 25.0, summary.RiskScore)
	assert.Equal(t, 1000.0, summary.TotalMonthlyCost)
	assert.Equal(t, 200.0, summary.PotentialSavings)
}

func TestAnalysisOrchestrator_CalculateAnalysisSummaryWithNilValues(t *testing.T) {
	orchestrator := NewAnalysisOrchestrator(nil)

	// Execute test with nil values
	summary := orchestrator.calculateAnalysisSummary(nil, nil, nil, 0)

	// Assertions
	assert.Equal(t, "0s", summary.AnalysisDuration)
	assert.Equal(t, 0, summary.TotalResources)
	assert.Equal(t, 0, summary.TotalDependencies)
	assert.Equal(t, 0, summary.SecurityFindings)
	assert.Equal(t, 0, summary.CriticalFindings)
	assert.Equal(t, 0.0, summary.ComplianceScore)
	assert.Equal(t, 0.0, summary.RiskScore)
	assert.Equal(t, 0.0, summary.TotalMonthlyCost)
	assert.Equal(t, 0.0, summary.PotentialSavings)
}

func TestAnalysisOrchestrator_ExportJSON(t *testing.T) {
	orchestrator := NewAnalysisOrchestrator(nil)

	// Create test report
	report := &AnalysisReport{
		Timestamp: time.Now(),
		Summary: AnalysisSummary{
			TotalResources: 5,
		},
	}

	// Execute test
	data, err := orchestrator.exportJSON(report)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, string(data), "message")
	assert.Contains(t, string(data), "JSON export not implemented yet")
}

func TestAnalysisOrchestrator_ExportYAML(t *testing.T) {
	orchestrator := NewAnalysisOrchestrator(nil)

	// Create test report
	report := &AnalysisReport{
		Timestamp: time.Now(),
		Summary: AnalysisSummary{
			TotalResources: 5,
		},
	}

	// Execute test
	data, err := orchestrator.exportYAML(report)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, string(data), "message")
	assert.Contains(t, string(data), "YAML export not implemented yet")
}

func TestAnalysisOrchestrator_ExportCSV(t *testing.T) {
	orchestrator := NewAnalysisOrchestrator(nil)

	// Create test report
	report := &AnalysisReport{
		Timestamp: time.Now(),
		Summary: AnalysisSummary{
			TotalResources: 5,
		},
	}

	// Execute test
	data, err := orchestrator.exportCSV(report)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, string(data), "message")
	assert.Contains(t, string(data), "CSV export not implemented yet")
}

func TestNewAnalysisOrchestrator(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)

	// Execute test
	orchestrator := NewAnalysisOrchestrator(mockStorage)

	// Assertions
	assert.NotNil(t, orchestrator)
	assert.NotNil(t, orchestrator.storage)
	assert.NotNil(t, orchestrator.dependencyAnalyzer)
	assert.NotNil(t, orchestrator.securityAnalyzer)
	assert.NotNil(t, orchestrator.costAnalyzer)
}
