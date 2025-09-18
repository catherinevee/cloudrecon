package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// CostAnalyzer handles cost analysis and optimization recommendations
type CostAnalyzer struct {
	storage core.Storage
}

// NewCostAnalyzer creates a new cost analyzer
func NewCostAnalyzer(storage core.Storage) *CostAnalyzer {
	return &CostAnalyzer{
		storage: storage,
	}
}

// CostEstimate represents a cost estimate for a resource
type CostEstimate struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceARN  string                 `json:"resource_arn"`
	Provider     string                 `json:"provider"`
	Service      string                 `json:"service"`
	Type         string                 `json:"type"`
	Region       string                 `json:"region"`
	MonthlyCost  float64                `json:"monthly_cost"`
	DailyCost    float64                `json:"daily_cost"`
	HourlyCost   float64                `json:"hourly_cost"`
	Currency     string                 `json:"currency"`
	PricingModel string                 `json:"pricing_model"` // "on-demand", "reserved", "spot"
	Confidence   float64                `json:"confidence"`    // 0.0 to 1.0
	Metadata     map[string]interface{} `json:"metadata"`
}

// CostOptimization represents a cost optimization recommendation
type CostOptimization struct {
	ID               string                 `json:"id"`
	ResourceID       string                 `json:"resource_id"`
	ResourceARN      string                 `json:"resource_arn"`
	Provider         string                 `json:"provider"`
	Service          string                 `json:"service"`
	Type             string                 `json:"type"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	CurrentCost      float64                `json:"current_cost"`
	PotentialSavings float64                `json:"potential_savings"`
	SavingsPercent   float64                `json:"savings_percent"`
	Priority         string                 `json:"priority"` // "high", "medium", "low"
	Category         string                 `json:"category"` // "rightsizing", "reserved", "spot", "unused"
	Recommendation   string                 `json:"recommendation"`
	Implementation   string                 `json:"implementation"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// CostReport represents the complete cost analysis report
type CostReport struct {
	TotalMonthlyCost float64            `json:"total_monthly_cost"`
	TotalDailyCost   float64            `json:"total_daily_cost"`
	TotalHourlyCost  float64            `json:"total_hourly_cost"`
	Currency         string             `json:"currency"`
	CostEstimates    []CostEstimate     `json:"cost_estimates"`
	Optimizations    []CostOptimization `json:"optimizations"`
	Summary          CostSummary        `json:"summary"`
	PotentialSavings float64            `json:"potential_savings"`
}

// CostSummary provides statistics about costs
type CostSummary struct {
	TotalResources      int                `json:"total_resources"`
	ResourcesWithCost   int                `json:"resources_with_cost"`
	AverageMonthlyCost  float64            `json:"average_monthly_cost"`
	HighestCostResource string             `json:"highest_cost_resource"`
	LowestCostResource  string             `json:"lowest_cost_resource"`
	CostByProvider      map[string]float64 `json:"cost_by_provider"`
	CostByService       map[string]float64 `json:"cost_by_service"`
}

// AnalyzeCost performs comprehensive cost analysis
func (ca *CostAnalyzer) AnalyzeCost(ctx context.Context) (*CostReport, error) {
	logrus.Info("Starting cost analysis")

	// Get all resources
	resources, err := ca.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	// Calculate cost estimates
	costEstimates := make([]CostEstimate, 0)
	for _, resource := range resources {
		estimate, err := ca.calculateResourceCost(resource)
		if err != nil {
			logrus.Warnf("Failed to calculate cost for resource %s: %v", resource.ID, err)
			continue
		}
		if estimate != nil {
			costEstimates = append(costEstimates, *estimate)
		}
	}

	// Generate optimization recommendations
	optimizations := ca.generateOptimizations(resources, costEstimates)

	// Calculate totals
	totalMonthlyCost := 0.0
	totalDailyCost := 0.0
	totalHourlyCost := 0.0
	for _, estimate := range costEstimates {
		totalMonthlyCost += estimate.MonthlyCost
		totalDailyCost += estimate.DailyCost
		totalHourlyCost += estimate.HourlyCost
	}

	// Calculate potential savings
	potentialSavings := 0.0
	for _, opt := range optimizations {
		potentialSavings += opt.PotentialSavings
	}

	// Calculate summary
	summary := ca.calculateCostSummary(resources, costEstimates)

	report := &CostReport{
		TotalMonthlyCost: totalMonthlyCost,
		TotalDailyCost:   totalDailyCost,
		TotalHourlyCost:  totalHourlyCost,
		Currency:         "USD",
		CostEstimates:    costEstimates,
		Optimizations:    optimizations,
		Summary:          summary,
		PotentialSavings: potentialSavings,
	}

	logrus.Infof("Cost analysis completed: $%.2f/month total cost, $%.2f potential savings",
		totalMonthlyCost, potentialSavings)

	return report, nil
}

// calculateResourceCost calculates the cost for a single resource
func (ca *CostAnalyzer) calculateResourceCost(resource core.Resource) (*CostEstimate, error) {
	// Use existing monthly cost if available
	if resource.MonthlyCost > 0 {
		return &CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  resource.MonthlyCost,
			DailyCost:    resource.MonthlyCost / 30,
			HourlyCost:   resource.MonthlyCost / (30 * 24),
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata: map[string]interface{}{
				"source": "existing_data",
			},
		}, nil
	}

	// Calculate cost based on resource type and provider
	switch resource.Provider {
	case "aws":
		return ca.calculateAWSCost(resource)
	case "azure":
		return ca.calculateAzureCost(resource)
	case "gcp":
		return ca.calculateGCPCost(resource)
	default:
		return nil, fmt.Errorf("unknown provider: %s", resource.Provider)
	}
}

// calculateAWSCost calculates cost for AWS resources
func (ca *CostAnalyzer) calculateAWSCost(resource core.Resource) (*CostEstimate, error) {
	// This is a simplified cost calculation
	// In practice, you'd integrate with AWS Pricing API

	var monthlyCost float64
	var confidence float64 = 0.6

	switch resource.Service {
	case "ec2":
		monthlyCost = ca.calculateEC2Cost(resource)
	case "rds":
		monthlyCost = ca.calculateRDSCost(resource)
	case "s3":
		monthlyCost = ca.calculateS3Cost(resource)
	case "lambda":
		monthlyCost = ca.calculateLambdaCost(resource)
	default:
		// Default cost estimate
		monthlyCost = 10.0
		confidence = 0.3
	}

	return &CostEstimate{
		ResourceID:   resource.ID,
		ResourceARN:  resource.ARN,
		Provider:     resource.Provider,
		Service:      resource.Service,
		Type:         resource.Type,
		Region:       resource.Region,
		MonthlyCost:  monthlyCost,
		DailyCost:    monthlyCost / 30,
		HourlyCost:   monthlyCost / (30 * 24),
		Currency:     "USD",
		PricingModel: "on-demand",
		Confidence:   confidence,
		Metadata: map[string]interface{}{
			"source": "estimated",
			"region": resource.Region,
		},
	}, nil
}

// calculateEC2Cost calculates cost for EC2 instances
func (ca *CostAnalyzer) calculateEC2Cost(resource core.Resource) float64 {
	// Simplified EC2 cost calculation
	// In practice, you'd parse instance type and calculate based on pricing

	config := string(resource.Configuration)

	// Extract instance type from configuration
	instanceType := "t3.micro" // Default
	if strings.Contains(config, "InstanceType") {
		// Parse instance type from JSON
		// This is simplified - in practice, you'd use proper JSON parsing
		if strings.Contains(config, "t3.large") {
			instanceType = "t3.large"
		} else if strings.Contains(config, "t3.medium") {
			instanceType = "t3.medium"
		}
	}

	// Simplified pricing (USD per month)
	pricing := map[string]float64{
		"t3.micro":  8.0,
		"t3.small":  15.0,
		"t3.medium": 30.0,
		"t3.large":  60.0,
		"m5.large":  70.0,
		"m5.xlarge": 140.0,
	}

	if cost, ok := pricing[instanceType]; ok {
		return cost
	}

	return 20.0 // Default cost
}

// calculateRDSCost calculates cost for RDS instances
func (ca *CostAnalyzer) calculateRDSCost(resource core.Resource) float64 {
	// Simplified RDS cost calculation
	config := string(resource.Configuration)

	// Extract instance class from configuration
	instanceClass := "db.t3.micro" // Default
	if strings.Contains(config, "DBInstanceClass") {
		// Parse instance class from JSON
		if strings.Contains(config, "db.t3.small") {
			instanceClass = "db.t3.small"
		} else if strings.Contains(config, "db.t3.medium") {
			instanceClass = "db.t3.medium"
		} else if strings.Contains(config, "db.m5.large") {
			instanceClass = "db.m5.large"
		}
	}

	// Simplified pricing (USD per month)
	pricing := map[string]float64{
		"db.t3.micro":  15.0,
		"db.t3.small":  25.0,
		"db.t3.medium": 50.0,
		"db.m5.large":  120.0,
	}

	if cost, ok := pricing[instanceClass]; ok {
		return cost
	}

	return 30.0 // Default cost
}

// calculateS3Cost calculates cost for S3 buckets
func (ca *CostAnalyzer) calculateS3Cost(resource core.Resource) float64 {
	// S3 cost is typically based on storage and requests
	// This is a simplified calculation

	// Default S3 cost (very low for most use cases)
	return 5.0
}

// calculateLambdaCost calculates cost for Lambda functions
func (ca *CostAnalyzer) calculateLambdaCost(resource core.Resource) float64 {
	// Lambda cost is based on requests and duration
	// This is a simplified calculation

	// Default Lambda cost (very low for most use cases)
	return 2.0
}

// calculateAzureCost calculates cost for Azure resources
func (ca *CostAnalyzer) calculateAzureCost(resource core.Resource) (*CostEstimate, error) {
	// Simplified Azure cost calculation
	var monthlyCost float64
	var confidence float64 = 0.6

	switch resource.Service {
	case "compute":
		monthlyCost = ca.calculateAzureComputeCost(resource)
	case "storage":
		monthlyCost = ca.calculateAzureStorageCost(resource)
	default:
		monthlyCost = 15.0
		confidence = 0.3
	}

	return &CostEstimate{
		ResourceID:   resource.ID,
		ResourceARN:  resource.ARN,
		Provider:     resource.Provider,
		Service:      resource.Service,
		Type:         resource.Type,
		Region:       resource.Region,
		MonthlyCost:  monthlyCost,
		DailyCost:    monthlyCost / 30,
		HourlyCost:   monthlyCost / (30 * 24),
		Currency:     "USD",
		PricingModel: "on-demand",
		Confidence:   confidence,
		Metadata: map[string]interface{}{
			"source": "estimated",
			"region": resource.Region,
		},
	}, nil
}

// calculateAzureComputeCost calculates cost for Azure compute resources
func (ca *CostAnalyzer) calculateAzureComputeCost(resource core.Resource) float64 {
	// Simplified Azure VM cost calculation
	return 25.0 // Default cost
}

// calculateAzureStorageCost calculates cost for Azure storage resources
func (ca *CostAnalyzer) calculateAzureStorageCost(resource core.Resource) float64 {
	// Simplified Azure storage cost calculation
	return 8.0 // Default cost
}

// calculateGCPCost calculates cost for GCP resources
func (ca *CostAnalyzer) calculateGCPCost(resource core.Resource) (*CostEstimate, error) {
	// Simplified GCP cost calculation
	var monthlyCost float64
	var confidence float64 = 0.6

	switch resource.Service {
	case "compute":
		monthlyCost = ca.calculateGCPComputeCost(resource)
	case "storage":
		monthlyCost = ca.calculateGCPStorageCost(resource)
	default:
		monthlyCost = 12.0
		confidence = 0.3
	}

	return &CostEstimate{
		ResourceID:   resource.ID,
		ResourceARN:  resource.ARN,
		Provider:     resource.Provider,
		Service:      resource.Service,
		Type:         resource.Type,
		Region:       resource.Region,
		MonthlyCost:  monthlyCost,
		DailyCost:    monthlyCost / 30,
		HourlyCost:   monthlyCost / (30 * 24),
		Currency:     "USD",
		PricingModel: "on-demand",
		Confidence:   confidence,
		Metadata: map[string]interface{}{
			"source": "estimated",
			"region": resource.Region,
		},
	}, nil
}

// calculateGCPComputeCost calculates cost for GCP compute resources
func (ca *CostAnalyzer) calculateGCPComputeCost(resource core.Resource) float64 {
	// Simplified GCP VM cost calculation
	return 20.0 // Default cost
}

// calculateGCPStorageCost calculates cost for GCP storage resources
func (ca *CostAnalyzer) calculateGCPStorageCost(resource core.Resource) float64 {
	// Simplified GCP storage cost calculation
	return 6.0 // Default cost
}

// generateOptimizations generates cost optimization recommendations
func (ca *CostAnalyzer) generateOptimizations(resources []core.Resource, estimates []CostEstimate) []CostOptimization {
	optimizations := make([]CostOptimization, 0)

	// Create cost lookup map
	costMap := make(map[string]CostEstimate)
	for _, estimate := range estimates {
		costMap[estimate.ResourceID] = estimate
	}

	// Generate optimizations for each resource
	for _, resource := range resources {
		estimate, exists := costMap[resource.ID]
		if !exists {
			continue
		}

		// Check for unused resources
		if ca.isResourceUnused(resource) {
			optimizations = append(optimizations, CostOptimization{
				ID:               fmt.Sprintf("unused-resource-%s", resource.ID),
				ResourceID:       resource.ID,
				ResourceARN:      resource.ARN,
				Provider:         resource.Provider,
				Service:          resource.Service,
				Type:             resource.Type,
				Title:            "Unused Resource",
				Description:      fmt.Sprintf("Resource %s appears to be unused", resource.Name),
				CurrentCost:      estimate.MonthlyCost,
				PotentialSavings: estimate.MonthlyCost,
				SavingsPercent:   100.0,
				Priority:         "high",
				Category:         "unused",
				Recommendation:   "Consider deleting this resource if it's no longer needed",
				Implementation:   "Delete the resource through the cloud console or API",
				Metadata: map[string]interface{}{
					"resource_name": resource.Name,
					"region":        resource.Region,
				},
			})
		}

		// Check for right-sizing opportunities
		if ca.isResourceOversized(resource, estimate) {
			optimizations = append(optimizations, CostOptimization{
				ID:               fmt.Sprintf("rightsize-%s", resource.ID),
				ResourceID:       resource.ID,
				ResourceARN:      resource.ARN,
				Provider:         resource.Provider,
				Service:          resource.Service,
				Type:             resource.Type,
				Title:            "Right-sizing Opportunity",
				Description:      fmt.Sprintf("Resource %s may be oversized for its workload", resource.Name),
				CurrentCost:      estimate.MonthlyCost,
				PotentialSavings: estimate.MonthlyCost * 0.3, // Assume 30% savings
				SavingsPercent:   30.0,
				Priority:         "medium",
				Category:         "rightsizing",
				Recommendation:   "Consider downsizing to a smaller instance type",
				Implementation:   "Monitor resource utilization and resize accordingly",
				Metadata: map[string]interface{}{
					"resource_name": resource.Name,
					"region":        resource.Region,
				},
			})
		}

		// Check for reserved instance opportunities
		if ca.isReservedInstanceCandidate(resource, estimate) {
			optimizations = append(optimizations, CostOptimization{
				ID:               fmt.Sprintf("reserved-instance-%s", resource.ID),
				ResourceID:       resource.ID,
				ResourceARN:      resource.ARN,
				Provider:         resource.Provider,
				Service:          resource.Service,
				Type:             resource.Type,
				Title:            "Reserved Instance Opportunity",
				Description:      fmt.Sprintf("Resource %s could benefit from reserved instance pricing", resource.Name),
				CurrentCost:      estimate.MonthlyCost,
				PotentialSavings: estimate.MonthlyCost * 0.4, // Assume 40% savings
				SavingsPercent:   40.0,
				Priority:         "medium",
				Category:         "reserved",
				Recommendation:   "Purchase reserved instances for predictable workloads",
				Implementation:   "Buy reserved instances through the cloud console",
				Metadata: map[string]interface{}{
					"resource_name": resource.Name,
					"region":        resource.Region,
				},
			})
		}
	}

	return optimizations
}

// isResourceUnused checks if a resource appears to be unused
func (ca *CostAnalyzer) isResourceUnused(resource core.Resource) bool {
	// This is a simplified check
	// In practice, you'd analyze usage metrics, logs, etc.

	// Check if resource has been created recently (less than 7 days)
	// This would require parsing the CreatedAt field properly

	// For now, return false as we don't have enough data
	return false
}

// isResourceOversized checks if a resource appears to be oversized
func (ca *CostAnalyzer) isResourceOversized(resource core.Resource, estimate CostEstimate) bool {
	// This is a simplified check
	// In practice, you'd analyze CPU, memory, and network utilization

	// Check if resource is expensive but might be oversized
	return estimate.MonthlyCost > 50.0 && resource.Service == "ec2"
}

// isReservedInstanceCandidate checks if a resource is a good candidate for reserved instances
func (ca *CostAnalyzer) isReservedInstanceCandidate(resource core.Resource, estimate CostEstimate) bool {
	// This is a simplified check
	// In practice, you'd analyze usage patterns and predictability

	// Check if resource is expensive and likely to run continuously
	return estimate.MonthlyCost > 100.0 && resource.Service == "ec2"
}

// calculateCostSummary calculates the cost summary
func (ca *CostAnalyzer) calculateCostSummary(resources []core.Resource, estimates []CostEstimate) CostSummary {
	summary := CostSummary{
		TotalResources:    len(resources),
		ResourcesWithCost: len(estimates),
		CostByProvider:    make(map[string]float64),
		CostByService:     make(map[string]float64),
	}

	if len(estimates) == 0 {
		return summary
	}

	// Calculate average cost
	totalCost := 0.0
	for _, estimate := range estimates {
		totalCost += estimate.MonthlyCost
		summary.CostByProvider[estimate.Provider] += estimate.MonthlyCost
		summary.CostByService[estimate.Service] += estimate.MonthlyCost
	}
	summary.AverageMonthlyCost = totalCost / float64(len(estimates))

	// Find highest and lowest cost resources
	if len(estimates) > 0 {
		highest := estimates[0]
		lowest := estimates[0]

		for _, estimate := range estimates {
			if estimate.MonthlyCost > highest.MonthlyCost {
				highest = estimate
			}
			if estimate.MonthlyCost < lowest.MonthlyCost {
				lowest = estimate
			}
		}

		summary.HighestCostResource = highest.ResourceID
		summary.LowestCostResource = lowest.ResourceID
	}

	return summary
}
