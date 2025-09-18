package analysis

import (
	"context"
	"testing"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCostAnalyzer_AnalyzeCost(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	analyzer := NewCostAnalyzer(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:          "ec2-instance-1",
			Provider:    "aws",
			Service:     "ec2",
			Type:        "instance",
			Name:        "test-instance",
			ARN:         "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			MonthlyCost: 50.0,
		},
		{
			ID:          "s3-bucket-1",
			Provider:    "aws",
			Service:     "s3",
			Type:        "bucket",
			Name:        "test-bucket",
			ARN:         "arn:aws:s3:::test-bucket",
			MonthlyCost: 10.0,
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	report, err := analyzer.AnalyzeCost(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 60.0, report.TotalMonthlyCost)
	assert.Equal(t, 2.0, report.TotalDailyCost)
	assert.InDelta(t, 0.083, report.TotalHourlyCost, 0.001)
	assert.Equal(t, "USD", report.Currency)
	assert.NotNil(t, report.CostEstimates)

	// Debug output
	t.Logf("Optimizations: %+v", report.Optimizations)
	t.Logf("Optimizations is nil: %v", report.Optimizations == nil)

	assert.NotNil(t, report.Optimizations) // May be empty slice
	assert.NotNil(t, report.Summary)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestCostAnalyzer_CalculateResourceCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: Resource with existing monthly cost
	resource := core.Resource{
		ID:          "ec2-instance-1",
		Provider:    "aws",
		Service:     "ec2",
		Type:        "instance",
		MonthlyCost: 50.0,
	}

	estimate, err := analyzer.calculateResourceCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, 50.0, estimate.MonthlyCost)
	assert.Equal(t, 50.0/30, estimate.DailyCost)
	assert.Equal(t, 50.0/(30*24), estimate.HourlyCost)
	assert.Equal(t, "USD", estimate.Currency)
	assert.Equal(t, 0.8, estimate.Confidence)

	// Test case 2: Resource without existing cost (AWS)
	resource.MonthlyCost = 0
	resource.Provider = "aws"
	resource.Service = "ec2"
	resource.Configuration = []byte(`{"InstanceType":"t3.medium"}`)

	estimate, err = analyzer.calculateResourceCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Greater(t, estimate.MonthlyCost, 0.0)
	assert.Equal(t, "aws", estimate.Provider)
	assert.Equal(t, "ec2", estimate.Service)

	// Test case 3: Resource without existing cost (Azure)
	resource.Provider = "azure"
	resource.Service = "compute"

	estimate, err = analyzer.calculateResourceCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Greater(t, estimate.MonthlyCost, 0.0)
	assert.Equal(t, "azure", estimate.Provider)
	assert.Equal(t, "compute", estimate.Service)

	// Test case 4: Resource without existing cost (GCP)
	resource.Provider = "gcp"
	resource.Service = "compute"

	estimate, err = analyzer.calculateResourceCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Greater(t, estimate.MonthlyCost, 0.0)
	assert.Equal(t, "gcp", estimate.Provider)
	assert.Equal(t, "compute", estimate.Service)

	// Test case 5: Unknown provider
	resource.Provider = "unknown"

	estimate, err = analyzer.calculateResourceCost(resource)
	assert.Error(t, err)
	assert.Nil(t, estimate)
}

func TestCostAnalyzer_CalculateEC2Cost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: t3.micro
	resource := core.Resource{
		Configuration: []byte(`{"InstanceType":"t3.micro"}`),
	}

	cost := analyzer.calculateEC2Cost(resource)
	assert.Equal(t, 8.0, cost)

	// Test case 2: t3.large
	resource.Configuration = []byte(`{"InstanceType":"t3.large"}`)
	cost = analyzer.calculateEC2Cost(resource)
	assert.Equal(t, 60.0, cost)

	// Test case 3: Unknown instance type
	resource.Configuration = []byte(`{"InstanceType":"unknown"}`)
	cost = analyzer.calculateEC2Cost(resource)
	assert.Equal(t, 8.0, cost) // Default cost (t3.micro)
}

func TestCostAnalyzer_CalculateRDSCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: db.t3.micro
	resource := core.Resource{
		Configuration: []byte(`{"DBInstanceClass":"db.t3.micro"}`),
	}

	cost := analyzer.calculateRDSCost(resource)
	assert.Equal(t, 15.0, cost)

	// Test case 2: db.m5.large
	resource.Configuration = []byte(`{"DBInstanceClass":"db.m5.large"}`)
	cost = analyzer.calculateRDSCost(resource)
	assert.Equal(t, 120.0, cost)

	// Test case 3: Unknown instance class
	resource.Configuration = []byte(`{"DBInstanceClass":"unknown"}`)
	cost = analyzer.calculateRDSCost(resource)
	assert.Equal(t, 15.0, cost) // Default cost (db.t3.micro)
}

func TestCostAnalyzer_CalculateS3Cost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	resource := core.Resource{}
	cost := analyzer.calculateS3Cost(resource)
	assert.Equal(t, 5.0, cost) // Default S3 cost
}

func TestCostAnalyzer_CalculateLambdaCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	resource := core.Resource{}
	cost := analyzer.calculateLambdaCost(resource)
	assert.Equal(t, 2.0, cost) // Default Lambda cost
}

func TestCostAnalyzer_CalculateAzureCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: Compute resource
	resource := core.Resource{
		Provider: "azure",
		Service:  "compute",
	}

	estimate, err := analyzer.calculateAzureCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, "azure", estimate.Provider)
	assert.Equal(t, "compute", estimate.Service)
	assert.Greater(t, estimate.MonthlyCost, 0.0)

	// Test case 2: Storage resource
	resource.Service = "storage"
	estimate, err = analyzer.calculateAzureCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, "storage", estimate.Service)
	assert.Greater(t, estimate.MonthlyCost, 0.0)

	// Test case 3: Unknown service
	resource.Service = "unknown"
	estimate, err = analyzer.calculateAzureCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, 15.0, estimate.MonthlyCost) // Default cost
}

func TestCostAnalyzer_CalculateGCPCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: Compute resource
	resource := core.Resource{
		Provider: "gcp",
		Service:  "compute",
	}

	estimate, err := analyzer.calculateGCPCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, "gcp", estimate.Provider)
	assert.Equal(t, "compute", estimate.Service)
	assert.Greater(t, estimate.MonthlyCost, 0.0)

	// Test case 2: Storage resource
	resource.Service = "storage"
	estimate, err = analyzer.calculateGCPCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, "storage", estimate.Service)
	assert.Greater(t, estimate.MonthlyCost, 0.0)

	// Test case 3: Unknown service
	resource.Service = "unknown"
	estimate, err = analyzer.calculateGCPCost(resource)
	assert.NoError(t, err)
	assert.NotNil(t, estimate)
	assert.Equal(t, 12.0, estimate.MonthlyCost) // Default cost
}

func TestCostAnalyzer_GenerateOptimizations(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Create test resources
	resources := []core.Resource{
		{
			ID:          "ec2-instance-1",
			Provider:    "aws",
			Service:     "ec2",
			Type:        "instance",
			Name:        "test-instance",
			MonthlyCost: 100.0,
		},
		{
			ID:          "s3-bucket-1",
			Provider:    "aws",
			Service:     "s3",
			Type:        "bucket",
			Name:        "test-bucket",
			MonthlyCost: 5.0,
		},
	}

	// Create test cost estimates
	estimates := []CostEstimate{
		{
			ResourceID:  "ec2-instance-1",
			MonthlyCost: 100.0,
		},
		{
			ResourceID:  "s3-bucket-1",
			MonthlyCost: 5.0,
		},
	}

	// Execute test
	optimizations := analyzer.generateOptimizations(resources, estimates)

	// Assertions
	assert.NotNil(t, optimizations)
	// Note: The actual number of optimizations depends on the logic in generateOptimizations
	// This test verifies the function runs without error and returns a valid slice
}

func TestCostAnalyzer_IsResourceUnused(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case: Resource (simplified implementation returns false)
	resource := core.Resource{
		ID: "test-resource",
	}

	unused := analyzer.isResourceUnused(resource)
	assert.False(t, unused) // Current implementation returns false
}

func TestCostAnalyzer_IsResourceOversized(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: Expensive EC2 instance
	resource := core.Resource{
		Service: "ec2",
	}
	estimate := CostEstimate{
		MonthlyCost: 100.0,
	}

	oversized := analyzer.isResourceOversized(resource, estimate)
	assert.True(t, oversized)

	// Test case 2: Cheap EC2 instance
	estimate.MonthlyCost = 10.0
	oversized = analyzer.isResourceOversized(resource, estimate)
	assert.False(t, oversized)

	// Test case 3: Non-EC2 resource
	resource.Service = "s3"
	estimate.MonthlyCost = 100.0
	oversized = analyzer.isResourceOversized(resource, estimate)
	assert.False(t, oversized)
}

func TestCostAnalyzer_IsReservedInstanceCandidate(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Test case 1: Expensive EC2 instance
	resource := core.Resource{
		Service: "ec2",
	}
	estimate := CostEstimate{
		MonthlyCost: 150.0,
	}

	candidate := analyzer.isReservedInstanceCandidate(resource, estimate)
	assert.True(t, candidate)

	// Test case 2: Cheap EC2 instance
	estimate.MonthlyCost = 50.0
	candidate = analyzer.isReservedInstanceCandidate(resource, estimate)
	assert.False(t, candidate)

	// Test case 3: Non-EC2 resource
	resource.Service = "s3"
	estimate.MonthlyCost = 150.0
	candidate = analyzer.isReservedInstanceCandidate(resource, estimate)
	assert.False(t, candidate)
}

func TestCostAnalyzer_CalculateCostSummary(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	// Create test data
	resources := []core.Resource{
		{ID: "resource-1", Provider: "aws"},
		{ID: "resource-2", Provider: "aws"},
		{ID: "resource-3", Provider: "azure"},
	}

	estimates := []CostEstimate{
		{ResourceID: "resource-1", Provider: "aws", Service: "ec2", MonthlyCost: 50.0},
		{ResourceID: "resource-2", Provider: "aws", Service: "s3", MonthlyCost: 10.0},
		{ResourceID: "resource-3", Provider: "azure", Service: "compute", MonthlyCost: 30.0},
	}

	// Execute test
	summary := analyzer.calculateCostSummary(resources, estimates)

	// Assertions
	assert.Equal(t, 3, summary.TotalResources)
	assert.Equal(t, 3, summary.ResourcesWithCost)
	assert.Equal(t, 30.0, summary.AverageMonthlyCost)          // (50+10+30)/3
	assert.Equal(t, "resource-1", summary.HighestCostResource) // 50.0
	assert.Equal(t, "resource-2", summary.LowestCostResource)  // 10.0
	assert.Equal(t, 60.0, summary.CostByProvider["aws"])       // 50+10
	assert.Equal(t, 30.0, summary.CostByProvider["azure"])     // 30
	assert.Equal(t, 50.0, summary.CostByService["ec2"])        // 50
	assert.Equal(t, 10.0, summary.CostByService["s3"])         // 10
	assert.Equal(t, 30.0, summary.CostByService["compute"])    // 30
}

func TestCostAnalyzer_CalculateAzureComputeCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	resource := core.Resource{}
	cost := analyzer.calculateAzureComputeCost(resource)
	assert.Equal(t, 25.0, cost) // Default Azure compute cost
}

func TestCostAnalyzer_CalculateAzureStorageCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	resource := core.Resource{}
	cost := analyzer.calculateAzureStorageCost(resource)
	assert.Equal(t, 8.0, cost) // Default Azure storage cost
}

func TestCostAnalyzer_CalculateGCPComputeCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	resource := core.Resource{}
	cost := analyzer.calculateGCPComputeCost(resource)
	assert.Equal(t, 20.0, cost) // Default GCP compute cost
}

func TestCostAnalyzer_CalculateGCPStorageCost(t *testing.T) {
	analyzer := NewCostAnalyzer(nil)

	resource := core.Resource{}
	cost := analyzer.calculateGCPStorageCost(resource)
	assert.Equal(t, 6.0, cost) // Default GCP storage cost
}
