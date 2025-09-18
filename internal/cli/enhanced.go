package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/analysis"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/cloudrecon/cloudrecon/internal/export"
)

// EnhancedCLI provides an interactive and enhanced CLI experience
type EnhancedCLI struct {
	storage core.Storage
	scanner *bufio.Scanner
}

// NewEnhancedCLI creates a new enhanced CLI instance
func NewEnhancedCLI(storage core.Storage) *EnhancedCLI {
	return &EnhancedCLI{
		storage: storage,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// InteractiveAnalysisMode provides an interactive analysis experience
func (e *EnhancedCLI) InteractiveAnalysisMode() error {
	fmt.Println("CloudRecon Interactive Analysis Mode")
	fmt.Println("=====================================")
	fmt.Println()

	for {
		fmt.Println("Select an analysis option:")
		fmt.Println("1. Quick Analysis (Dependencies + Security + Cost)")
		fmt.Println("2. Dependency Analysis Only")
		fmt.Println("3. Security Analysis Only")
		fmt.Println("4. Cost Analysis Only")
		fmt.Println("5. Custom Analysis")
		fmt.Println("6. Export Results")
		fmt.Println("7. View Cache Statistics")
		fmt.Println("8. Exit")
		fmt.Println()

		choice := e.getUserInput("Enter your choice (1-8): ")

		switch choice {
		case "1":
			err := e.runQuickAnalysis()
			if err != nil {
				fmt.Printf("Error: Error: %v\n", err)
			}
		case "2":
			err := e.runDependencyAnalysis()
			if err != nil {
				fmt.Printf("Error: Error: %v\n", err)
			}
		case "3":
			err := e.runSecurityAnalysis()
			if err != nil {
				fmt.Printf("Error: Error: %v\n", err)
			}
		case "4":
			err := e.runCostAnalysis()
			if err != nil {
				fmt.Printf("Error: Error: %v\n", err)
			}
		case "5":
			err := e.runCustomAnalysis()
			if err != nil {
				fmt.Printf("Error: Error: %v\n", err)
			}
		case "6":
			err := e.exportResults()
			if err != nil {
				fmt.Printf("Error: Error: %v\n", err)
			}
		case "7":
			_ = e.showCacheStatistics()
		case "8":
			fmt.Println("Goodbye!")
			return nil
		default:
			fmt.Println("Error: Invalid choice. Please try again.")
		}

		fmt.Println()
		fmt.Println("Press Enter to continue...")
		e.scanner.Scan()
		fmt.Println()
	}
}

// runQuickAnalysis runs a comprehensive analysis with progress indicators
func (e *EnhancedCLI) runQuickAnalysis() error {
	fmt.Println("Starting Quick Analysis...")
	fmt.Println()

	// Create performance-optimized orchestrator
	config := analysis.DefaultPerformanceConfig()
	orchestrator := analysis.NewPerformanceOptimizedAnalysisOrchestrator(e.storage, config)

	// Show progress
	fmt.Println(" Analyzing dependencies...")
	start := time.Now()

	report, err := orchestrator.AnalyzeAllOptimized(nil)
	if err != nil {
		return err
	}

	duration := time.Since(start)

	// Display results
	e.displayAnalysisResults(report, duration)

	return nil
}

// runDependencyAnalysis runs dependency analysis only
func (e *EnhancedCLI) runDependencyAnalysis() error {
	fmt.Println(" Starting Dependency Analysis...")
	fmt.Println()

	config := analysis.DefaultPerformanceConfig()
	analyzer := analysis.NewPerformanceOptimizedDependencyAnalyzer(e.storage, config)

	start := time.Now()
	graph, err := analyzer.AnalyzeDependenciesOptimized(nil)
	if err != nil {
		return err
	}
	duration := time.Since(start)

	e.displayDependencyResults(graph, duration)
	return nil
}

// runSecurityAnalysis runs security analysis only
func (e *EnhancedCLI) runSecurityAnalysis() error {
	fmt.Println(" Starting Security Analysis...")
	fmt.Println()

	config := analysis.DefaultPerformanceConfig()
	analyzer := analysis.NewPerformanceOptimizedSecurityAnalyzer(e.storage, config)

	start := time.Now()
	report, err := analyzer.AnalyzeSecurityOptimized(nil)
	if err != nil {
		return err
	}
	duration := time.Since(start)

	e.displaySecurityResults(report, duration)
	return nil
}

// runCostAnalysis runs cost analysis only
func (e *EnhancedCLI) runCostAnalysis() error {
	fmt.Println(" Starting Cost Analysis...")
	fmt.Println()

	config := analysis.DefaultPerformanceConfig()
	analyzer := analysis.NewPerformanceOptimizedCostAnalyzer(e.storage, config)

	start := time.Now()
	report, err := analyzer.AnalyzeCostOptimized(nil)
	if err != nil {
		return err
	}
	duration := time.Since(start)

	e.displayCostResults(report, duration)
	return nil
}

// runCustomAnalysis allows users to customize analysis parameters
func (e *EnhancedCLI) runCustomAnalysis() error {
	fmt.Println("  Custom Analysis Configuration")
	fmt.Println()

	// Get analysis preferences
	includeDeps := e.getYesNoInput("Include dependency analysis? (y/n): ")
	includeSecurity := e.getYesNoInput("Include security analysis? (y/n): ")
	includeCost := e.getYesNoInput("Include cost analysis? (y/n): ")

	if !includeDeps && !includeSecurity && !includeCost {
		fmt.Println("Error: At least one analysis type must be selected.")
		return nil
	}

	// Get performance settings
	fmt.Println("\nPerformance Settings:")
	maxWorkers := e.getIntInput("Max workers (default 4): ", 4)
	batchSize := e.getIntInput("Batch size (default 100): ", 100)

	config := &analysis.PerformanceConfig{
		MaxWorkers:      maxWorkers,
		BatchSize:       batchSize,
		CacheTimeout:    5 * time.Minute,
		EnableParallel:  true,
		MemoryOptimize:  true,
		EnableProfiling: false,
	}

	fmt.Println("\n Starting Custom Analysis...")
	fmt.Println()

	start := time.Now()

	if includeDeps && includeSecurity && includeCost {
		// Run comprehensive analysis
		orchestrator := analysis.NewPerformanceOptimizedAnalysisOrchestrator(e.storage, config)
		report, err := orchestrator.AnalyzeAllOptimized(nil)
		if err != nil {
			return err
		}
		duration := time.Since(start)
		e.displayAnalysisResults(report, duration)
	} else {
		// Run individual analyses
		if includeDeps {
			fmt.Println(" Analyzing dependencies...")
			analyzer := analysis.NewPerformanceOptimizedDependencyAnalyzer(e.storage, config)
			graph, err := analyzer.AnalyzeDependenciesOptimized(nil)
			if err != nil {
				return err
			}
			e.displayDependencyResults(graph, time.Since(start))
		}

		if includeSecurity {
			fmt.Println(" Analyzing security...")
			analyzer := analysis.NewPerformanceOptimizedSecurityAnalyzer(e.storage, config)
			report, err := analyzer.AnalyzeSecurityOptimized(nil)
			if err != nil {
				return err
			}
			e.displaySecurityResults(report, time.Since(start))
		}

		if includeCost {
			fmt.Println(" Analyzing costs...")
			analyzer := analysis.NewPerformanceOptimizedCostAnalyzer(e.storage, config)
			report, err := analyzer.AnalyzeCostOptimized(nil)
			if err != nil {
				return err
			}
			e.displayCostResults(report, time.Since(start))
		}
	}

	return nil
}

// exportResults allows users to export analysis results
func (e *EnhancedCLI) exportResults() error {
	fmt.Println("📤 Export Results")
	fmt.Println()

	fmt.Println("Select export format:")
	fmt.Println("1. JSON")
	fmt.Println("2. CSV")
	fmt.Println("3. YAML")
	fmt.Println("4. HTML Report")
	fmt.Println("5. PDF Report")
	fmt.Println()

	choice := e.getUserInput("Enter your choice (1-5): ")

	var format string
	switch choice {
	case "1":
		format = "json"
	case "2":
		format = "csv"
	case "3":
		format = "yaml"
	case "4":
		format = "html"
	case "5":
		format = "pdf"
	default:
		fmt.Println("Error: Invalid choice.")
		return nil
	}

	filename := e.getUserInput("Enter filename (without extension): ")
	if filename == "" {
		filename = fmt.Sprintf("cloudrecon-analysis-%d", time.Now().Unix())
	}

	// Run analysis and export
	config := analysis.DefaultPerformanceConfig()
	orchestrator := analysis.NewPerformanceOptimizedAnalysisOrchestrator(e.storage, config)

	fmt.Println(" Running analysis for export...")
	report, err := orchestrator.AnalyzeAllOptimized(nil)
	if err != nil {
		return err
	}

	fmt.Printf("📤 Exporting to %s format...\n", strings.ToUpper(format))

	// Use enhanced exporter
	exporter := export.NewEnhancedExporter("./exports")

	if format == "html" || format == "pdf" {
		err = exporter.ExportCustomFormat(report, format, filename+"."+format)
		if err != nil {
			return fmt.Errorf("failed to export %s: %w", format, err)
		}
	} else {
		// For other formats, we'll create a simple JSON export
		// In a real implementation, you would convert the report to the appropriate format
		fmt.Printf("  %s format export not yet implemented for analysis reports\n", strings.ToUpper(format))
		fmt.Println("   Falling back to JSON format...")

		// Export as JSON
		exporter := export.NewEnhancedExporter("./exports")
		err = exporter.ExportCustomFormat(report, "json", filename+".json")
		if err != nil {
			return fmt.Errorf("failed to export JSON: %w", err)
		}
	}

	fmt.Printf(" Export completed: %s.%s\n", filename, format)

	return nil
}

// showCacheStatistics displays cache statistics
func (e *EnhancedCLI) showCacheStatistics() error {
	fmt.Println(" Cache Statistics")
	fmt.Println()

	config := analysis.DefaultPerformanceConfig()

	// Get statistics from all analyzers
	dependencyAnalyzer := analysis.NewPerformanceOptimizedDependencyAnalyzer(e.storage, config)
	securityAnalyzer := analysis.NewPerformanceOptimizedSecurityAnalyzer(e.storage, config)
	costAnalyzer := analysis.NewPerformanceOptimizedCostAnalyzer(e.storage, config)
	orchestrator := analysis.NewPerformanceOptimizedAnalysisOrchestrator(e.storage, config)

	fmt.Println("Dependency Analyzer Cache:")
	e.displayCacheStats(dependencyAnalyzer.GetCacheStats())

	fmt.Println("\nSecurity Analyzer Cache:")
	e.displayCacheStats(securityAnalyzer.GetCacheStats())

	fmt.Println("\nCost Analyzer Cache:")
	e.displayCacheStats(costAnalyzer.GetCacheStats())

	fmt.Println("\nOrchestrator Cache:")
	e.displayCacheStats(orchestrator.GetCacheStats())

	fmt.Println("\nPerformance Metrics:")
	e.displayPerformanceMetrics(orchestrator.GetPerformanceMetrics())

	return nil
}

// displayAnalysisResults shows comprehensive analysis results
func (e *EnhancedCLI) displayAnalysisResults(report *analysis.ComprehensiveAnalysisReport, duration time.Duration) {
	fmt.Println(" Analysis Results")
	fmt.Println("==================")
	fmt.Printf("  Analysis completed in %v\n", duration)
	fmt.Printf("📦 Total Resources: %d\n", report.Summary.TotalResources)
	fmt.Printf(" Total Dependencies: %d\n", report.Summary.TotalDependencies)
	fmt.Printf(" Security Findings: %d\n", report.Summary.SecurityFindings)
	fmt.Printf(" Total Monthly Cost: $%.2f\n", report.Summary.TotalMonthlyCost)
	fmt.Printf(" Compliance Score: %.1f%%\n", report.Summary.ComplianceScore)
	fmt.Printf("  Risk Score: %.1f\n", report.Summary.RiskScore)
	fmt.Println()

	// Show key findings
	if report.Security != nil && len(report.Security.Findings) > 0 {
		fmt.Println(" Security Findings:")
		for i, finding := range report.Security.Findings {
			if i >= 5 { // Show only first 5 findings
				fmt.Printf("   ... and %d more findings\n", len(report.Security.Findings)-5)
				break
			}
			fmt.Printf("   • %s (%s): %s\n", finding.Title, finding.Severity, finding.Description)
		}
		fmt.Println()
	}

	// Show cost insights
	if report.Cost != nil && len(report.Cost.Optimizations) > 0 {
		fmt.Println(" Cost Optimizations:")
		for i, opt := range report.Cost.Optimizations {
			if i >= 3 { // Show only first 3 optimizations
				fmt.Printf("   ... and %d more optimizations\n", len(report.Cost.Optimizations)-3)
				break
			}
			fmt.Printf("   • %s: Save $%.2f/month\n", opt.Title, opt.PotentialSavings)
		}
		fmt.Println()
	}
}

// displayDependencyResults shows dependency analysis results
func (e *EnhancedCLI) displayDependencyResults(graph *analysis.DependencyGraph, duration time.Duration) {
	fmt.Println(" Dependency Analysis Results")
	fmt.Println("=============================")
	fmt.Printf("  Analysis completed in %v\n", duration)
	fmt.Printf("📦 Total Resources: %d\n", graph.Stats.TotalResources)
	fmt.Printf(" Total Dependencies: %d\n", graph.Stats.TotalDependencies)
	fmt.Printf("🔄 Cycles: %d\n", graph.Stats.Cycles)
	fmt.Printf("  Islands: %d\n", graph.Stats.Islands)
	fmt.Printf("📏 Max Depth: %d\n", graph.Stats.MaxDepth)
	fmt.Println()
}

// displaySecurityResults shows security analysis results
func (e *EnhancedCLI) displaySecurityResults(report *analysis.SecurityReport, duration time.Duration) {
	fmt.Println(" Security Analysis Results")
	fmt.Println("===========================")
	fmt.Printf("  Analysis completed in %v\n", duration)
	fmt.Printf(" Total Findings: %d\n", report.Summary.TotalFindings)
	fmt.Printf("🔴 Critical: %d\n", report.Summary.CriticalFindings)
	fmt.Printf("🟠 High: %d\n", report.Summary.HighFindings)
	fmt.Printf("🟡 Medium: %d\n", report.Summary.MediumFindings)
	fmt.Printf("🟢 Low: %d\n", report.Summary.LowFindings)
	fmt.Printf(" Compliance Score: %.1f%%\n", report.ComplianceScore)
	fmt.Printf("  Risk Score: %.1f\n", report.RiskScore)
	fmt.Println()

	// Show top findings
	if len(report.Findings) > 0 {
		fmt.Println(" Top Security Findings:")
		for i, finding := range report.Findings {
			if i >= 5 { // Show only first 5 findings
				fmt.Printf("   ... and %d more findings\n", len(report.Findings)-5)
				break
			}
			severity := strings.ToUpper(finding.Severity)
			fmt.Printf("   • [%s] %s: %s\n", severity, finding.Title, finding.Description)
		}
		fmt.Println()
	}
}

// displayCostResults shows cost analysis results
func (e *EnhancedCLI) displayCostResults(report *analysis.CostReport, duration time.Duration) {
	fmt.Println(" Cost Analysis Results")
	fmt.Println("=======================")
	fmt.Printf("  Analysis completed in %v\n", duration)
	fmt.Printf("📦 Total Resources: %d\n", report.Summary.TotalResources)
	fmt.Printf(" Resources with Cost: %d\n", report.Summary.ResourcesWithCost)
	fmt.Printf("💵 Average Monthly Cost: $%.2f\n", report.Summary.AverageMonthlyCost)
	fmt.Printf("💸 Total Potential Savings: $%.2f\n", report.PotentialSavings)
	fmt.Println()

	// Show cost by provider
	if len(report.Summary.CostByProvider) > 0 {
		fmt.Println(" Cost by Provider:")
		for provider, cost := range report.Summary.CostByProvider {
			fmt.Printf("   • %s: $%.2f\n", strings.ToUpper(provider), cost)
		}
		fmt.Println()
	}

	// Show top optimizations
	if len(report.Optimizations) > 0 {
		fmt.Println("💡 Top Cost Optimizations:")
		for i, opt := range report.Optimizations {
			if i >= 3 { // Show only first 3 optimizations
				fmt.Printf("   ... and %d more optimizations\n", len(report.Optimizations)-3)
				break
			}
			fmt.Printf("   • %s: Save $%.2f/month (%.1f%%)\n", opt.Title, opt.PotentialSavings, opt.SavingsPercent)
		}
		fmt.Println()
	}
}

// displayCacheStats shows cache statistics
func (e *EnhancedCLI) displayCacheStats(stats map[string]interface{}) {
	fmt.Printf("   Cache Size: %v\n", stats["cache_size"])
	fmt.Printf("   Max Workers: %v\n", stats["max_workers"])
	fmt.Printf("   Batch Size: %v\n", stats["batch_size"])
	fmt.Printf("   Parallel Enabled: %v\n", stats["parallel_enabled"])
}

// displayPerformanceMetrics shows performance metrics
func (e *EnhancedCLI) displayPerformanceMetrics(metrics map[string]interface{}) {
	fmt.Printf("   Max Workers: %v\n", metrics["max_workers"])
	fmt.Printf("   Batch Size: %v\n", metrics["batch_size"])
	fmt.Printf("   Parallel Enabled: %v\n", metrics["parallel_enabled"])
	fmt.Printf("   Memory Optimize: %v\n", metrics["memory_optimize"])
	fmt.Printf("   Cache Timeout: %v\n", metrics["cache_timeout"])
}

// Helper methods for user input
func (e *EnhancedCLI) getUserInput(prompt string) string {
	fmt.Print(prompt)
	e.scanner.Scan()
	return strings.TrimSpace(e.scanner.Text())
}

func (e *EnhancedCLI) getYesNoInput(prompt string) bool {
	for {
		response := strings.ToLower(e.getUserInput(prompt))
		if response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" {
			return false
		}
		fmt.Println("Please enter 'y' or 'n'")
	}
}

func (e *EnhancedCLI) getIntInput(prompt string, defaultValue int) int {
	for {
		response := e.getUserInput(prompt)
		if response == "" {
			return defaultValue
		}
		if value, err := strconv.Atoi(response); err == nil && value > 0 {
			return value
		}
		fmt.Println("Please enter a valid positive integer")
	}
}
