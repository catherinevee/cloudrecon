package analysis

import (
	"context"
	"sync"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// ComprehensiveAnalysisReport represents a comprehensive analysis report with all components
type ComprehensiveAnalysisReport struct {
	Resources    []core.Resource  `json:"resources"`
	Dependencies *DependencyGraph `json:"dependencies"`
	Security     *SecurityReport  `json:"security"`
	Cost         *CostReport      `json:"cost"`
	Summary      AnalysisSummary  `json:"summary"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

// AnalysisInsights provides high-level insights from the analysis
type AnalysisInsights struct {
	KeyFindings      []string                  `json:"key_findings"`
	Recommendations  []string                  `json:"recommendations"`
	RiskAssessment   *RiskAssessment           `json:"risk_assessment"`
	CostOptimization *CostOptimizationInsights `json:"cost_optimization"`
	SecurityPosture  *SecurityPostureInsights  `json:"security_posture"`
	GeneratedAt      time.Time                 `json:"generated_at"`
}

// RiskAssessment provides risk assessment information
type RiskAssessment struct {
	OverallRisk          string   `json:"overall_risk"`
	RiskFactors          []string `json:"risk_factors"`
	MitigationStrategies []string `json:"mitigation_strategies"`
}

// CostOptimizationInsights provides cost optimization insights
type CostOptimizationInsights struct {
	TotalPotentialSavings float64  `json:"total_potential_savings"`
	TopOptimizations      []string `json:"top_optimizations"`
	CostTrends            []string `json:"cost_trends"`
}

// SecurityPostureInsights provides security posture insights
type SecurityPostureInsights struct {
	ComplianceScore float64  `json:"compliance_score"`
	RiskScore       float64  `json:"risk_score"`
	TopThreats      []string `json:"top_threats"`
	SecurityTrends  []string `json:"security_trends"`
}

// PerformanceOptimizedAnalysisOrchestrator is an optimized version of AnalysisOrchestrator
type PerformanceOptimizedAnalysisOrchestrator struct {
	*AnalysisOrchestrator
	config     *PerformanceConfig
	cache      map[string]interface{}
	cacheMutex sync.RWMutex
}

// NewPerformanceOptimizedAnalysisOrchestrator creates a new performance-optimized analysis orchestrator
func NewPerformanceOptimizedAnalysisOrchestrator(storage core.Storage, config *PerformanceConfig) *PerformanceOptimizedAnalysisOrchestrator {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	return &PerformanceOptimizedAnalysisOrchestrator{
		AnalysisOrchestrator: NewAnalysisOrchestrator(storage),
		config:               config,
		cache:                make(map[string]interface{}),
	}
}

// AnalyzeAllOptimized performs optimized comprehensive analysis with parallel processing
func (poao *PerformanceOptimizedAnalysisOrchestrator) AnalyzeAllOptimized(ctx context.Context) (*ComprehensiveAnalysisReport, error) {
	start := time.Now()
	logrus.Info("Starting optimized comprehensive analysis")

	// Get all resources with caching
	resources, err := poao.getResourcesCached(ctx)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return &ComprehensiveAnalysisReport{
			Resources:    []core.Resource{},
			Dependencies: &DependencyGraph{},
			Security:     &SecurityReport{},
			Cost:         &CostReport{},
			Summary:      AnalysisSummary{},
			GeneratedAt:  time.Now(),
		}, nil
	}

	// Create performance-optimized analyzers
	dependencyAnalyzer := NewPerformanceOptimizedDependencyAnalyzer(poao.storage, poao.config)
	securityAnalyzer := NewPerformanceOptimizedSecurityAnalyzer(poao.storage, poao.config)
	costAnalyzer := NewPerformanceOptimizedCostAnalyzer(poao.storage, poao.config)

	// Run analyses in parallel
	var dependencyGraph *DependencyGraph
	var securityReport *SecurityReport
	var costReport *CostReport
	var analysisErr error

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Dependency analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		dep, err := dependencyAnalyzer.AnalyzeDependenciesOptimized(ctx)
		mu.Lock()
		dependencyGraph = dep
		if err != nil {
			analysisErr = err
		}
		mu.Unlock()
	}()

	// Security analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		sec, err := securityAnalyzer.AnalyzeSecurityOptimized(ctx)
		mu.Lock()
		securityReport = sec
		if err != nil {
			analysisErr = err
		}
		mu.Unlock()
	}()

	// Cost analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		cost, err := costAnalyzer.AnalyzeCostOptimized(ctx)
		mu.Lock()
		costReport = cost
		if err != nil {
			analysisErr = err
		}
		mu.Unlock()
	}()

	// Wait for all analyses to complete
	wg.Wait()

	if analysisErr != nil {
		return nil, analysisErr
	}

	// Calculate summary
	summary := poao.calculateAnalysisSummaryOptimized(dependencyGraph, securityReport, costReport)

	report := &ComprehensiveAnalysisReport{
		Resources:    resources,
		Dependencies: dependencyGraph,
		Security:     securityReport,
		Cost:         costReport,
		Summary:      summary,
		GeneratedAt:  time.Now(),
	}

	duration := time.Since(start)
	logrus.Infof("Optimized comprehensive analysis completed in %v", duration)

	return report, nil
}

// AnalyzeDependenciesOptimized performs optimized dependency analysis
func (poao *PerformanceOptimizedAnalysisOrchestrator) AnalyzeDependenciesOptimized(ctx context.Context) (*DependencyGraph, error) {
	analyzer := NewPerformanceOptimizedDependencyAnalyzer(poao.storage, poao.config)
	return analyzer.AnalyzeDependenciesOptimized(ctx)
}

// AnalyzeSecurityOptimized performs optimized security analysis
func (poao *PerformanceOptimizedAnalysisOrchestrator) AnalyzeSecurityOptimized(ctx context.Context) (*SecurityReport, error) {
	analyzer := NewPerformanceOptimizedSecurityAnalyzer(poao.storage, poao.config)
	return analyzer.AnalyzeSecurityOptimized(ctx)
}

// AnalyzeCostOptimized performs optimized cost analysis
func (poao *PerformanceOptimizedAnalysisOrchestrator) AnalyzeCostOptimized(ctx context.Context) (*CostReport, error) {
	analyzer := NewPerformanceOptimizedCostAnalyzer(poao.storage, poao.config)
	return analyzer.AnalyzeCostOptimized(ctx)
}

// GetAnalysisInsightsOptimized provides optimized analysis insights
func (poao *PerformanceOptimizedAnalysisOrchestrator) GetAnalysisInsightsOptimized(ctx context.Context) (*AnalysisInsights, error) {
	start := time.Now()
	logrus.Info("Starting optimized analysis insights generation")

	// Get comprehensive analysis
	report, err := poao.AnalyzeAllOptimized(ctx)
	if err != nil {
		return nil, err
	}

	// Generate insights
	insights := &AnalysisInsights{
		KeyFindings:      poao.generateKeyFindingsOptimized(report),
		Recommendations:  poao.generateRecommendationsOptimized(report),
		RiskAssessment:   poao.generateRiskAssessmentOptimized(report),
		CostOptimization: poao.generateCostOptimizationInsightsOptimized(report),
		SecurityPosture:  poao.generateSecurityPostureInsightsOptimized(report),
		GeneratedAt:      time.Now(),
	}

	duration := time.Since(start)
	logrus.Infof("Optimized analysis insights generated in %v", duration)

	return insights, nil
}

// generateKeyFindingsOptimized generates key findings efficiently
func (poao *PerformanceOptimizedAnalysisOrchestrator) generateKeyFindingsOptimized(report *ComprehensiveAnalysisReport) []string {
	var findings []string

	// Dependency findings
	if report.Dependencies != nil {
		if report.Dependencies.Stats.Cycles > 0 {
			findings = append(findings, "Circular dependencies detected in infrastructure")
		}
		if report.Dependencies.Stats.Islands > 0 {
			findings = append(findings, "Isolated resources found that may be unused")
		}
	}

	// Security findings
	if report.Security != nil {
		if report.Security.Summary.TotalFindings > 0 {
			findings = append(findings, "Security issues detected requiring attention")
		}
		if report.Security.ComplianceScore < 80 {
			findings = append(findings, "Low compliance score indicates security gaps")
		}
	}

	// Cost findings
	if report.Cost != nil {
		totalCost := 0.0
		for _, estimate := range report.Cost.CostEstimates {
			totalCost += estimate.MonthlyCost
		}
		if totalCost > 1000 {
			findings = append(findings, "High monthly costs detected - optimization opportunities available")
		}
		if len(report.Cost.Optimizations) > 0 {
			findings = append(findings, "Cost optimization recommendations available")
		}
	}

	return findings
}

// generateRecommendationsOptimized generates recommendations efficiently
func (poao *PerformanceOptimizedAnalysisOrchestrator) generateRecommendationsOptimized(report *ComprehensiveAnalysisReport) []string {
	var recommendations []string

	// Security recommendations
	if report.Security != nil && report.Security.Summary.TotalFindings > 0 {
		recommendations = append(recommendations, "Address security findings to improve compliance score")
		recommendations = append(recommendations, "Implement security best practices for cloud resources")
	}

	// Cost recommendations
	if report.Cost != nil && len(report.Cost.Optimizations) > 0 {
		recommendations = append(recommendations, "Implement cost optimization recommendations")
		recommendations = append(recommendations, "Consider reserved instances for predictable workloads")
	}

	// Dependency recommendations
	if report.Dependencies != nil && report.Dependencies.Stats.Cycles > 0 {
		recommendations = append(recommendations, "Resolve circular dependencies to improve infrastructure stability")
	}

	return recommendations
}

// generateRiskAssessmentOptimized generates risk assessment efficiently
func (poao *PerformanceOptimizedAnalysisOrchestrator) generateRiskAssessmentOptimized(report *ComprehensiveAnalysisReport) *RiskAssessment {
	assessment := &RiskAssessment{
		OverallRisk:          "low",
		RiskFactors:          []string{},
		MitigationStrategies: []string{},
	}

	// Security risk assessment
	if report.Security != nil {
		if report.Security.RiskScore > 70 {
			assessment.OverallRisk = "high"
			assessment.RiskFactors = append(assessment.RiskFactors, "High security risk score")
			assessment.MitigationStrategies = append(assessment.MitigationStrategies, "Implement security hardening measures")
		} else if report.Security.RiskScore > 40 {
			assessment.OverallRisk = "medium"
			assessment.RiskFactors = append(assessment.RiskFactors, "Medium security risk score")
			assessment.MitigationStrategies = append(assessment.MitigationStrategies, "Review and address security findings")
		}
	}

	// Cost risk assessment
	if report.Cost != nil {
		totalCost := 0.0
		for _, estimate := range report.Cost.CostEstimates {
			totalCost += estimate.MonthlyCost
		}
		if totalCost > 5000 {
			assessment.RiskFactors = append(assessment.RiskFactors, "High monthly costs")
			assessment.MitigationStrategies = append(assessment.MitigationStrategies, "Implement cost optimization strategies")
		}
	}

	return assessment
}

// generateCostOptimizationInsightsOptimized generates cost optimization insights efficiently
func (poao *PerformanceOptimizedAnalysisOrchestrator) generateCostOptimizationInsightsOptimized(report *ComprehensiveAnalysisReport) *CostOptimizationInsights {
	insights := &CostOptimizationInsights{
		TotalPotentialSavings: 0.0,
		TopOptimizations:      []string{},
		CostTrends:            []string{},
	}

	if report.Cost != nil {
		// Calculate total potential savings
		for _, opt := range report.Cost.Optimizations {
			insights.TotalPotentialSavings += opt.PotentialSavings
		}

		// Get top optimizations
		for _, opt := range report.Cost.Optimizations {
			if opt.Priority == "high" {
				insights.TopOptimizations = append(insights.TopOptimizations, opt.Description)
			}
		}

		// Generate cost trends
		totalCost := 0.0
		for _, estimate := range report.Cost.CostEstimates {
			totalCost += estimate.MonthlyCost
		}
		if totalCost > 1000 {
			insights.CostTrends = append(insights.CostTrends, "High monthly costs detected")
		}
		if len(report.Cost.Optimizations) > 0 {
			insights.CostTrends = append(insights.CostTrends, "Optimization opportunities available")
		}
	}

	return insights
}

// generateSecurityPostureInsightsOptimized generates security posture insights efficiently
func (poao *PerformanceOptimizedAnalysisOrchestrator) generateSecurityPostureInsightsOptimized(report *ComprehensiveAnalysisReport) *SecurityPostureInsights {
	insights := &SecurityPostureInsights{
		ComplianceScore: 100.0,
		RiskScore:       0.0,
		TopThreats:      []string{},
		SecurityTrends:  []string{},
	}

	if report.Security != nil {
		insights.ComplianceScore = report.Security.ComplianceScore
		insights.RiskScore = report.Security.RiskScore

		// Get top threats
		for _, finding := range report.Security.Findings {
			if finding.Severity == "critical" || finding.Severity == "high" {
				insights.TopThreats = append(insights.TopThreats, finding.Title)
			}
		}

		// Generate security trends
		if report.Security.ComplianceScore < 80 {
			insights.SecurityTrends = append(insights.SecurityTrends, "Low compliance score")
		}
		if report.Security.RiskScore > 50 {
			insights.SecurityTrends = append(insights.SecurityTrends, "High risk score")
		}
	}

	return insights
}

// calculateAnalysisSummaryOptimized calculates analysis summary efficiently
func (poao *PerformanceOptimizedAnalysisOrchestrator) calculateAnalysisSummaryOptimized(dependencyGraph *DependencyGraph, securityReport *SecurityReport, costReport *CostReport) AnalysisSummary {
	summary := AnalysisSummary{
		TotalResources:    0,
		TotalDependencies: 0,
		SecurityFindings:  0,
		CriticalFindings:  0,
		TotalMonthlyCost:  0.0,
		PotentialSavings:  0.0,
		ComplianceScore:   100.0,
		RiskScore:         0.0,
		AnalysisDuration:  "",
	}

	// Dependency summary
	if dependencyGraph != nil {
		summary.TotalResources = dependencyGraph.Stats.TotalResources
		summary.TotalDependencies = dependencyGraph.Stats.TotalDependencies
	}

	// Security summary
	if securityReport != nil {
		summary.SecurityFindings = securityReport.Summary.TotalFindings
		summary.ComplianceScore = securityReport.ComplianceScore
		summary.RiskScore = securityReport.RiskScore
	}

	// Cost summary
	if costReport != nil {
		for _, estimate := range costReport.CostEstimates {
			summary.TotalMonthlyCost += estimate.MonthlyCost
		}
		for _, opt := range costReport.Optimizations {
			summary.PotentialSavings += opt.PotentialSavings
		}
	}

	return summary
}

// getResourcesCached retrieves resources with caching
func (poao *PerformanceOptimizedAnalysisOrchestrator) getResourcesCached(ctx context.Context) ([]core.Resource, error) {
	cacheKey := "resources_all"

	// Check cache first
	poao.cacheMutex.RLock()
	if cached, exists := poao.cache[cacheKey]; exists {
		if resources, ok := cached.([]core.Resource); ok {
			poao.cacheMutex.RUnlock()
			return resources, nil
		}
	}
	poao.cacheMutex.RUnlock()

	// Get from storage
	resources, err := poao.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, err
	}

	// Cache the result
	poao.cacheMutex.Lock()
	poao.cache[cacheKey] = resources
	poao.cacheMutex.Unlock()

	return resources, nil
}

// ClearCache clears the analysis cache
func (poao *PerformanceOptimizedAnalysisOrchestrator) ClearCache() {
	poao.cacheMutex.Lock()
	defer poao.cacheMutex.Unlock()

	poao.cache = make(map[string]interface{})
}

// GetCacheStats returns cache statistics
func (poao *PerformanceOptimizedAnalysisOrchestrator) GetCacheStats() map[string]interface{} {
	poao.cacheMutex.RLock()
	defer poao.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cache_size":       len(poao.cache),
		"max_workers":      poao.config.MaxWorkers,
		"batch_size":       poao.config.BatchSize,
		"parallel_enabled": poao.config.EnableParallel,
	}
}

// GetPerformanceMetrics returns performance metrics
func (poao *PerformanceOptimizedAnalysisOrchestrator) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"max_workers":      poao.config.MaxWorkers,
		"batch_size":       poao.config.BatchSize,
		"parallel_enabled": poao.config.EnableParallel,
		"memory_optimize":  poao.config.MemoryOptimize,
		"cache_timeout":    poao.config.CacheTimeout,
	}
}
