package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// SecurityAnalyzer handles security analysis and compliance checking
type SecurityAnalyzer struct {
	storage core.Storage
}

// NewSecurityAnalyzer creates a new security analyzer
func NewSecurityAnalyzer(storage core.Storage) *SecurityAnalyzer {
	return &SecurityAnalyzer{
		storage: storage,
	}
}

// SecurityFinding represents a security issue or compliance violation
type SecurityFinding struct {
	ID             string                 `json:"id"`
	ResourceID     string                 `json:"resource_id"`
	ResourceARN    string                 `json:"resource_arn"`
	Provider       string                 `json:"provider"`
	Service        string                 `json:"service"`
	Type           string                 `json:"type"`
	Severity       string                 `json:"severity"` // "critical", "high", "medium", "low", "info"
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Recommendation string                 `json:"recommendation"`
	Compliance     []string               `json:"compliance"` // CIS, SOC2, PCI-DSS, etc.
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      string                 `json:"created_at"`
}

// SecurityReport represents the complete security analysis report
type SecurityReport struct {
	Findings        []SecurityFinding `json:"findings"`
	Summary         SecuritySummary   `json:"summary"`
	ComplianceScore float64           `json:"compliance_score"`
	RiskScore       float64           `json:"risk_score"`
}

// SecuritySummary provides statistics about security findings
type SecuritySummary struct {
	TotalFindings    int `json:"total_findings"`
	CriticalFindings int `json:"critical_findings"`
	HighFindings     int `json:"high_findings"`
	MediumFindings   int `json:"medium_findings"`
	LowFindings      int `json:"low_findings"`
	InfoFindings     int `json:"info_findings"`
	CompliancePass   int `json:"compliance_pass"`
	ComplianceFail   int `json:"compliance_fail"`
}

// AnalyzeSecurity performs comprehensive security analysis
func (sa *SecurityAnalyzer) AnalyzeSecurity(ctx context.Context) (*SecurityReport, error) {
	logrus.Info("Starting security analysis")

	// Get all resources
	resources, err := sa.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	var findings []SecurityFinding

	// Group resources by provider for provider-specific analysis
	providerResources := make(map[string][]core.Resource)
	for _, resource := range resources {
		providerResources[resource.Provider] = append(providerResources[resource.Provider], resource)
	}

	// Analyze each provider
	for provider, providerResources := range providerResources {
		logrus.Infof("Analyzing security for %s resources", provider)
		providerFindings, err := sa.analyzeProviderSecurity(ctx, provider, providerResources)
		if err != nil {
			logrus.Warnf("Failed to analyze security for %s: %v", provider, err)
			continue
		}
		findings = append(findings, providerFindings...)
	}

	// Perform cross-provider security analysis
	crossProviderFindings, err := sa.analyzeCrossProviderSecurity(ctx, resources)
	if err != nil {
		logrus.Warnf("Failed to analyze cross-provider security: %v", err)
	}
	findings = append(findings, crossProviderFindings...)

	// Calculate summary and scores
	summary := sa.calculateSecuritySummary(findings)
	complianceScore := sa.calculateComplianceScore(findings)
	riskScore := sa.calculateRiskScore(findings)

	report := &SecurityReport{
		Findings:        findings,
		Summary:         summary,
		ComplianceScore: complianceScore,
		RiskScore:       riskScore,
	}

	logrus.Infof("Security analysis completed: %d findings", len(findings))

	return report, nil
}

// analyzeProviderSecurity analyzes security for a specific provider
func (sa *SecurityAnalyzer) analyzeProviderSecurity(ctx context.Context, provider string, resources []core.Resource) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	switch provider {
	case "aws":
		findings = sa.analyzeAWSSecurity(resources)
	case "azure":
		findings = sa.analyzeAzureSecurity(resources)
	case "gcp":
		findings = sa.analyzeGCPSecurity(resources)
	default:
		logrus.Warnf("Unknown provider for security analysis: %s", provider)
	}

	return findings, nil
}

// analyzeAWSSecurity performs AWS-specific security analysis
func (sa *SecurityAnalyzer) analyzeAWSSecurity(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		switch resource.Service {
		case "ec2":
			findings = append(findings, sa.analyzeEC2Security(resource)...)
		case "s3":
			findings = append(findings, sa.analyzeS3Security(resource)...)
		case "rds":
			findings = append(findings, sa.analyzeRDSSecurity(resource)...)
		case "iam":
			findings = append(findings, sa.analyzeIAMSecurity(resource)...)
		case "lambda":
			findings = append(findings, sa.analyzeLambdaSecurity(resource)...)
		}
	}

	return findings
}

// analyzeEC2Security analyzes EC2 instance security
func (sa *SecurityAnalyzer) analyzeEC2Security(instance core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for public IP addresses
	if instance.PublicAccess {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("ec2-public-access-%s", instance.ID),
			ResourceID:     instance.ID,
			ResourceARN:    instance.ARN,
			Provider:       instance.Provider,
			Service:        instance.Service,
			Type:           "public_access",
			Severity:       "high",
			Title:          "EC2 Instance has public IP address",
			Description:    "EC2 instance is accessible from the internet",
			Recommendation: "Remove public IP or use NAT gateway for outbound access",
			Compliance:     []string{"CIS-2.1", "SOC2-CC6.1"},
			Metadata: map[string]interface{}{
				"resource_type": instance.Type,
				"region":        instance.Region,
			},
		})
	}

	// Check for encryption
	if !instance.Encrypted {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("ec2-unencrypted-%s", instance.ID),
			ResourceID:     instance.ID,
			ResourceARN:    instance.ARN,
			Provider:       instance.Provider,
			Service:        instance.Service,
			Type:           "encryption",
			Severity:       "medium",
			Title:          "EC2 Instance storage is not encrypted",
			Description:    "EC2 instance EBS volumes are not encrypted",
			Recommendation: "Enable EBS encryption for all volumes",
			Compliance:     []string{"CIS-2.2", "PCI-DSS-3.4"},
			Metadata: map[string]interface{}{
				"resource_type": instance.Type,
				"region":        instance.Region,
			},
		})
	}

	// Check for compliance flags
	for _, flag := range instance.Compliance {
		switch flag {
		case "public-access":
			findings = append(findings, SecurityFinding{
				ID:             fmt.Sprintf("ec2-compliance-public-%s", instance.ID),
				ResourceID:     instance.ID,
				ResourceARN:    instance.ARN,
				Provider:       instance.Provider,
				Service:        instance.Service,
				Type:           "compliance",
				Severity:       "high",
				Title:          "EC2 Instance violates public access policy",
				Description:    "Instance has public access which violates security policy",
				Recommendation: "Review and restrict public access",
				Compliance:     []string{"CIS-2.1"},
				Metadata: map[string]interface{}{
					"compliance_flag": flag,
				},
			})
		case "unencrypted":
			findings = append(findings, SecurityFinding{
				ID:             fmt.Sprintf("ec2-compliance-encryption-%s", instance.ID),
				ResourceID:     instance.ID,
				ResourceARN:    instance.ARN,
				Provider:       instance.Provider,
				Service:        instance.Service,
				Type:           "compliance",
				Severity:       "medium",
				Title:          "EC2 Instance violates encryption policy",
				Description:    "Instance storage is not encrypted",
				Recommendation: "Enable encryption for all storage",
				Compliance:     []string{"CIS-2.2"},
				Metadata: map[string]interface{}{
					"compliance_flag": flag,
				},
			})
		}
	}

	return findings
}

// analyzeS3Security analyzes S3 bucket security
func (sa *SecurityAnalyzer) analyzeS3Security(bucket core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for public access
	if bucket.PublicAccess {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("s3-public-access-%s", bucket.ID),
			ResourceID:     bucket.ID,
			ResourceARN:    bucket.ARN,
			Provider:       bucket.Provider,
			Service:        bucket.Service,
			Type:           "public_access",
			Severity:       "critical",
			Title:          "S3 Bucket allows public access",
			Description:    "S3 bucket is publicly accessible",
			Recommendation: "Remove public access policies and block public access",
			Compliance:     []string{"CIS-2.1", "SOC2-CC6.1", "PCI-DSS-1.2"},
			Metadata: map[string]interface{}{
				"resource_type": bucket.Type,
				"region":        bucket.Region,
			},
		})
	}

	// Check for encryption
	if !bucket.Encrypted {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("s3-unencrypted-%s", bucket.ID),
			ResourceID:     bucket.ID,
			ResourceARN:    bucket.ARN,
			Provider:       bucket.Provider,
			Service:        bucket.Service,
			Type:           "encryption",
			Severity:       "high",
			Title:          "S3 Bucket is not encrypted",
			Description:    "S3 bucket does not have encryption enabled",
			Recommendation: "Enable server-side encryption for the bucket",
			Compliance:     []string{"CIS-2.2", "SOC2-CC6.1", "PCI-DSS-3.4"},
			Metadata: map[string]interface{}{
				"resource_type": bucket.Type,
				"region":        bucket.Region,
			},
		})
	}

	return findings
}

// analyzeRDSSecurity analyzes RDS instance security
func (sa *SecurityAnalyzer) analyzeRDSSecurity(rds core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for encryption
	if !rds.Encrypted {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("rds-unencrypted-%s", rds.ID),
			ResourceID:     rds.ID,
			ResourceARN:    rds.ARN,
			Provider:       rds.Provider,
			Service:        rds.Service,
			Type:           "encryption",
			Severity:       "high",
			Title:          "RDS Instance is not encrypted",
			Description:    "RDS instance storage is not encrypted",
			Recommendation: "Enable encryption for RDS instance",
			Compliance:     []string{"CIS-2.2", "SOC2-CC6.1", "PCI-DSS-3.4"},
			Metadata: map[string]interface{}{
				"resource_type": rds.Type,
				"region":        rds.Region,
			},
		})
	}

	return findings
}

// analyzeIAMSecurity analyzes IAM security
func (sa *SecurityAnalyzer) analyzeIAMSecurity(iam core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for overly permissive policies
	config := string(iam.Configuration)
	if strings.Contains(config, "Effect\":\"Allow\"") && strings.Contains(config, "Resource\":\"*\"") {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("iam-overly-permissive-%s", iam.ID),
			ResourceID:     iam.ID,
			ResourceARN:    iam.ARN,
			Provider:       iam.Provider,
			Service:        iam.Service,
			Type:           "permissions",
			Severity:       "high",
			Title:          "IAM policy is overly permissive",
			Description:    "IAM policy allows access to all resources",
			Recommendation: "Apply principle of least privilege",
			Compliance:     []string{"CIS-1.16", "SOC2-CC6.1"},
			Metadata: map[string]interface{}{
				"resource_type": iam.Type,
				"region":        iam.Region,
			},
		})
	}

	return findings
}

// analyzeLambdaSecurity analyzes Lambda function security
func (sa *SecurityAnalyzer) analyzeLambdaSecurity(lambda core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for VPC configuration
	config := string(lambda.Configuration)
	if !strings.Contains(config, "VpcConfig") {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("lambda-no-vpc-%s", lambda.ID),
			ResourceID:     lambda.ID,
			ResourceARN:    lambda.ARN,
			Provider:       lambda.Provider,
			Service:        lambda.Service,
			Type:           "network",
			Severity:       "medium",
			Title:          "Lambda function not in VPC",
			Description:    "Lambda function is not configured to run in a VPC",
			Recommendation: "Consider running Lambda in VPC for better network isolation",
			Compliance:     []string{"CIS-2.3"},
			Metadata: map[string]interface{}{
				"resource_type": lambda.Type,
				"region":        lambda.Region,
			},
		})
	}

	return findings
}

// analyzeAzureSecurity performs Azure-specific security analysis
func (sa *SecurityAnalyzer) analyzeAzureSecurity(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		switch resource.Service {
		case "compute":
			findings = append(findings, sa.analyzeAzureComputeSecurity(resource)...)
		case "storage":
			findings = append(findings, sa.analyzeAzureStorageSecurity(resource)...)
		}
	}

	return findings
}

// analyzeAzureComputeSecurity analyzes Azure compute security
func (sa *SecurityAnalyzer) analyzeAzureComputeSecurity(vm core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for public access
	if vm.PublicAccess {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("azure-vm-public-access-%s", vm.ID),
			ResourceID:     vm.ID,
			ResourceARN:    vm.ARN,
			Provider:       vm.Provider,
			Service:        vm.Service,
			Type:           "public_access",
			Severity:       "high",
			Title:          "Azure VM has public IP address",
			Description:    "Azure VM is accessible from the internet",
			Recommendation: "Remove public IP or use NAT gateway",
			Compliance:     []string{"CIS-2.1", "SOC2-CC6.1"},
			Metadata: map[string]interface{}{
				"resource_type": vm.Type,
				"region":        vm.Region,
			},
		})
	}

	return findings
}

// analyzeAzureStorageSecurity analyzes Azure storage security
func (sa *SecurityAnalyzer) analyzeAzureStorageSecurity(storage core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for public access
	if storage.PublicAccess {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("azure-storage-public-access-%s", storage.ID),
			ResourceID:     storage.ID,
			ResourceARN:    storage.ARN,
			Provider:       storage.Provider,
			Service:        storage.Service,
			Type:           "public_access",
			Severity:       "critical",
			Title:          "Azure Storage allows public access",
			Description:    "Azure Storage account is publicly accessible",
			Recommendation: "Restrict public access to storage account",
			Compliance:     []string{"CIS-2.1", "SOC2-CC6.1", "PCI-DSS-1.2"},
			Metadata: map[string]interface{}{
				"resource_type": storage.Type,
				"region":        storage.Region,
			},
		})
	}

	return findings
}

// analyzeGCPSecurity performs GCP-specific security analysis
func (sa *SecurityAnalyzer) analyzeGCPSecurity(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		switch resource.Service {
		case "compute":
			findings = append(findings, sa.analyzeGCPComputeSecurity(resource)...)
		case "storage":
			findings = append(findings, sa.analyzeGCPStorageSecurity(resource)...)
		}
	}

	return findings
}

// analyzeGCPComputeSecurity analyzes GCP compute security
func (sa *SecurityAnalyzer) analyzeGCPComputeSecurity(instance core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for public access
	if instance.PublicAccess {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("gcp-vm-public-access-%s", instance.ID),
			ResourceID:     instance.ID,
			ResourceARN:    instance.ARN,
			Provider:       instance.Provider,
			Service:        instance.Service,
			Type:           "public_access",
			Severity:       "high",
			Title:          "GCP VM has external IP address",
			Description:    "GCP VM is accessible from the internet",
			Recommendation: "Remove external IP or use Cloud NAT",
			Compliance:     []string{"CIS-2.1", "SOC2-CC6.1"},
			Metadata: map[string]interface{}{
				"resource_type": instance.Type,
				"region":        instance.Region,
			},
		})
	}

	return findings
}

// analyzeGCPStorageSecurity analyzes GCP storage security
func (sa *SecurityAnalyzer) analyzeGCPStorageSecurity(storage core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for public access
	if storage.PublicAccess {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("gcp-storage-public-access-%s", storage.ID),
			ResourceID:     storage.ID,
			ResourceARN:    storage.ARN,
			Provider:       storage.Provider,
			Service:        storage.Service,
			Type:           "public_access",
			Severity:       "critical",
			Title:          "GCP Storage allows public access",
			Description:    "GCP Storage bucket is publicly accessible",
			Recommendation: "Restrict public access to storage bucket",
			Compliance:     []string{"CIS-2.1", "SOC2-CC6.1", "PCI-DSS-1.2"},
			Metadata: map[string]interface{}{
				"resource_type": storage.Type,
				"region":        storage.Region,
			},
		})
	}

	return findings
}

// analyzeCrossProviderSecurity analyzes security across different cloud providers
func (sa *SecurityAnalyzer) analyzeCrossProviderSecurity(ctx context.Context, resources []core.Resource) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	// Look for cross-cloud security issues
	// Example: Resources with similar names but different security configurations
	resourceGroups := sa.groupResourcesByPatterns(resources)

	for _, group := range resourceGroups {
		if len(group) > 1 {
			// Check for inconsistent security configurations
			securityConfigs := sa.analyzeSecurityConsistency(group)
			findings = append(findings, securityConfigs...)
		}
	}

	return findings, nil
}

// groupResourcesByPatterns groups resources by common patterns
func (sa *SecurityAnalyzer) groupResourcesByPatterns(resources []core.Resource) [][]core.Resource {
	groups := make(map[string][]core.Resource)

	for _, resource := range resources {
		// Group by common tags or naming patterns
		key := sa.generateGroupKey(resource)
		groups[key] = append(groups[key], resource)
	}

	var result [][]core.Resource
	for _, group := range groups {
		if len(group) > 1 {
			result = append(result, group)
		}
	}

	return result
}

// generateGroupKey generates a key for grouping resources
func (sa *SecurityAnalyzer) generateGroupKey(resource core.Resource) string {
	key := resource.Provider + ":" + resource.Service

	if resource.Tags != nil {
		if env, ok := resource.Tags["Environment"]; ok {
			key += ":" + env
		}
		if project, ok := resource.Tags["Project"]; ok {
			key += ":" + project
		}
	}

	return key
}

// analyzeSecurityConsistency analyzes security consistency within a group
func (sa *SecurityAnalyzer) analyzeSecurityConsistency(group []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Check for inconsistent encryption settings
	encryptedCount := 0
	for _, resource := range group {
		if resource.Encrypted {
			encryptedCount++
		}
	}

	if encryptedCount > 0 && encryptedCount < len(group) {
		findings = append(findings, SecurityFinding{
			ID:             fmt.Sprintf("inconsistent-encryption-%s", group[0].ID),
			ResourceID:     group[0].ID,
			ResourceARN:    group[0].ARN,
			Provider:       group[0].Provider,
			Service:        group[0].Service,
			Type:           "consistency",
			Severity:       "medium",
			Title:          "Inconsistent encryption across related resources",
			Description:    "Some resources in the group are encrypted while others are not",
			Recommendation: "Apply consistent encryption policy across all related resources",
			Compliance:     []string{"CIS-2.2"},
			Metadata: map[string]interface{}{
				"group_size":        len(group),
				"encrypted_count":   encryptedCount,
				"unencrypted_count": len(group) - encryptedCount,
			},
		})
	}

	return findings
}

// calculateSecuritySummary calculates the security summary
func (sa *SecurityAnalyzer) calculateSecuritySummary(findings []SecurityFinding) SecuritySummary {
	summary := SecuritySummary{
		TotalFindings: len(findings),
	}

	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			summary.CriticalFindings++
		case "high":
			summary.HighFindings++
		case "medium":
			summary.MediumFindings++
		case "low":
			summary.LowFindings++
		case "info":
			summary.InfoFindings++
		}
	}

	// Calculate compliance pass/fail
	summary.ComplianceFail = len(findings)
	summary.CompliancePass = 0 // This would be calculated based on total resources

	return summary
}

// calculateComplianceScore calculates the compliance score (0-100)
func (sa *SecurityAnalyzer) calculateComplianceScore(findings []SecurityFinding) float64 {
	// Simplified calculation - in practice, this would be more sophisticated
	totalFindings := len(findings)
	if totalFindings == 0 {
		return 100.0
	}

	// Weight findings by severity
	weightedScore := 0.0
	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			weightedScore += 10.0
		case "high":
			weightedScore += 5.0
		case "medium":
			weightedScore += 2.0
		case "low":
			weightedScore += 1.0
		case "info":
			weightedScore += 0.5
		}
	}

	// Calculate score (higher is better)
	score := 100.0 - (weightedScore / float64(totalFindings))
	if score < 0 {
		score = 0
	}

	return score
}

// calculateRiskScore calculates the risk score (0-100)
func (sa *SecurityAnalyzer) calculateRiskScore(findings []SecurityFinding) float64 {
	// Simplified calculation - in practice, this would be more sophisticated
	totalFindings := len(findings)
	if totalFindings == 0 {
		return 0.0
	}

	// Weight findings by severity
	weightedScore := 0.0
	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			weightedScore += 10.0
		case "high":
			weightedScore += 5.0
		case "medium":
			weightedScore += 2.0
		case "low":
			weightedScore += 1.0
		case "info":
			weightedScore += 0.5
		}
	}

	// Calculate risk score (higher is worse)
	riskScore := weightedScore / float64(totalFindings)
	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore
}
