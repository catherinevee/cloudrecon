package unit

import (
	"testing"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestResourceCreation(t *testing.T) {
	resource := core.Resource{
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
	}

	assert.Equal(t, "test-resource-1", resource.ID)
	assert.Equal(t, "aws", resource.Provider)
	assert.Equal(t, "123456789012", resource.AccountID)
	assert.Equal(t, "us-east-1", resource.Region)
	assert.Equal(t, "ec2", resource.Service)
	assert.Equal(t, "instance", resource.Type)
	assert.Equal(t, "test-instance", resource.Name)
}

func TestAccountCreation(t *testing.T) {
	account := core.Account{
		ID:       "123456789012",
		Provider: "aws",
		Name:     "Test Account",
		Type:     "account",
		Tags: map[string]string{
			"environment": "test",
			"owner":       "test-team",
		},
	}

	assert.Equal(t, "123456789012", account.ID)
	assert.Equal(t, "aws", account.Provider)
	assert.Equal(t, "Test Account", account.Name)
	assert.Equal(t, "account", account.Type)
	assert.Equal(t, "test", account.Tags["environment"])
	assert.Equal(t, "test-team", account.Tags["owner"])
}

func TestDiscoveryMode(t *testing.T) {
	assert.Equal(t, 0, int(core.QuickMode))
	assert.Equal(t, 1, int(core.StandardMode))
	assert.Equal(t, 2, int(core.DeepMode))
}

func TestDiscoveryOptions(t *testing.T) {
	options := core.DiscoveryOptions{
		Mode:           core.StandardMode,
		Providers:      []string{"aws", "azure"},
		Accounts:       []string{"123456789012"},
		Regions:        []string{"us-east-1"},
		ResourceTypes:  []string{"EC2", "S3"},
		UseNativeTools: true,
		MaxParallel:    10,
		Timeout:        30 * time.Minute,
	}

	assert.Equal(t, core.StandardMode, options.Mode)
	assert.Equal(t, []string{"aws", "azure"}, options.Providers)
	assert.Equal(t, []string{"123456789012"}, options.Accounts)
	assert.Equal(t, []string{"us-east-1"}, options.Regions)
	assert.Equal(t, []string{"EC2", "S3"}, options.ResourceTypes)
	assert.True(t, options.UseNativeTools)
	assert.Equal(t, 10, options.MaxParallel)
	assert.Equal(t, 30*time.Minute, options.Timeout)
}

func TestResourceFilter(t *testing.T) {
	filter := core.ResourceFilter{
		Providers:    []string{"aws", "azure"},
		Accounts:     []string{"123456789012"},
		Regions:      []string{"us-east-1"},
		Services:     []string{"ec2", "s3"},
		Types:        []string{"instance", "bucket"},
		PublicAccess: boolPtr(true),
		Encrypted:    boolPtr(false),
		MinCost:      float64Ptr(100.0),
		MaxCost:      float64Ptr(1000.0),
	}

	assert.Equal(t, []string{"aws", "azure"}, filter.Providers)
	assert.Equal(t, []string{"123456789012"}, filter.Accounts)
	assert.Equal(t, []string{"us-east-1"}, filter.Regions)
	assert.Equal(t, []string{"ec2", "s3"}, filter.Services)
	assert.Equal(t, []string{"instance", "bucket"}, filter.Types)
	assert.True(t, *filter.PublicAccess)
	assert.False(t, *filter.Encrypted)
	assert.Equal(t, 100.0, *filter.MinCost)
	assert.Equal(t, 1000.0, *filter.MaxCost)
}

func TestResourceSummary(t *testing.T) {
	summary := core.ResourceSummary{
		TotalResources:   100,
		ByProvider:       map[string]int{"aws": 60, "azure": 40},
		ByService:        map[string]int{"ec2": 30, "s3": 20, "vm": 40, "storage": 10},
		ByType:           map[string]int{"instance": 30, "bucket": 20, "virtualmachine": 40, "storageaccount": 10},
		ByRegion:         map[string]int{"us-east-1": 50, "us-west-2": 30, "eastus": 20},
		TotalCost:        5000.0,
		CostByProvider:   map[string]float64{"aws": 3000.0, "azure": 2000.0},
		SecurityIssues:   5,
		ComplianceIssues: 3,
	}

	assert.Equal(t, 100, summary.TotalResources)
	assert.Equal(t, 60, summary.ByProvider["aws"])
	assert.Equal(t, 40, summary.ByProvider["azure"])
	assert.Equal(t, 30, summary.ByService["ec2"])
	assert.Equal(t, 20, summary.ByService["s3"])
	assert.Equal(t, 30, summary.ByType["instance"])
	assert.Equal(t, 20, summary.ByType["bucket"])
	assert.Equal(t, 50, summary.ByRegion["us-east-1"])
	assert.Equal(t, 30, summary.ByRegion["us-west-2"])
	assert.Equal(t, 5000.0, summary.TotalCost)
	assert.Equal(t, 3000.0, summary.CostByProvider["aws"])
	assert.Equal(t, 2000.0, summary.CostByProvider["azure"])
	assert.Equal(t, 5, summary.SecurityIssues)
	assert.Equal(t, 3, summary.ComplianceIssues)
}

func TestResourceTemplate(t *testing.T) {
	template := core.ResourceTemplate{
		Name:        "public_resources",
		Description: "Find all public resources",
		SQL:         "SELECT * FROM resources WHERE public_access = true",
		Parameters: []core.TemplateParam{
			{
				Name:        "min_cost",
				Type:        "float",
				Description: "Minimum monthly cost",
				Required:    false,
				Default:     0.0,
			},
		},
		Category: "security",
	}

	assert.Equal(t, "public_resources", template.Name)
	assert.Equal(t, "Find all public resources", template.Description)
	assert.Equal(t, "SELECT * FROM resources WHERE public_access = true", template.SQL)
	assert.Equal(t, 1, len(template.Parameters))
	assert.Equal(t, "min_cost", template.Parameters[0].Name)
	assert.Equal(t, "float", template.Parameters[0].Type)
	assert.Equal(t, "Minimum monthly cost", template.Parameters[0].Description)
	assert.False(t, template.Parameters[0].Required)
	assert.Equal(t, 0.0, template.Parameters[0].Default)
	assert.Equal(t, "security", template.Category)
}

func TestResourceRelationship(t *testing.T) {
	relationship := core.ResourceRelationship{
		SourceID:     "i-1234567890abcdef0",
		TargetID:     "sg-12345678",
		Relationship: "uses",
		Weight:       5,
	}

	assert.Equal(t, "i-1234567890abcdef0", relationship.SourceID)
	assert.Equal(t, "sg-12345678", relationship.TargetID)
	assert.Equal(t, "uses", relationship.Relationship)
	assert.Equal(t, 5, relationship.Weight)
}

func TestResourceMetrics(t *testing.T) {
	metrics := core.ResourceMetrics{
		ResourceID:     "i-1234567890abcdef0",
		CPUUtilization: 75.5,
		MemoryUsage:    60.2,
		NetworkIn:      1000.0,
		NetworkOut:     500.0,
		DiskUsage:      40.0,
		LastActivity:   time.Now(),
		Uptime:         99.9,
	}

	assert.Equal(t, "i-1234567890abcdef0", metrics.ResourceID)
	assert.Equal(t, 75.5, metrics.CPUUtilization)
	assert.Equal(t, 60.2, metrics.MemoryUsage)
	assert.Equal(t, 1000.0, metrics.NetworkIn)
	assert.Equal(t, 500.0, metrics.NetworkOut)
	assert.Equal(t, 40.0, metrics.DiskUsage)
	assert.Equal(t, 99.9, metrics.Uptime)
}

func TestResourceCost(t *testing.T) {
	cost := core.ResourceCost{
		ResourceID:   "i-1234567890abcdef0",
		Service:      "ec2",
		ResourceType: "instance",
		Region:       "us-east-1",
		MonthlyCost:  100.0,
		DailyCost:    3.33,
		HourlyCost:   0.14,
		Currency:     "USD",
		LastUpdated:  time.Now(),
		CostBreakdown: map[string]float64{
			"compute": 80.0,
			"storage": 20.0,
		},
	}

	assert.Equal(t, "i-1234567890abcdef0", cost.ResourceID)
	assert.Equal(t, "ec2", cost.Service)
	assert.Equal(t, "instance", cost.ResourceType)
	assert.Equal(t, "us-east-1", cost.Region)
	assert.Equal(t, 100.0, cost.MonthlyCost)
	assert.Equal(t, 3.33, cost.DailyCost)
	assert.Equal(t, 0.14, cost.HourlyCost)
	assert.Equal(t, "USD", cost.Currency)
	assert.Equal(t, 80.0, cost.CostBreakdown["compute"])
	assert.Equal(t, 20.0, cost.CostBreakdown["storage"])
}

func TestResourceSecurity(t *testing.T) {
	security := core.ResourceSecurity{
		ResourceID:       "i-1234567890abcdef0",
		PublicAccess:     false,
		Encrypted:        true,
		EncryptionKey:    "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		SecurityGroups:   []string{"sg-12345678", "sg-87654321"},
		IAMRoles:         []string{"EC2-Role", "S3-Role"},
		ComplianceStatus: "compliant",
		Vulnerabilities:  []string{"CVE-2023-1234"},
		LastScanned:      time.Now(),
	}

	assert.Equal(t, "i-1234567890abcdef0", security.ResourceID)
	assert.False(t, security.PublicAccess)
	assert.True(t, security.Encrypted)
	assert.Equal(t, "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", security.EncryptionKey)
	assert.Equal(t, 2, len(security.SecurityGroups))
	assert.Equal(t, "sg-12345678", security.SecurityGroups[0])
	assert.Equal(t, "sg-87654321", security.SecurityGroups[1])
	assert.Equal(t, 2, len(security.IAMRoles))
	assert.Equal(t, "EC2-Role", security.IAMRoles[0])
	assert.Equal(t, "S3-Role", security.IAMRoles[1])
	assert.Equal(t, "compliant", security.ComplianceStatus)
	assert.Equal(t, 1, len(security.Vulnerabilities))
	assert.Equal(t, "CVE-2023-1234", security.Vulnerabilities[0])
}

func TestResourceCompliance(t *testing.T) {
	compliance := core.ResourceCompliance{
		ResourceID: "i-1234567890abcdef0",
		Standards:  []string{"CIS", "SOC2", "PCI-DSS"},
		Status: map[string]string{
			"CIS":     "compliant",
			"SOC2":    "non-compliant",
			"PCI-DSS": "not-applicable",
		},
		LastAudit: time.Now(),
		Auditor:   "security-team",
		Findings: []core.ComplianceFinding{
			{
				ID:          "finding-1",
				Standard:    "SOC2",
				Rule:        "encryption-at-rest",
				Severity:    "high",
				Description: "Data not encrypted at rest",
				Remediation: "Enable encryption for all data stores",
				Status:      "fail",
			},
		},
	}

	assert.Equal(t, "i-1234567890abcdef0", compliance.ResourceID)
	assert.Equal(t, 3, len(compliance.Standards))
	assert.Equal(t, "CIS", compliance.Standards[0])
	assert.Equal(t, "SOC2", compliance.Standards[1])
	assert.Equal(t, "PCI-DSS", compliance.Standards[2])
	assert.Equal(t, "compliant", compliance.Status["CIS"])
	assert.Equal(t, "non-compliant", compliance.Status["SOC2"])
	assert.Equal(t, "not-applicable", compliance.Status["PCI-DSS"])
	assert.Equal(t, "security-team", compliance.Auditor)
	assert.Equal(t, 1, len(compliance.Findings))
	assert.Equal(t, "finding-1", compliance.Findings[0].ID)
	assert.Equal(t, "SOC2", compliance.Findings[0].Standard)
	assert.Equal(t, "encryption-at-rest", compliance.Findings[0].Rule)
	assert.Equal(t, "high", compliance.Findings[0].Severity)
	assert.Equal(t, "Data not encrypted at rest", compliance.Findings[0].Description)
	assert.Equal(t, "Enable encryption for all data stores", compliance.Findings[0].Remediation)
	assert.Equal(t, "fail", compliance.Findings[0].Status)
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}
