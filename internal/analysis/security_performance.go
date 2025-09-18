package analysis

import (
	"context"
	"sync"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// PerformanceOptimizedSecurityAnalyzer is an optimized version of SecurityAnalyzer
type PerformanceOptimizedSecurityAnalyzer struct {
	*SecurityAnalyzer
	config     *PerformanceConfig
	cache      map[string]interface{}
	cacheMutex sync.RWMutex
}

// NewPerformanceOptimizedSecurityAnalyzer creates a new performance-optimized security analyzer
func NewPerformanceOptimizedSecurityAnalyzer(storage core.Storage, config *PerformanceConfig) *PerformanceOptimizedSecurityAnalyzer {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	return &PerformanceOptimizedSecurityAnalyzer{
		SecurityAnalyzer: NewSecurityAnalyzer(storage),
		config:           config,
		cache:            make(map[string]interface{}),
	}
}

// AnalyzeSecurityOptimized performs optimized security analysis with parallel processing
func (posa *PerformanceOptimizedSecurityAnalyzer) AnalyzeSecurityOptimized(ctx context.Context) (*SecurityReport, error) {
	start := time.Now()
	logrus.Info("Starting optimized security analysis")

	// Get all resources with caching
	resources, err := posa.getResourcesCached(ctx)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return &SecurityReport{
			Findings:        []SecurityFinding{},
			Summary:         SecuritySummary{},
			ComplianceScore: 100.0,
			RiskScore:       0.0,
		}, nil
	}

	// Group resources by provider for parallel processing
	providerResources := posa.groupResourcesByProvider(resources)

	// Process providers in parallel
	var findings []SecurityFinding
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create worker pool
	workerCount := posa.config.MaxWorkers
	if len(providerResources) < workerCount {
		workerCount = len(providerResources)
	}

	providerChan := make(chan providerSecurityWork, len(providerResources))

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go posa.securityWorker(ctx, providerChan, &findings, &mu, &wg)
	}

	// Send work to workers
	for provider, resources := range providerResources {
		providerChan <- providerSecurityWork{
			Provider:  provider,
			Resources: resources,
		}
	}
	close(providerChan)

	// Wait for all workers to complete
	wg.Wait()

	// Calculate summary and scores
	summary := posa.calculateSecuritySummaryOptimized(findings)
	complianceScore := posa.calculateComplianceScoreOptimized(findings)
	riskScore := posa.calculateRiskScoreOptimized(findings)

	report := &SecurityReport{
		Findings:        findings,
		Summary:         summary,
		ComplianceScore: complianceScore,
		RiskScore:       riskScore,
	}

	duration := time.Since(start)
	logrus.Infof("Optimized security analysis completed: %d resources, %d findings in %v",
		len(resources), len(findings), duration)

	return report, nil
}

type providerSecurityWork struct {
	Provider  string
	Resources []core.Resource
}

// securityWorker processes provider security work items
func (posa *PerformanceOptimizedSecurityAnalyzer) securityWorker(ctx context.Context, workChan <-chan providerSecurityWork, findings *[]SecurityFinding, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	for work := range workChan {
		select {
		case <-ctx.Done():
			return
		default:
			providerFindings, err := posa.analyzeProviderSecurity(ctx, work.Provider, work.Resources)
			if err != nil {
				logrus.Warnf("Failed to analyze security for %s: %v", work.Provider, err)
				continue
			}

			mu.Lock()
			*findings = append(*findings, providerFindings...)
			mu.Unlock()
		}
	}
}

// analyzeProviderSecurity analyzes security for a specific provider
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeProviderSecurity(ctx context.Context, provider string, resources []core.Resource) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	switch provider {
	case "aws":
		findings = posa.analyzeAWSSecurityOptimized(resources)
	case "azure":
		findings = posa.analyzeAzureSecurityOptimized(resources)
	case "gcp":
		findings = posa.analyzeGCPSecurityOptimized(resources)
	default:
		logrus.Warnf("Unknown provider for security analysis: %s", provider)
	}

	return findings, nil
}

// analyzeAWSSecurityOptimized performs optimized AWS security analysis
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAWSSecurityOptimized(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Process resources in batches for better memory usage
	batchSize := posa.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		batchFindings := posa.analyzeAWSBatch(batch)
		findings = append(findings, batchFindings...)
	}

	return findings
}

// analyzeAWSBatch analyzes a batch of AWS resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAWSBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Group resources by service for efficient processing
	serviceGroups := make(map[string][]core.Resource)
	for _, resource := range resources {
		serviceGroups[resource.Service] = append(serviceGroups[resource.Service], resource)
	}

	// Process each service group
	for service, serviceResources := range serviceGroups {
		switch service {
		case "ec2":
			findings = append(findings, posa.analyzeEC2SecurityBatch(serviceResources)...)
		case "s3":
			findings = append(findings, posa.analyzeS3SecurityBatch(serviceResources)...)
		case "rds":
			findings = append(findings, posa.analyzeRDSSecurityBatch(serviceResources)...)
		case "iam":
			findings = append(findings, posa.analyzeIAMSecurityBatch(serviceResources)...)
		case "lambda":
			findings = append(findings, posa.analyzeLambdaSecurityBatch(serviceResources)...)
		}
	}

	return findings
}

// analyzeEC2SecurityBatch analyzes EC2 security for a batch of resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeEC2SecurityBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		// Check for public IP addresses
		if posa.hasPublicIP(resource) {
			findings = append(findings, SecurityFinding{
				ID:             "ec2_public_ip_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "high",
				Title:          "EC2 instance with public IP",
				Description:    "EC2 instance has a public IP address which may expose it to the internet",
				Recommendation: "Consider using a private subnet or NAT gateway",
				Compliance:     []string{"CIS"},
				Metadata: map[string]interface{}{
					"service": "ec2",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}

		// Check for unencrypted volumes
		if !posa.isVolumeEncrypted(resource) {
			findings = append(findings, SecurityFinding{
				ID:             "ec2_unencrypted_volume_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "medium",
				Title:          "Unencrypted EBS volume",
				Description:    "EBS volume is not encrypted",
				Recommendation: "Enable EBS encryption for data at rest",
				Compliance:     []string{"CIS"},
				Metadata: map[string]interface{}{
					"service": "ec2",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}
	}

	return findings
}

// analyzeS3SecurityBatch analyzes S3 security for a batch of resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeS3SecurityBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		// Check for public access
		if resource.PublicAccess {
			findings = append(findings, SecurityFinding{
				ID:             "s3_public_access_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "critical",
				Title:          "S3 bucket with public access",
				Description:    "S3 bucket allows public access which may expose sensitive data",
				Recommendation: "Remove public access and use IAM policies for access control",
				Compliance:     []string{"CIS", "SOC2"},
				Metadata: map[string]interface{}{
					"service": "s3",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}

		// Check for encryption
		if !resource.Encrypted {
			findings = append(findings, SecurityFinding{
				ID:             "s3_unencrypted_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "medium",
				Title:          "S3 bucket without encryption",
				Description:    "S3 bucket is not encrypted",
				Recommendation: "Enable S3 bucket encryption",
				Compliance:     []string{"CIS"},
				Metadata: map[string]interface{}{
					"service": "s3",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}
	}

	return findings
}

// analyzeRDSSecurityBatch analyzes RDS security for a batch of resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeRDSSecurityBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		// Check for encryption
		if !resource.Encrypted {
			findings = append(findings, SecurityFinding{
				ID:             "rds_unencrypted_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "high",
				Title:          "RDS instance without encryption",
				Description:    "RDS instance is not encrypted",
				Recommendation: "Enable RDS encryption for data at rest",
				Compliance:     []string{"CIS"},
				Metadata: map[string]interface{}{
					"service": "rds",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}

		// Check for public access
		if posa.isRDSPubliclyAccessible(resource) {
			findings = append(findings, SecurityFinding{
				ID:             "rds_public_access_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "critical",
				Title:          "RDS instance with public access",
				Description:    "RDS instance is publicly accessible",
				Recommendation: "Remove public access and use VPC security groups",
				Compliance:     []string{"CIS", "SOC2"},
				Metadata: map[string]interface{}{
					"service": "rds",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}
	}

	return findings
}

// analyzeIAMSecurityBatch analyzes IAM security for a batch of resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeIAMSecurityBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		// Check for overly permissive policies
		if posa.hasOverlyPermissivePolicy(resource) {
			findings = append(findings, SecurityFinding{
				ID:             "iam_overly_permissive_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "high",
				Title:          "Overly permissive IAM policy",
				Description:    "IAM policy grants excessive permissions",
				Recommendation: "Apply principle of least privilege",
				Compliance:     []string{"CIS"},
				Metadata: map[string]interface{}{
					"service": "iam",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}
	}

	return findings
}

// analyzeLambdaSecurityBatch analyzes Lambda security for a batch of resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeLambdaSecurityBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		// Check for environment variables with sensitive data
		if posa.hasSensitiveEnvironmentVariables(resource) {
			findings = append(findings, SecurityFinding{
				ID:             "lambda_sensitive_env_" + resource.ID,
				ResourceID:     resource.ID,
				ResourceARN:    resource.ARN,
				Provider:       resource.Provider,
				Service:        resource.Service,
				Type:           resource.Type,
				Severity:       "medium",
				Title:          "Lambda with sensitive environment variables",
				Description:    "Lambda function has environment variables that may contain sensitive data",
				Recommendation: "Use AWS Secrets Manager or Parameter Store for sensitive data",
				Compliance:     []string{"CIS"},
				Metadata: map[string]interface{}{
					"service": "lambda",
					"region":  resource.Region,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
			})
		}
	}

	return findings
}

// analyzeAzureSecurityOptimized performs optimized Azure security analysis
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAzureSecurityOptimized(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Process resources in batches
	batchSize := posa.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		batchFindings := posa.analyzeAzureBatch(batch)
		findings = append(findings, batchFindings...)
	}

	return findings
}

// analyzeAzureBatch analyzes a batch of Azure resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAzureBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Group resources by service
	serviceGroups := make(map[string][]core.Resource)
	for _, resource := range resources {
		serviceGroups[resource.Service] = append(serviceGroups[resource.Service], resource)
	}

	// Process each service group
	for service, serviceResources := range serviceGroups {
		switch service {
		case "compute":
			findings = append(findings, posa.analyzeAzureComputeSecurityBatch(serviceResources)...)
		case "storage":
			findings = append(findings, posa.analyzeAzureStorageSecurityBatch(serviceResources)...)
		case "database":
			findings = append(findings, posa.analyzeAzureDatabaseSecurityBatch(serviceResources)...)
		}
	}

	return findings
}

// analyzeGCPSecurityOptimized performs optimized GCP security analysis
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeGCPSecurityOptimized(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Process resources in batches
	batchSize := posa.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		batchFindings := posa.analyzeGCPBatch(batch)
		findings = append(findings, batchFindings...)
	}

	return findings
}

// analyzeGCPBatch analyzes a batch of GCP resources
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeGCPBatch(resources []core.Resource) []SecurityFinding {
	var findings []SecurityFinding

	// Group resources by service
	serviceGroups := make(map[string][]core.Resource)
	for _, resource := range resources {
		serviceGroups[resource.Service] = append(serviceGroups[resource.Service], resource)
	}

	// Process each service group
	for service, serviceResources := range serviceGroups {
		switch service {
		case "compute":
			findings = append(findings, posa.analyzeGCPComputeSecurityBatch(serviceResources)...)
		case "storage":
			findings = append(findings, posa.analyzeGCPStorageSecurityBatch(serviceResources)...)
		case "database":
			findings = append(findings, posa.analyzeGCPDatabaseSecurityBatch(serviceResources)...)
		}
	}

	return findings
}

// Helper methods for security analysis
func (posa *PerformanceOptimizedSecurityAnalyzer) hasPublicIP(resource core.Resource) bool {
	// Simplified implementation - in practice, you'd parse the configuration
	return resource.PublicAccess
}

func (posa *PerformanceOptimizedSecurityAnalyzer) isVolumeEncrypted(resource core.Resource) bool {
	// Simplified implementation - in practice, you'd parse the configuration
	return resource.Encrypted
}

func (posa *PerformanceOptimizedSecurityAnalyzer) isRDSPubliclyAccessible(resource core.Resource) bool {
	// Simplified implementation - in practice, you'd parse the configuration
	return resource.PublicAccess
}

func (posa *PerformanceOptimizedSecurityAnalyzer) hasOverlyPermissivePolicy(resource core.Resource) bool {
	// Simplified implementation - in practice, you'd parse the IAM policy
	config := string(resource.Configuration)
	return len(config) > 1000 // Heuristic for overly permissive policies
}

func (posa *PerformanceOptimizedSecurityAnalyzer) hasSensitiveEnvironmentVariables(resource core.Resource) bool {
	// Simplified implementation - in practice, you'd parse the Lambda configuration
	config := string(resource.Configuration)
	return len(config) > 500 // Heuristic for sensitive environment variables
}

// Placeholder methods for Azure and GCP batch analysis
func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAzureComputeSecurityBatch(resources []core.Resource) []SecurityFinding {
	return []SecurityFinding{}
}

func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAzureStorageSecurityBatch(resources []core.Resource) []SecurityFinding {
	return []SecurityFinding{}
}

func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeAzureDatabaseSecurityBatch(resources []core.Resource) []SecurityFinding {
	return []SecurityFinding{}
}

func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeGCPComputeSecurityBatch(resources []core.Resource) []SecurityFinding {
	return []SecurityFinding{}
}

func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeGCPStorageSecurityBatch(resources []core.Resource) []SecurityFinding {
	return []SecurityFinding{}
}

func (posa *PerformanceOptimizedSecurityAnalyzer) analyzeGCPDatabaseSecurityBatch(resources []core.Resource) []SecurityFinding {
	return []SecurityFinding{}
}

// getResourcesCached retrieves resources with caching
func (posa *PerformanceOptimizedSecurityAnalyzer) getResourcesCached(ctx context.Context) ([]core.Resource, error) {
	cacheKey := "resources_all"

	// Check cache first
	posa.cacheMutex.RLock()
	if cached, exists := posa.cache[cacheKey]; exists {
		if resources, ok := cached.([]core.Resource); ok {
			posa.cacheMutex.RUnlock()
			return resources, nil
		}
	}
	posa.cacheMutex.RUnlock()

	// Get from storage
	resources, err := posa.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, err
	}

	// Cache the result
	posa.cacheMutex.Lock()
	posa.cache[cacheKey] = resources
	posa.cacheMutex.Unlock()

	return resources, nil
}

// groupResourcesByProvider groups resources by provider efficiently
func (posa *PerformanceOptimizedSecurityAnalyzer) groupResourcesByProvider(resources []core.Resource) map[string][]core.Resource {
	providerResources := make(map[string][]core.Resource)

	for _, resource := range resources {
		providerResources[resource.Provider] = append(providerResources[resource.Provider], resource)
	}

	return providerResources
}

// calculateSecuritySummaryOptimized calculates security summary efficiently
func (posa *PerformanceOptimizedSecurityAnalyzer) calculateSecuritySummaryOptimized(findings []SecurityFinding) SecuritySummary {
	summary := SecuritySummary{
		TotalFindings:    len(findings),
		CriticalFindings: 0,
		HighFindings:     0,
		MediumFindings:   0,
		LowFindings:      0,
		InfoFindings:     0,
		CompliancePass:   0,
		ComplianceFail:   0,
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

	return summary
}

// calculateComplianceScoreOptimized calculates compliance score efficiently
func (posa *PerformanceOptimizedSecurityAnalyzer) calculateComplianceScoreOptimized(findings []SecurityFinding) float64 {
	if len(findings) == 0 {
		return 100.0
	}

	// Calculate score based on severity
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}

	// Weighted scoring
	score := 100.0
	score -= float64(criticalCount) * 20.0
	score -= float64(highCount) * 10.0
	score -= float64(mediumCount) * 5.0
	score -= float64(lowCount) * 1.0

	if score < 0 {
		score = 0
	}

	return score
}

// calculateRiskScoreOptimized calculates risk score efficiently
func (posa *PerformanceOptimizedSecurityAnalyzer) calculateRiskScoreOptimized(findings []SecurityFinding) float64 {
	if len(findings) == 0 {
		return 0.0
	}

	// Calculate risk score based on severity
	riskScore := 0.0
	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			riskScore += 10.0
		case "high":
			riskScore += 7.0
		case "medium":
			riskScore += 4.0
		case "low":
			riskScore += 1.0
		}
	}

	// Normalize to 0-100 scale
	maxPossibleScore := float64(len(findings)) * 10.0
	if maxPossibleScore > 0 {
		riskScore = (riskScore / maxPossibleScore) * 100.0
	}

	return riskScore
}

// ClearCache clears the analysis cache
func (posa *PerformanceOptimizedSecurityAnalyzer) ClearCache() {
	posa.cacheMutex.Lock()
	defer posa.cacheMutex.Unlock()

	posa.cache = make(map[string]interface{})
}

// GetCacheStats returns cache statistics
func (posa *PerformanceOptimizedSecurityAnalyzer) GetCacheStats() map[string]interface{} {
	posa.cacheMutex.RLock()
	defer posa.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cache_size":       len(posa.cache),
		"max_workers":      posa.config.MaxWorkers,
		"batch_size":       posa.config.BatchSize,
		"parallel_enabled": posa.config.EnableParallel,
	}
}
