package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// AnalysisOrchestrator coordinates all analysis operations
type AnalysisOrchestrator struct {
	storage            core.Storage
	dependencyAnalyzer *DependencyAnalyzer
	securityAnalyzer   *SecurityAnalyzer
	costAnalyzer       *CostAnalyzer
}

// NewAnalysisOrchestrator creates a new analysis orchestrator
func NewAnalysisOrchestrator(storage core.Storage) *AnalysisOrchestrator {
	return &AnalysisOrchestrator{
		storage:            storage,
		dependencyAnalyzer: NewDependencyAnalyzer(storage),
		securityAnalyzer:   NewSecurityAnalyzer(storage),
		costAnalyzer:       NewCostAnalyzer(storage),
	}
}

// AnalysisReport represents a comprehensive analysis report
type AnalysisReport struct {
	Timestamp       time.Time        `json:"timestamp"`
	DependencyGraph *DependencyGraph `json:"dependency_graph"`
	SecurityReport  *SecurityReport  `json:"security_report"`
	CostReport      *CostReport      `json:"cost_report"`
	Summary         AnalysisSummary  `json:"summary"`
}

// AnalysisSummary provides a high-level summary of all analyses
type AnalysisSummary struct {
	TotalResources    int     `json:"total_resources"`
	TotalDependencies int     `json:"total_dependencies"`
	SecurityFindings  int     `json:"security_findings"`
	CriticalFindings  int     `json:"critical_findings"`
	TotalMonthlyCost  float64 `json:"total_monthly_cost"`
	PotentialSavings  float64 `json:"potential_savings"`
	ComplianceScore   float64 `json:"compliance_score"`
	RiskScore         float64 `json:"risk_score"`
	AnalysisDuration  string  `json:"analysis_duration"`
}

// AnalyzeAll performs comprehensive analysis across all dimensions
func (ao *AnalysisOrchestrator) AnalyzeAll(ctx context.Context) (*AnalysisReport, error) {
	startTime := time.Now()
	logrus.Info("Starting comprehensive analysis")

	// Run all analyses in parallel
	type analysisResult struct {
		name string
		data interface{}
		err  error
	}

	resultChan := make(chan analysisResult, 3)

	// Dependency analysis
	go func() {
		graph, err := ao.dependencyAnalyzer.AnalyzeDependencies(ctx)
		resultChan <- analysisResult{
			name: "dependency",
			data: graph,
			err:  err,
		}
	}()

	// Security analysis
	go func() {
		report, err := ao.securityAnalyzer.AnalyzeSecurity(ctx)
		resultChan <- analysisResult{
			name: "security",
			data: report,
			err:  err,
		}
	}()

	// Cost analysis
	go func() {
		report, err := ao.costAnalyzer.AnalyzeCost(ctx)
		resultChan <- analysisResult{
			name: "cost",
			data: report,
			err:  err,
		}
	}()

	// Collect results
	var dependencyGraph *DependencyGraph
	var securityReport *SecurityReport
	var costReport *CostReport

	for i := 0; i < 3; i++ {
		result := <-resultChan
		switch result.name {
		case "dependency":
			if result.err != nil {
				logrus.Errorf("Dependency analysis failed: %v", result.err)
			} else {
				dependencyGraph = result.data.(*DependencyGraph)
			}
		case "security":
			if result.err != nil {
				logrus.Errorf("Security analysis failed: %v", result.err)
			} else {
				securityReport = result.data.(*SecurityReport)
			}
		case "cost":
			if result.err != nil {
				logrus.Errorf("Cost analysis failed: %v", result.err)
			} else {
				costReport = result.data.(*CostReport)
			}
		}
	}

	// Calculate summary
	summary := ao.calculateAnalysisSummary(dependencyGraph, securityReport, costReport, time.Since(startTime))

	report := &AnalysisReport{
		Timestamp:       startTime,
		DependencyGraph: dependencyGraph,
		SecurityReport:  securityReport,
		CostReport:      costReport,
		Summary:         summary,
	}

	logrus.Infof("Comprehensive analysis completed in %v", time.Since(startTime))

	return report, nil
}

// AnalyzeDependencies performs dependency analysis only
func (ao *AnalysisOrchestrator) AnalyzeDependencies(ctx context.Context) (*DependencyGraph, error) {
	return ao.dependencyAnalyzer.AnalyzeDependencies(ctx)
}

// AnalyzeSecurity performs security analysis only
func (ao *AnalysisOrchestrator) AnalyzeSecurity(ctx context.Context) (*SecurityReport, error) {
	return ao.securityAnalyzer.AnalyzeSecurity(ctx)
}

// AnalyzeCost performs cost analysis only
func (ao *AnalysisOrchestrator) AnalyzeCost(ctx context.Context) (*CostReport, error) {
	return ao.costAnalyzer.AnalyzeCost(ctx)
}

// calculateAnalysisSummary calculates the overall analysis summary
func (ao *AnalysisOrchestrator) calculateAnalysisSummary(
	dependencyGraph *DependencyGraph,
	securityReport *SecurityReport,
	costReport *CostReport,
	duration time.Duration,
) AnalysisSummary {
	summary := AnalysisSummary{
		AnalysisDuration: duration.String(),
	}

	// Dependency statistics
	if dependencyGraph != nil {
		summary.TotalResources = dependencyGraph.Stats.TotalResources
		summary.TotalDependencies = dependencyGraph.Stats.TotalDependencies
	}

	// Security statistics
	if securityReport != nil {
		summary.SecurityFindings = securityReport.Summary.TotalFindings
		summary.CriticalFindings = securityReport.Summary.CriticalFindings
		summary.ComplianceScore = securityReport.ComplianceScore
		summary.RiskScore = securityReport.RiskScore
	}

	// Cost statistics
	if costReport != nil {
		summary.TotalMonthlyCost = costReport.TotalMonthlyCost
		summary.PotentialSavings = costReport.PotentialSavings
	}

	return summary
}

// GetAnalysisInsights provides high-level insights from the analysis
func (ao *AnalysisOrchestrator) GetAnalysisInsights(ctx context.Context) ([]string, error) {
	report, err := ao.AnalyzeAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to perform analysis: %w", err)
	}

	var insights []string

	// Resource insights
	if report.Summary.TotalResources > 0 {
		insights = append(insights, fmt.Sprintf("Discovered %d resources across your cloud infrastructure", report.Summary.TotalResources))
	}

	// Dependency insights
	if report.DependencyGraph != nil {
		if report.DependencyGraph.Stats.TotalDependencies > 0 {
			insights = append(insights, fmt.Sprintf("Found %d resource dependencies with %d potential cycles",
				report.DependencyGraph.Stats.TotalDependencies, report.DependencyGraph.Stats.Cycles))
		}
		if report.DependencyGraph.Stats.Islands > 0 {
			insights = append(insights, fmt.Sprintf("%d resources appear to be isolated with no dependencies",
				report.DependencyGraph.Stats.Islands))
		}
	}

	// Security insights
	if report.SecurityReport != nil {
		if report.SecurityReport.Summary.CriticalFindings > 0 {
			insights = append(insights, fmt.Sprintf("  %d critical security findings require immediate attention",
				report.SecurityReport.Summary.CriticalFindings))
		}
		if report.SecurityReport.Summary.TotalFindings > 0 {
			insights = append(insights, fmt.Sprintf("Security analysis found %d total issues with %.1f%% compliance score",
				report.SecurityReport.Summary.TotalFindings, report.SecurityReport.ComplianceScore))
		}
	}

	// Cost insights
	if report.CostReport != nil {
		insights = append(insights, fmt.Sprintf(" Total monthly cost: $%.2f with potential savings of $%.2f",
			report.CostReport.TotalMonthlyCost, report.CostReport.PotentialSavings))

		if len(report.CostReport.Optimizations) > 0 {
			highPriorityOpts := 0
			for _, opt := range report.CostReport.Optimizations {
				if opt.Priority == "high" {
					highPriorityOpts++
				}
			}
			if highPriorityOpts > 0 {
				insights = append(insights, fmt.Sprintf("💡 %d high-priority cost optimization opportunities identified", highPriorityOpts))
			}
		}
	}

	// Performance insights
	if report.Summary.AnalysisDuration != "" {
		insights = append(insights, fmt.Sprintf(" Analysis completed in %s", report.Summary.AnalysisDuration))
	}

	return insights, nil
}

// ExportAnalysisReport exports the analysis report in various formats
func (ao *AnalysisOrchestrator) ExportAnalysisReport(ctx context.Context, format string) ([]byte, error) {
	report, err := ao.AnalyzeAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to perform analysis: %w", err)
	}

	switch format {
	case "json":
		return ao.exportJSON(report)
	case "yaml":
		return ao.exportYAML(report)
	case "csv":
		return ao.exportCSV(report)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports the report as JSON
func (ao *AnalysisOrchestrator) exportJSON(report *AnalysisReport) ([]byte, error) {
	// This would use json.Marshal in practice
	// For now, return a placeholder
	return []byte(`{"message": "JSON export not implemented yet"}`), nil
}

// exportYAML exports the report as YAML
func (ao *AnalysisOrchestrator) exportYAML(report *AnalysisReport) ([]byte, error) {
	// This would use yaml.Marshal in practice
	// For now, return a placeholder
	return []byte(`message: "YAML export not implemented yet"`), nil
}

// exportCSV exports the report as CSV
func (ao *AnalysisOrchestrator) exportCSV(report *AnalysisReport) ([]byte, error) {
	// This would generate CSV data in practice
	// For now, return a placeholder
	return []byte(`message,CSV export not implemented yet`), nil
}
