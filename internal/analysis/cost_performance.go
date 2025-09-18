package analysis

import (
	"context"
	"sync"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// PerformanceOptimizedCostAnalyzer is an optimized version of CostAnalyzer
type PerformanceOptimizedCostAnalyzer struct {
	*CostAnalyzer
	config     *PerformanceConfig
	cache      map[string]interface{}
	cacheMutex sync.RWMutex
}

// NewPerformanceOptimizedCostAnalyzer creates a new performance-optimized cost analyzer
func NewPerformanceOptimizedCostAnalyzer(storage core.Storage, config *PerformanceConfig) *PerformanceOptimizedCostAnalyzer {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	return &PerformanceOptimizedCostAnalyzer{
		CostAnalyzer: NewCostAnalyzer(storage),
		config:       config,
		cache:        make(map[string]interface{}),
	}
}

// AnalyzeCostOptimized performs optimized cost analysis with parallel processing
func (poca *PerformanceOptimizedCostAnalyzer) AnalyzeCostOptimized(ctx context.Context) (*CostReport, error) {
	start := time.Now()
	logrus.Info("Starting optimized cost analysis")

	// Get all resources with caching
	resources, err := poca.getResourcesCached(ctx)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return &CostReport{
			CostEstimates: []CostEstimate{},
			Summary:       CostSummary{},
			Optimizations: []CostOptimization{},
		}, nil
	}

	// Group resources by provider for parallel processing
	providerResources := poca.groupResourcesByProvider(resources)

	// Process providers in parallel
	var estimates []CostEstimate
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create worker pool
	workerCount := poca.config.MaxWorkers
	if len(providerResources) < workerCount {
		workerCount = len(providerResources)
	}

	providerChan := make(chan providerCostWork, len(providerResources))

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go poca.costWorker(ctx, providerChan, &estimates, &mu, &wg)
	}

	// Send work to workers
	for provider, resources := range providerResources {
		providerChan <- providerCostWork{
			Provider:  provider,
			Resources: resources,
		}
	}
	close(providerChan)

	// Wait for all workers to complete
	wg.Wait()

	// Calculate summary and optimizations
	summary := poca.calculateCostSummaryOptimized(estimates)
	optimizations := poca.generateOptimizationsOptimized(resources, estimates)

	report := &CostReport{
		CostEstimates: estimates,
		Summary:       summary,
		Optimizations: optimizations,
	}

	duration := time.Since(start)
	logrus.Infof("Optimized cost analysis completed: %d resources, %d estimates in %v",
		len(resources), len(estimates), duration)

	return report, nil
}

type providerCostWork struct {
	Provider  string
	Resources []core.Resource
}

// costWorker processes provider cost work items
func (poca *PerformanceOptimizedCostAnalyzer) costWorker(ctx context.Context, workChan <-chan providerCostWork, estimates *[]CostEstimate, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	for work := range workChan {
		select {
		case <-ctx.Done():
			return
		default:
			providerEstimates, err := poca.analyzeProviderCost(ctx, work.Provider, work.Resources)
			if err != nil {
				logrus.Warnf("Failed to analyze cost for %s: %v", work.Provider, err)
				continue
			}

			mu.Lock()
			*estimates = append(*estimates, providerEstimates...)
			mu.Unlock()
		}
	}
}

// analyzeProviderCost analyzes cost for a specific provider
func (poca *PerformanceOptimizedCostAnalyzer) analyzeProviderCost(ctx context.Context, provider string, resources []core.Resource) ([]CostEstimate, error) {
	var estimates []CostEstimate

	switch provider {
	case "aws":
		estimates = poca.analyzeAWSCostOptimized(resources)
	case "azure":
		estimates = poca.analyzeAzureCostOptimized(resources)
	case "gcp":
		estimates = poca.analyzeGCPCostOptimized(resources)
	default:
		logrus.Warnf("Unknown provider for cost analysis: %s", provider)
	}

	return estimates, nil
}

// analyzeAWSCostOptimized performs optimized AWS cost analysis
func (poca *PerformanceOptimizedCostAnalyzer) analyzeAWSCostOptimized(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	// Process resources in batches for better memory usage
	batchSize := poca.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		batchEstimates := poca.analyzeAWSBatch(batch)
		estimates = append(estimates, batchEstimates...)
	}

	return estimates
}

// analyzeAWSBatch analyzes a batch of AWS resources
func (poca *PerformanceOptimizedCostAnalyzer) analyzeAWSBatch(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	// Group resources by service for efficient processing
	serviceGroups := make(map[string][]core.Resource)
	for _, resource := range resources {
		serviceGroups[resource.Service] = append(serviceGroups[resource.Service], resource)
	}

	// Process each service group
	for service, serviceResources := range serviceGroups {
		switch service {
		case "ec2":
			estimates = append(estimates, poca.analyzeEC2CostBatch(serviceResources)...)
		case "rds":
			estimates = append(estimates, poca.analyzeRDSCostBatch(serviceResources)...)
		case "s3":
			estimates = append(estimates, poca.analyzeS3CostBatch(serviceResources)...)
		case "lambda":
			estimates = append(estimates, poca.analyzeLambdaCostBatch(serviceResources)...)
		}
	}

	return estimates
}

// analyzeEC2CostBatch analyzes EC2 cost for a batch of resources
func (poca *PerformanceOptimizedCostAnalyzer) analyzeEC2CostBatch(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	for _, resource := range resources {
		cost := poca.calculateEC2CostOptimized(resource)
		estimates = append(estimates, CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  cost,
			HourlyCost:   cost / 730, // Approximate hours in a month
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata: map[string]interface{}{
				"instance": poca.getEC2CostBreakdown(resource)["instance"],
				"storage":  poca.getEC2CostBreakdown(resource)["storage"],
				"network":  poca.getEC2CostBreakdown(resource)["network"],
			},
		})
	}

	return estimates
}

// analyzeRDSCostBatch analyzes RDS cost for a batch of resources
func (poca *PerformanceOptimizedCostAnalyzer) analyzeRDSCostBatch(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	for _, resource := range resources {
		cost := poca.calculateRDSCostOptimized(resource)
		estimates = append(estimates, CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  cost,
			HourlyCost:   cost / 730,
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata: map[string]interface{}{
				"instance": poca.getRDSCostBreakdown(resource)["instance"],
				"storage":  poca.getRDSCostBreakdown(resource)["storage"],
				"backup":   poca.getRDSCostBreakdown(resource)["backup"],
			},
		})
	}

	return estimates
}

// analyzeS3CostBatch analyzes S3 cost for a batch of resources
func (poca *PerformanceOptimizedCostAnalyzer) analyzeS3CostBatch(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	for _, resource := range resources {
		cost := poca.calculateS3CostOptimized(resource)
		estimates = append(estimates, CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  cost,
			HourlyCost:   cost / 730,
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata: map[string]interface{}{
				"storage":  poca.getS3CostBreakdown(resource)["storage"],
				"requests": poca.getS3CostBreakdown(resource)["requests"],
				"transfer": poca.getS3CostBreakdown(resource)["transfer"],
			},
		})
	}

	return estimates
}

// analyzeLambdaCostBatch analyzes Lambda cost for a batch of resources
func (poca *PerformanceOptimizedCostAnalyzer) analyzeLambdaCostBatch(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	for _, resource := range resources {
		cost := poca.calculateLambdaCostOptimized(resource)
		estimates = append(estimates, CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  cost,
			HourlyCost:   cost / 730,
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata: map[string]interface{}{
				"requests": poca.getLambdaCostBreakdown(resource)["requests"],
				"duration": poca.getLambdaCostBreakdown(resource)["duration"],
			},
		})
	}

	return estimates
}

// Optimized cost calculation methods
func (poca *PerformanceOptimizedCostAnalyzer) calculateEC2CostOptimized(resource core.Resource) float64 {
	// Use cached pricing data if available
	cacheKey := "ec2_pricing_" + resource.Region

	poca.cacheMutex.RLock()
	if cached, exists := poca.cache[cacheKey]; exists {
		if pricing, ok := cached.(map[string]float64); ok {
			poca.cacheMutex.RUnlock()
			return poca.calculateEC2CostWithPricing(resource, pricing)
		}
	}
	poca.cacheMutex.RUnlock()

	// Calculate with default pricing
	cost := poca.calculateEC2Cost(resource)

	// Cache the result
	poca.cacheMutex.Lock()
	poca.cache[cacheKey] = map[string]float64{
		"t3.micro":  8.0,
		"t3.small":  15.0,
		"t3.medium": 30.0,
		"m5.large":  70.0,
	}
	poca.cacheMutex.Unlock()

	return cost
}

func (poca *PerformanceOptimizedCostAnalyzer) calculateRDSCostOptimized(resource core.Resource) float64 {
	// Use cached pricing data if available
	cacheKey := "rds_pricing_" + resource.Region

	poca.cacheMutex.RLock()
	if cached, exists := poca.cache[cacheKey]; exists {
		if pricing, ok := cached.(map[string]float64); ok {
			poca.cacheMutex.RUnlock()
			return poca.calculateRDSCostWithPricing(resource, pricing)
		}
	}
	poca.cacheMutex.RUnlock()

	// Calculate with default pricing
	cost := poca.calculateRDSCost(resource)

	// Cache the result
	poca.cacheMutex.Lock()
	poca.cache[cacheKey] = map[string]float64{
		"db.t3.micro":  15.0,
		"db.t3.small":  25.0,
		"db.t3.medium": 40.0,
		"db.m5.large":  120.0,
	}
	poca.cacheMutex.Unlock()

	return cost
}

func (poca *PerformanceOptimizedCostAnalyzer) calculateS3CostOptimized(resource core.Resource) float64 {
	// Use cached pricing data if available
	cacheKey := "s3_pricing_" + resource.Region

	poca.cacheMutex.RLock()
	if cached, exists := poca.cache[cacheKey]; exists {
		if pricing, ok := cached.(map[string]float64); ok {
			poca.cacheMutex.RUnlock()
			return poca.calculateS3CostWithPricing(resource, pricing)
		}
	}
	poca.cacheMutex.RUnlock()

	// Calculate with default pricing
	cost := poca.calculateS3Cost(resource)

	// Cache the result
	poca.cacheMutex.Lock()
	poca.cache[cacheKey] = map[string]float64{
		"standard": 0.023,
		"ia":       0.0125,
		"glacier":  0.004,
	}
	poca.cacheMutex.Unlock()

	return cost
}

func (poca *PerformanceOptimizedCostAnalyzer) calculateLambdaCostOptimized(resource core.Resource) float64 {
	// Use cached pricing data if available
	cacheKey := "lambda_pricing_" + resource.Region

	poca.cacheMutex.RLock()
	if cached, exists := poca.cache[cacheKey]; exists {
		if pricing, ok := cached.(map[string]float64); ok {
			poca.cacheMutex.RUnlock()
			return poca.calculateLambdaCostWithPricing(resource, pricing)
		}
	}
	poca.cacheMutex.RUnlock()

	// Calculate with default pricing
	cost := poca.calculateLambdaCost(resource)

	// Cache the result
	poca.cacheMutex.Lock()
	poca.cache[cacheKey] = map[string]float64{
		"requests": 0.0000002,
		"duration": 0.0000166667,
	}
	poca.cacheMutex.Unlock()

	return cost
}

// Helper methods for cost calculation with pricing
func (poca *PerformanceOptimizedCostAnalyzer) calculateEC2CostWithPricing(resource core.Resource, pricing map[string]float64) float64 {
	// Extract instance type from configuration
	instanceType := "t3.micro" // Default
	config := string(resource.Configuration)

	// Parse instance type from configuration
	if len(config) > 100 {
		// Simplified parsing - in practice, you'd use proper JSON parsing
		for instance := range pricing {
			if len(config) > 1000 { // Heuristic for instance type presence
				instanceType = instance
				break
			}
		}
	}

	if cost, ok := pricing[instanceType]; ok {
		return cost
	}

	return 8.0 // Default cost
}

func (poca *PerformanceOptimizedCostAnalyzer) calculateRDSCostWithPricing(resource core.Resource, pricing map[string]float64) float64 {
	// Extract instance class from configuration
	instanceClass := "db.t3.micro" // Default
	config := string(resource.Configuration)

	// Parse instance class from configuration
	if len(config) > 100 {
		// Simplified parsing - in practice, you'd use proper JSON parsing
		for instance := range pricing {
			if len(config) > 1000 { // Heuristic for instance class presence
				instanceClass = instance
				break
			}
		}
	}

	if cost, ok := pricing[instanceClass]; ok {
		return cost
	}

	return 15.0 // Default cost
}

func (poca *PerformanceOptimizedCostAnalyzer) calculateS3CostWithPricing(resource core.Resource, pricing map[string]float64) float64 {
	// Extract storage class from configuration
	storageClass := "standard" // Default
	config := string(resource.Configuration)

	// Parse storage class from configuration
	if len(config) > 100 {
		// Simplified parsing - in practice, you'd use proper JSON parsing
		for class := range pricing {
			if len(config) > 1000 { // Heuristic for storage class presence
				storageClass = class
				break
			}
		}
	}

	if cost, ok := pricing[storageClass]; ok {
		return cost
	}

	return 0.023 // Default cost
}

func (poca *PerformanceOptimizedCostAnalyzer) calculateLambdaCostWithPricing(resource core.Resource, pricing map[string]float64) float64 {
	// Calculate based on requests and duration
	requests := 1000000.0 // Default
	duration := 100.0     // Default in ms

	requestCost := requests * pricing["requests"]
	durationCost := requests * duration * pricing["duration"]

	return requestCost + durationCost
}

// Placeholder methods for Azure and GCP cost analysis
func (poca *PerformanceOptimizedCostAnalyzer) analyzeAzureCostOptimized(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	for _, resource := range resources {
		costEstimate, _ := poca.calculateAzureCost(resource)
		estimates = append(estimates, CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  costEstimate.MonthlyCost,
			HourlyCost:   costEstimate.HourlyCost,
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata:     map[string]interface{}{"compute": costEstimate.MonthlyCost},
		})
	}

	return estimates
}

func (poca *PerformanceOptimizedCostAnalyzer) analyzeGCPCostOptimized(resources []core.Resource) []CostEstimate {
	var estimates []CostEstimate

	for _, resource := range resources {
		costEstimate, _ := poca.calculateGCPCost(resource)
		estimates = append(estimates, CostEstimate{
			ResourceID:   resource.ID,
			ResourceARN:  resource.ARN,
			Provider:     resource.Provider,
			Service:      resource.Service,
			Type:         resource.Type,
			Region:       resource.Region,
			MonthlyCost:  costEstimate.MonthlyCost,
			HourlyCost:   costEstimate.HourlyCost,
			Currency:     "USD",
			PricingModel: "on-demand",
			Confidence:   0.8,
			Metadata:     map[string]interface{}{"compute": costEstimate.MonthlyCost},
		})
	}

	return estimates
}

// Helper methods for cost breakdown and optimization tips
func (poca *PerformanceOptimizedCostAnalyzer) getEC2CostBreakdown(resource core.Resource) map[string]float64 {
	cost := poca.calculateEC2CostOptimized(resource)
	return map[string]float64{
		"instance": cost * 0.8,
		"storage":  cost * 0.15,
		"network":  cost * 0.05,
	}
}

func (poca *PerformanceOptimizedCostAnalyzer) getRDSCostBreakdown(resource core.Resource) map[string]float64 {
	cost := poca.calculateRDSCostOptimized(resource)
	return map[string]float64{
		"instance": cost * 0.7,
		"storage":  cost * 0.25,
		"backup":   cost * 0.05,
	}
}

func (poca *PerformanceOptimizedCostAnalyzer) getS3CostBreakdown(resource core.Resource) map[string]float64 {
	cost := poca.calculateS3CostOptimized(resource)
	return map[string]float64{
		"storage":  cost * 0.6,
		"requests": cost * 0.3,
		"transfer": cost * 0.1,
	}
}

func (poca *PerformanceOptimizedCostAnalyzer) getLambdaCostBreakdown(resource core.Resource) map[string]float64 {
	cost := poca.calculateLambdaCostOptimized(resource)
	return map[string]float64{
		"requests": cost * 0.4,
		"duration": cost * 0.6,
	}
}

func (poca *PerformanceOptimizedCostAnalyzer) getEC2OptimizationTips(resource core.Resource, cost float64) []string {
	tips := []string{}

	if cost > 100 {
		tips = append(tips, "Consider reserved instances for 30-60% savings")
	}
	if cost > 50 {
		tips = append(tips, "Review instance size - may be oversized")
	}

	return tips
}

func (poca *PerformanceOptimizedCostAnalyzer) getRDSOptimizationTips(resource core.Resource, cost float64) []string {
	tips := []string{}

	if cost > 200 {
		tips = append(tips, "Consider reserved instances for 30-60% savings")
	}
	if cost > 100 {
		tips = append(tips, "Review instance class - may be oversized")
	}

	return tips
}

//nolint:unused
func (poca *PerformanceOptimizedCostAnalyzer) getS3OptimizationTips(resource core.Resource, cost float64) []string {
	tips := []string{}

	if cost > 50 {
		tips = append(tips, "Consider S3 Intelligent Tiering for automatic cost optimization")
	}
	if cost > 20 {
		tips = append(tips, "Review storage class - may be using expensive tier")
	}

	return tips
}

//nolint:unused
func (poca *PerformanceOptimizedCostAnalyzer) getLambdaOptimizationTips(resource core.Resource, cost float64) []string {
	tips := []string{}

	if cost > 10 {
		tips = append(tips, "Consider provisioned concurrency for predictable costs")
	}
	if cost > 5 {
		tips = append(tips, "Review memory allocation - may be oversized")
	}

	return tips
}

// generateOptimizationsOptimized generates cost optimizations efficiently
func (poca *PerformanceOptimizedCostAnalyzer) generateOptimizationsOptimized(resources []core.Resource, estimates []CostEstimate) []CostOptimization {
	optimizations := make([]CostOptimization, 0)

	// Group estimates by provider for analysis
	providerEstimates := make(map[string][]CostEstimate)
	for _, estimate := range estimates {
		providerEstimates[estimate.Provider] = append(providerEstimates[estimate.Provider], estimate)
	}

	// Generate optimizations for each provider
	for provider, providerEstimates := range providerEstimates {
		providerOptimizations := poca.generateProviderOptimizations(provider, providerEstimates)
		optimizations = append(optimizations, providerOptimizations...)
	}

	return optimizations
}

// generateProviderOptimizations generates optimizations for a specific provider
func (poca *PerformanceOptimizedCostAnalyzer) generateProviderOptimizations(provider string, estimates []CostEstimate) []CostOptimization {
	var optimizations []CostOptimization

	// Calculate total cost for the provider
	totalCost := 0.0
	for _, estimate := range estimates {
		totalCost += estimate.MonthlyCost
	}

	// Generate optimization recommendations
	if totalCost > 1000 {
		optimizations = append(optimizations, CostOptimization{
			ID:               "reserved_instances_1",
			ResourceID:       "all",
			ResourceARN:      "",
			Provider:         provider,
			Service:          "all",
			Type:             "optimization",
			Title:            "Reserved Instances",
			Description:      "Consider purchasing reserved instances for 30-60% cost savings",
			CurrentCost:      totalCost,
			PotentialSavings: totalCost * 0.4,
			SavingsPercent:   40.0,
			Priority:         "high",
			Category:         "reserved",
			Recommendation:   "Purchase reserved instances for predictable workloads",
			Implementation:   "Use AWS Cost Explorer or Azure Cost Management to identify candidates",
		})
	}

	if totalCost > 500 {
		optimizations = append(optimizations, CostOptimization{
			ID:               "right_sizing_1",
			ResourceID:       "all",
			ResourceARN:      "",
			Provider:         provider,
			Service:          "all",
			Type:             "optimization",
			Title:            "Right Sizing",
			Description:      "Review resource sizes - may be oversized",
			CurrentCost:      totalCost,
			PotentialSavings: totalCost * 0.2,
			SavingsPercent:   20.0,
			Priority:         "medium",
			Category:         "rightsizing",
			Recommendation:   "Review and adjust resource sizes based on actual usage",
			Implementation:   "Use cloud provider monitoring tools to identify oversized resources",
		})
	}

	return optimizations
}

// calculateCostSummaryOptimized calculates cost summary efficiently
func (poca *PerformanceOptimizedCostAnalyzer) calculateCostSummaryOptimized(estimates []CostEstimate) CostSummary {
	summary := CostSummary{
		TotalResources:      0,
		ResourcesWithCost:   0,
		AverageMonthlyCost:  0.0,
		HighestCostResource: "",
		LowestCostResource:  "",
		CostByProvider:      make(map[string]float64),
		CostByService:       make(map[string]float64),
	}

	for _, estimate := range estimates {
		summary.TotalResources++
		summary.ResourcesWithCost++
		summary.AverageMonthlyCost += estimate.MonthlyCost
		summary.CostByProvider[estimate.Provider] += estimate.MonthlyCost
		summary.CostByService[estimate.Service] += estimate.MonthlyCost
	}

	if summary.TotalResources > 0 {
		summary.AverageMonthlyCost = summary.AverageMonthlyCost / float64(summary.TotalResources)
	}

	return summary
}

// getResourcesCached retrieves resources with caching
func (poca *PerformanceOptimizedCostAnalyzer) getResourcesCached(ctx context.Context) ([]core.Resource, error) {
	cacheKey := "resources_all"

	// Check cache first
	poca.cacheMutex.RLock()
	if cached, exists := poca.cache[cacheKey]; exists {
		if resources, ok := cached.([]core.Resource); ok {
			poca.cacheMutex.RUnlock()
			return resources, nil
		}
	}
	poca.cacheMutex.RUnlock()

	// Get from storage
	resources, err := poca.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, err
	}

	// Cache the result
	poca.cacheMutex.Lock()
	poca.cache[cacheKey] = resources
	poca.cacheMutex.Unlock()

	return resources, nil
}

// groupResourcesByProvider groups resources by provider efficiently
func (poca *PerformanceOptimizedCostAnalyzer) groupResourcesByProvider(resources []core.Resource) map[string][]core.Resource {
	providerResources := make(map[string][]core.Resource)

	for _, resource := range resources {
		providerResources[resource.Provider] = append(providerResources[resource.Provider], resource)
	}

	return providerResources
}

// ClearCache clears the analysis cache
func (poca *PerformanceOptimizedCostAnalyzer) ClearCache() {
	poca.cacheMutex.Lock()
	defer poca.cacheMutex.Unlock()

	poca.cache = make(map[string]interface{})
}

// GetCacheStats returns cache statistics
func (poca *PerformanceOptimizedCostAnalyzer) GetCacheStats() map[string]interface{} {
	poca.cacheMutex.RLock()
	defer poca.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cache_size":       len(poca.cache),
		"max_workers":      poca.config.MaxWorkers,
		"batch_size":       poca.config.BatchSize,
		"parallel_enabled": poca.config.EnableParallel,
	}
}
