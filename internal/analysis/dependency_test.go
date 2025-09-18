package analysis

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDependencyAnalyzer_AnalyzeDependencies(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	analyzer := NewDependencyAnalyzer(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:            "ec2-instance-1",
			Provider:      "aws",
			Service:       "ec2",
			Type:          "instance",
			Name:          "test-instance",
			ARN:           "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			Configuration: []byte(`{"SecurityGroupIds": ["sg-12345678"], "VpcId": "vpc-12345678"}`),
		},
		{
			ID:       "security-group-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "security-group",
			Name:     "test-sg",
			ARN:      "arn:aws:ec2:us-east-1:123456789012:security-group/sg-12345678",
		},
		{
			ID:       "vpc-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "vpc",
			Name:     "test-vpc",
			ARN:      "arn:aws:ec2:us-east-1:123456789012:vpc/vpc-12345678",
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	graph, err := analyzer.AnalyzeDependencies(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, 3, graph.Stats.TotalResources)
	assert.Greater(t, graph.Stats.TotalDependencies, 0)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestDependencyAnalyzer_AnalyzeAWSDependencies(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Create test resources
	resources := []core.Resource{
		{
			ID:            "ec2-instance-1",
			Provider:      "aws",
			Service:       "ec2",
			Type:          "instance",
			Name:          "test-instance",
			Configuration: []byte(`{"SecurityGroupIds": ["sg-12345678"], "VpcId": "vpc-12345678"}`),
		},
		{
			ID:       "security-group-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "security-group",
			Name:     "test-sg",
		},
		{
			ID:       "vpc-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "vpc",
			Name:     "test-vpc",
		},
	}

	// Execute test
	dependencies := analyzer.analyzeAWSDependencies(resources)

	// Assertions
	assert.NotNil(t, dependencies)
	assert.Greater(t, len(dependencies), 0)

	// Check for expected dependency types
	foundSecurityGroupDep := false
	foundVpcDep := false
	for _, dep := range dependencies {
		if dep.Relationship == "uses_security_group" {
			foundSecurityGroupDep = true
		}
		if dep.Relationship == "runs_in_vpc" {
			foundVpcDep = true
		}
	}
	assert.True(t, foundSecurityGroupDep, "Should find security group dependency")
	assert.True(t, foundVpcDep, "Should find VPC dependency")
}

func TestDependencyAnalyzer_AnalyzeEC2Dependencies(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Create test instance
	instance := core.Resource{
		ID:            "ec2-instance-1",
		Provider:      "aws",
		Service:       "ec2",
		Type:          "instance",
		Name:          "test-instance",
		ARN:           "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		Configuration: []byte(`{"SecurityGroupIds": ["sg-12345678"], "VpcId": "vpc-12345678"}`),
	}

	// Create resource map
	resourceMap := map[string]core.Resource{
		"security-group-1": {
			ID:       "security-group-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "security-group",
			Name:     "test-sg",
			ARN:      "arn:aws:ec2:us-east-1:123456789012:security-group/sg-12345678",
		},
		"vpc-1": {
			ID:       "vpc-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "vpc",
			Name:     "test-vpc",
			ARN:      "arn:aws:ec2:us-east-1:123456789012:vpc/vpc-12345678",
		},
	}

	// Execute test
	dependencies := analyzer.analyzeEC2Dependencies(instance, resourceMap)

	// Assertions
	assert.NotNil(t, dependencies)
	assert.Greater(t, len(dependencies), 0)

	// Check dependency properties
	for _, dep := range dependencies {
		assert.Equal(t, instance.ID, dep.SourceID)
		assert.Equal(t, instance.ARN, dep.SourceARN)
		assert.NotEmpty(t, dep.TargetID)
		assert.NotEmpty(t, dep.TargetARN)
		assert.NotEmpty(t, dep.Relationship)
		assert.NotEmpty(t, dep.Direction)
		assert.GreaterOrEqual(t, dep.Confidence, 0.0)
		assert.LessOrEqual(t, dep.Confidence, 1.0)
	}
}

func TestDependencyAnalyzer_CalculateGraphStats(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Create test data
	resources := []core.Resource{
		{ID: "resource-1", Provider: "aws"},
		{ID: "resource-2", Provider: "aws"},
		{ID: "resource-3", Provider: "azure"},
	}

	dependencies := []Dependency{
		{SourceID: "resource-1", TargetID: "resource-2", Relationship: "uses"},
		{SourceID: "resource-2", TargetID: "resource-3", Relationship: "depends_on"},
	}

	// Execute test
	stats := analyzer.calculateGraphStats(resources, dependencies)

	// Assertions
	assert.Equal(t, 3, stats.TotalResources)
	assert.Equal(t, 2, stats.TotalDependencies)
	assert.GreaterOrEqual(t, stats.MaxDepth, 0)
	assert.GreaterOrEqual(t, stats.Cycles, 0)
	assert.GreaterOrEqual(t, stats.Islands, 0)
}

func TestDependencyAnalyzer_CountIslands(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Create test data with some isolated resources
	resources := []core.Resource{
		{ID: "resource-1", Provider: "aws"},
		{ID: "resource-2", Provider: "aws"},
		{ID: "resource-3", Provider: "aws"},
		{ID: "resource-4", Provider: "aws"},
	}

	dependencies := []Dependency{
		{SourceID: "resource-1", TargetID: "resource-2", Relationship: "uses"},
		// resource-3 and resource-4 are isolated
	}

	// Execute test
	islands := analyzer.countIslands(resources, dependencies)

	// Assertions
	assert.Equal(t, 2, islands) // resource-3 and resource-4 should be isolated
}

func TestDependencyAnalyzer_GenerateGroupKey(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Test resource with tags
	resource := core.Resource{
		Provider: "aws",
		Service:  "ec2",
		Tags: map[string]string{
			"Environment": "production",
			"Project":     "cloudrecon",
		},
	}

	// Execute test
	key := analyzer.generateGroupKey(resource)

	// Assertions
	assert.Contains(t, key, "aws")
	assert.Contains(t, key, "ec2")
	assert.Contains(t, key, "production")
	assert.Contains(t, key, "cloudrecon")
}

func TestDependencyAnalyzer_GroupResourcesByPatterns(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Create test resources with similar patterns
	resources := []core.Resource{
		{
			ID:       "resource-1",
			Provider: "aws",
			Service:  "ec2",
			Tags: map[string]string{
				"Environment": "production",
				"Project":     "cloudrecon",
			},
		},
		{
			ID:       "resource-2",
			Provider: "aws",
			Service:  "ec2",
			Tags: map[string]string{
				"Environment": "production",
				"Project":     "cloudrecon",
			},
		},
		{
			ID:       "resource-3",
			Provider: "azure",
			Service:  "compute",
			Tags: map[string]string{
				"Environment": "staging",
				"Project":     "cloudrecon",
			},
		},
	}

	// Execute test
	groups := analyzer.groupResourcesByPatterns(resources)

	// Assertions
	assert.NotNil(t, groups)
	assert.Greater(t, len(groups), 0)

	// Check that resources with similar patterns are grouped together
	for _, group := range groups {
		assert.Greater(t, len(group), 1, "Groups should contain multiple resources")
	}
}

func TestDependencyAnalyzer_FindResourcesByPattern(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Create test resource map
	resourceMap := map[string]core.Resource{
		"security-group-1": {
			ID:       "security-group-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "security-group",
			Name:     "test-sg",
		},
		"vpc-1": {
			ID:       "vpc-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "vpc",
			Name:     "test-vpc",
		},
		"instance-1": {
			ID:       "instance-1",
			Provider: "aws",
			Service:  "ec2",
			Type:     "instance",
			Name:     "test-instance",
		},
	}

	// Execute test
	matches := analyzer.findResourcesByPattern(resourceMap, "security-group", "ec2")

	// Assertions
	assert.NotNil(t, matches)
	assert.Greater(t, len(matches), 0)

	// Check that all matches are security groups
	for _, match := range matches {
		assert.Equal(t, "ec2", match.Service)
		assert.Contains(t, strings.ToLower(match.Type), "security-group")
	}
}

func TestDependencyAnalyzer_SplitString(t *testing.T) {
	analyzer := NewDependencyAnalyzer(nil)

	// Test the helper function
	result := analyzer.splitString("a-b-c-d", "-")
	expected := []string{"a", "b", "c", "d"}
	assert.Equal(t, expected, result)

	// Test with different delimiter
	result = analyzer.splitString("a.b.c.d", ".")
	expected = []string{"a", "b", "c", "d"}
	assert.Equal(t, expected, result)

	// Test with no delimiter
	result = analyzer.splitString("abc", "-")
	expected = []string{"abc"}
	assert.Equal(t, expected, result)
}
