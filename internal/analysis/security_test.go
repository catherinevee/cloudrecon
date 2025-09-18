package analysis

import (
	"context"
	"testing"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSecurityAnalyzer_AnalyzeSecurity(t *testing.T) {
	// Create mock storage
	mockStorage := new(MockStorage)
	analyzer := NewSecurityAnalyzer(mockStorage)

	// Create test resources
	testResources := []core.Resource{
		{
			ID:           "s3-bucket-1",
			Provider:     "aws",
			Service:      "s3",
			Type:         "bucket",
			Name:         "test-bucket",
			ARN:          "arn:aws:s3:::test-bucket",
			PublicAccess: true,
			Encrypted:    false,
			Compliance:   []string{"public-access", "unencrypted"},
		},
		{
			ID:           "ec2-instance-1",
			Provider:     "aws",
			Service:      "ec2",
			Type:         "instance",
			Name:         "test-instance",
			ARN:          "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			PublicAccess: true,
			Encrypted:    true,
			Compliance:   []string{"public-access"},
		},
	}

	// Set up mock expectations
	mockStorage.On("GetResources", "SELECT * FROM resources", mock.Anything).Return(testResources, nil)

	// Execute test
	ctx := context.Background()
	report, err := analyzer.AnalyzeSecurity(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Greater(t, report.Summary.TotalFindings, 0)
	assert.Greater(t, report.ComplianceScore, 0.0)
	assert.Greater(t, report.RiskScore, 0.0)

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

func TestSecurityAnalyzer_AnalyzeAWSSecurity(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Create test resources
	resources := []core.Resource{
		{
			ID:           "s3-bucket-1",
			Provider:     "aws",
			Service:      "s3",
			Type:         "bucket",
			Name:         "test-bucket",
			ARN:          "arn:aws:s3:::test-bucket",
			PublicAccess: true,
			Encrypted:    false,
			Compliance:   []string{"public-access", "unencrypted"},
		},
		{
			ID:           "ec2-instance-1",
			Provider:     "aws",
			Service:      "ec2",
			Type:         "instance",
			Name:         "test-instance",
			ARN:          "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			PublicAccess: true,
			Encrypted:    true,
			Compliance:   []string{"public-access"},
		},
		{
			ID:           "rds-instance-1",
			Provider:     "aws",
			Service:      "rds",
			Type:         "instance",
			Name:         "test-rds",
			ARN:          "arn:aws:rds:us-east-1:123456789012:db:test-db",
			PublicAccess: false,
			Encrypted:    false,
			Compliance:   []string{"unencrypted"},
		},
	}

	// Execute test
	findings := analyzer.analyzeAWSSecurity(resources)

	// Assertions
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for expected finding types
	foundPublicAccess := false
	foundEncryption := false
	for _, finding := range findings {
		if finding.Type == "public_access" {
			foundPublicAccess = true
		}
		if finding.Type == "encryption" {
			foundEncryption = true
		}
		assert.NotEmpty(t, finding.ID)
		assert.NotEmpty(t, finding.Title)
		assert.NotEmpty(t, finding.Description)
		assert.NotEmpty(t, finding.Recommendation)
		assert.Contains(t, []string{"critical", "high", "medium", "low", "info"}, finding.Severity)
	}
	assert.True(t, foundPublicAccess, "Should find public access issues")
	assert.True(t, foundEncryption, "Should find encryption issues")
}

func TestSecurityAnalyzer_AnalyzeEC2Security(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case 1: Public access issue
	instance := core.Resource{
		ID:           "ec2-instance-1",
		Provider:     "aws",
		Service:      "ec2",
		Type:         "instance",
		Name:         "test-instance",
		ARN:          "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		PublicAccess: true,
		Encrypted:    true,
		Compliance:   []string{"public-access"},
	}

	findings := analyzer.analyzeEC2Security(instance)
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for public access finding
	foundPublicAccess := false
	for _, finding := range findings {
		if finding.Type == "public_access" {
			foundPublicAccess = true
			assert.Equal(t, "high", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.1")
		}
	}
	assert.True(t, foundPublicAccess, "Should find public access issue")

	// Test case 2: Encryption issue
	instance.Encrypted = false
	instance.Compliance = []string{"unencrypted"}

	findings = analyzer.analyzeEC2Security(instance)
	foundEncryption := false
	for _, finding := range findings {
		if finding.Type == "encryption" {
			foundEncryption = true
			assert.Equal(t, "medium", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.2")
		}
	}
	assert.True(t, foundEncryption, "Should find encryption issue")
}

func TestSecurityAnalyzer_AnalyzeS3Security(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case 1: Public access issue
	bucket := core.Resource{
		ID:           "s3-bucket-1",
		Provider:     "aws",
		Service:      "s3",
		Type:         "bucket",
		Name:         "test-bucket",
		ARN:          "arn:aws:s3:::test-bucket",
		PublicAccess: true,
		Encrypted:    false,
		Compliance:   []string{"public-access", "unencrypted"},
	}

	findings := analyzer.analyzeS3Security(bucket)
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for public access finding
	foundPublicAccess := false
	for _, finding := range findings {
		if finding.Type == "public_access" {
			foundPublicAccess = true
			assert.Equal(t, "critical", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.1")
			assert.Contains(t, finding.Compliance, "SOC2-CC6.1")
			assert.Contains(t, finding.Compliance, "PCI-DSS-1.2")
		}
	}
	assert.True(t, foundPublicAccess, "Should find public access issue")

	// Check for encryption finding
	foundEncryption := false
	for _, finding := range findings {
		if finding.Type == "encryption" {
			foundEncryption = true
			assert.Equal(t, "high", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.2")
		}
	}
	assert.True(t, foundEncryption, "Should find encryption issue")
}

func TestSecurityAnalyzer_AnalyzeRDSSecurity(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case: Unencrypted RDS instance
	rds := core.Resource{
		ID:           "rds-instance-1",
		Provider:     "aws",
		Service:      "rds",
		Type:         "instance",
		Name:         "test-rds",
		ARN:          "arn:aws:rds:us-east-1:123456789012:db:test-db",
		PublicAccess: false,
		Encrypted:    false,
		Compliance:   []string{"unencrypted"},
	}

	findings := analyzer.analyzeRDSSecurity(rds)
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for encryption finding
	foundEncryption := false
	for _, finding := range findings {
		if finding.Type == "encryption" {
			foundEncryption = true
			assert.Equal(t, "high", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.2")
			assert.Contains(t, finding.Compliance, "SOC2-CC6.1")
			assert.Contains(t, finding.Compliance, "PCI-DSS-3.4")
		}
	}
	assert.True(t, foundEncryption, "Should find encryption issue")
}

func TestSecurityAnalyzer_AnalyzeIAMSecurity(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case: Overly permissive policy
	iam := core.Resource{
		ID:            "iam-policy-1",
		Provider:      "aws",
		Service:       "iam",
		Type:          "policy",
		Name:          "test-policy",
		ARN:           "arn:aws:iam::123456789012:policy/test-policy",
		Configuration: []byte(`{"Effect":"Allow","Resource":"*"}`),
	}

	findings := analyzer.analyzeIAMSecurity(iam)
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for overly permissive finding
	foundOverlyPermissive := false
	for _, finding := range findings {
		if finding.Type == "permissions" {
			foundOverlyPermissive = true
			assert.Equal(t, "high", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-1.16")
			assert.Contains(t, finding.Compliance, "SOC2-CC6.1")
		}
	}
	assert.True(t, foundOverlyPermissive, "Should find overly permissive policy")
}

func TestSecurityAnalyzer_AnalyzeLambdaSecurity(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case: Lambda not in VPC
	lambda := core.Resource{
		ID:            "lambda-function-1",
		Provider:      "aws",
		Service:       "lambda",
		Type:          "function",
		Name:          "test-function",
		ARN:           "arn:aws:lambda:us-east-1:123456789012:function:test-function",
		Configuration: []byte(`{"FunctionName":"test-function"}`), // No VpcConfig
	}

	findings := analyzer.analyzeLambdaSecurity(lambda)
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for VPC finding
	foundVpc := false
	for _, finding := range findings {
		if finding.Type == "network" {
			foundVpc = true
			assert.Equal(t, "medium", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.3")
		}
	}
	assert.True(t, foundVpc, "Should find VPC issue")
}

func TestSecurityAnalyzer_CalculateSecuritySummary(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Create test findings
	findings := []SecurityFinding{
		{Severity: "critical", Compliance: []string{"CIS-2.1"}},
		{Severity: "high", Compliance: []string{"CIS-2.2"}},
		{Severity: "medium", Compliance: []string{"CIS-2.3"}},
		{Severity: "low", Compliance: []string{"CIS-2.4"}},
		{Severity: "info", Compliance: []string{"CIS-2.5"}},
	}

	// Execute test
	summary := analyzer.calculateSecuritySummary(findings)

	// Assertions
	assert.Equal(t, 5, summary.TotalFindings)
	assert.Equal(t, 1, summary.CriticalFindings)
	assert.Equal(t, 1, summary.HighFindings)
	assert.Equal(t, 1, summary.MediumFindings)
	assert.Equal(t, 1, summary.LowFindings)
	assert.Equal(t, 1, summary.InfoFindings)
	assert.Equal(t, 0, summary.CompliancePass) // No resources passed
	assert.Equal(t, 5, summary.ComplianceFail) // All findings failed
}

func TestSecurityAnalyzer_CalculateComplianceScore(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case 1: No findings
	score := analyzer.calculateComplianceScore([]SecurityFinding{})
	assert.Equal(t, 100.0, score)

	// Test case 2: Some findings
	findings := []SecurityFinding{
		{Severity: "critical"},
		{Severity: "high"},
		{Severity: "medium"},
	}

	score = analyzer.calculateComplianceScore(findings)
	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestSecurityAnalyzer_CalculateRiskScore(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Test case 1: No findings
	score := analyzer.calculateRiskScore([]SecurityFinding{})
	assert.Equal(t, 0.0, score)

	// Test case 2: Some findings
	findings := []SecurityFinding{
		{Severity: "critical"},
		{Severity: "high"},
		{Severity: "medium"},
	}

	score = analyzer.calculateRiskScore(findings)
	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestSecurityAnalyzer_GroupResourcesByPatterns(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

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

func TestSecurityAnalyzer_GenerateGroupKey(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

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

func TestSecurityAnalyzer_AnalyzeSecurityConsistency(t *testing.T) {
	analyzer := NewSecurityAnalyzer(nil)

	// Create test group with inconsistent encryption
	group := []core.Resource{
		{
			ID:        "resource-1",
			Provider:  "aws",
			Service:   "ec2",
			Encrypted: true,
		},
		{
			ID:        "resource-2",
			Provider:  "aws",
			Service:   "ec2",
			Encrypted: false,
		},
		{
			ID:        "resource-3",
			Provider:  "aws",
			Service:   "ec2",
			Encrypted: false,
		},
	}

	// Execute test
	findings := analyzer.analyzeSecurityConsistency(group)

	// Assertions
	assert.NotNil(t, findings)
	assert.Greater(t, len(findings), 0)

	// Check for consistency finding
	foundConsistency := false
	for _, finding := range findings {
		if finding.Type == "consistency" {
			foundConsistency = true
			assert.Equal(t, "medium", finding.Severity)
			assert.Contains(t, finding.Compliance, "CIS-2.2")
		}
	}
	assert.True(t, foundConsistency, "Should find consistency issue")
}
