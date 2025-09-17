package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResource_ToJSON(t *testing.T) {
	resource := Resource{
		ID:              "test-resource-1",
		Provider:        "aws",
		AccountID:       "123456789012",
		Region:          "us-east-1",
		Service:         "ec2",
		Type:            "instance",
		Name:            "test-instance",
		ARN:             "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "direct_api",
		Tags: map[string]string{
			"Environment": "test",
			"Project":     "cloudrecon",
		},
		Configuration: []byte(`{"instanceType":"t3.micro"}`),
		PublicAccess:  false,
		Encrypted:     true,
		MonthlyCost:   10.50,
	}

	// Test basic resource properties
	assert.Equal(t, "test-resource-1", resource.ID)
	assert.Equal(t, "aws", resource.Provider)
	assert.Equal(t, "ec2", resource.Service)
	assert.Equal(t, "test-instance", resource.Name)
}

func TestResource_ToMap(t *testing.T) {
	resource := Resource{
		ID:              "test-resource-2",
		Provider:        "azure",
		AccountID:       "subscription-123",
		Region:          "eastus",
		Service:         "compute",
		Type:            "virtualMachine",
		Name:            "test-vm",
		ARN:             "/subscriptions/subscription-123/resourceGroups/rg-test/providers/Microsoft.Compute/virtualMachines/test-vm",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "azure_resource_graph",
		Tags: map[string]string{
			"Environment": "production",
			"Owner":       "team-alpha",
		},
		Configuration: []byte(`{"vmSize":"Standard_B1s"}`),
		PublicAccess:  true,
		Encrypted:     false,
		MonthlyCost:   25.75,
	}

	// Test basic resource properties
	assert.Equal(t, "test-resource-2", resource.ID)
	assert.Equal(t, "azure", resource.Provider)
	assert.Equal(t, "compute", resource.Service)
	assert.Equal(t, "test-vm", resource.Name)
	assert.Equal(t, true, resource.PublicAccess)
	assert.Equal(t, false, resource.Encrypted)
	assert.Equal(t, 25.75, resource.MonthlyCost)
}

func TestResource_Properties(t *testing.T) {
	now := time.Now()
	createdAt := now.AddDate(0, 0, -10) // 10 days ago
	updatedAt := now.AddDate(0, 0, -5)  // 5 days ago

	resource := Resource{
		ID:           "test-resource-3",
		Provider:     "gcp",
		AccountID:    "project-123",
		Region:       "us-central1",
		Service:      "compute",
		Type:         "instance",
		Name:         "test-instance",
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		DiscoveredAt: now,
		PublicAccess: false,
		Encrypted:    true,
		MonthlyCost:  100.0,
		Tags: map[string]string{
			"Environment": "production",
			"Team":        "platform",
		},
	}

	// Test basic properties
	assert.Equal(t, "test-resource-3", resource.ID)
	assert.Equal(t, "gcp", resource.Provider)
	assert.Equal(t, "project-123", resource.AccountID)
	assert.Equal(t, "us-central1", resource.Region)
	assert.Equal(t, "compute", resource.Service)
	assert.Equal(t, "instance", resource.Type)
	assert.Equal(t, "test-instance", resource.Name)
	assert.Equal(t, false, resource.PublicAccess)
	assert.Equal(t, true, resource.Encrypted)
	assert.Equal(t, 100.0, resource.MonthlyCost)

	// Test tags
	assert.Equal(t, "production", resource.Tags["Environment"])
	assert.Equal(t, "platform", resource.Tags["Team"])

	// Test timestamps
	assert.Equal(t, createdAt, resource.CreatedAt)
	assert.Equal(t, updatedAt, resource.UpdatedAt)
	assert.Equal(t, now, resource.DiscoveredAt)
}
