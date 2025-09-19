package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/analysis"
	"github.com/cloudrecon/cloudrecon/internal/cli"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/cloudrecon/cloudrecon/internal/export"
	"github.com/cloudrecon/cloudrecon/internal/providers/aws"
	"github.com/cloudrecon/cloudrecon/internal/providers/azure"
	"github.com/cloudrecon/cloudrecon/internal/providers/gcp"
	"github.com/cloudrecon/cloudrecon/internal/query"
	"github.com/cloudrecon/cloudrecon/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version    = "1.0.0"
	verbose    bool
	configFile string
)

func main() {
	// Set up logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Create root command
	var rootCmd = &cobra.Command{
		Use:   "cloudrecon",
		Short: "Unified Multi-Cloud Discovery CLI",
		Long: `CloudRecon discovers and inventories cloud infrastructure across AWS, Azure, and GCP
in a single command. It provides instant visibility into all cloud resources with
zero infrastructure requirements.`,
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				logrus.SetLevel(logrus.DebugLevel)
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config/cloudrecon.yaml", "Configuration file path")

	// Add subcommands
	rootCmd.AddCommand(createDiscoverCmd())
	rootCmd.AddCommand(createQueryCmd())
	rootCmd.AddCommand(createExportCmd())
	rootCmd.AddCommand(createAskCmd())
	rootCmd.AddCommand(createStatusCmd())
	rootCmd.AddCommand(createAnalyzeCmd())
	rootCmd.AddCommand(createSecurityCmd())
	rootCmd.AddCommand(createCostCmd())
	rootCmd.AddCommand(createDependenciesCmd())
	rootCmd.AddCommand(createInteractiveCmd())

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logrus.Info("Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Execute command
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logrus.Fatal(err)
	}
}

func createDiscoverCmd() *cobra.Command {
	var (
		providers      []string
		accounts       []string
		regions        []string
		resourceTypes  []string
		mode           string
		useNativeTools bool
		maxParallel    int
		timeout        time.Duration
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover cloud resources across all providers",
		Long: `Discover and inventory cloud resources across AWS, Azure, and GCP.
This command will automatically detect available accounts and discover resources
using the most efficient method available (cloud-native tools when possible).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create context manager with proper timeouts
			ctxManager := core.NewContextManager(cmd.Context())
			ctxManager.SetTimeout(core.OperationDiscovery, timeout)
			ctxManager.SetTimeout(core.OperationStorage, 10*time.Second)

			// Load configuration
			config, err := loadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Initialize storage with context
			storageCtx := ctxManager.NewOperationContext(core.OperationStorage)
			defer storageCtx.Cancel()
			
			storage, err := storage.NewSQLiteStorage(config.Storage.DatabasePath)
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Initialize providers based on command line flags
			providerMap := make(map[string]core.CloudProvider)

			// If no providers specified, initialize all
			if len(providers) == 0 {
				providers = []string{"aws", "azure", "gcp"}
			}

			if contains(providers, "aws") {
				awsProvider, err := aws.NewProvider(config.AWS)
				if err != nil {
					logrus.Warnf("Failed to initialize AWS provider: %v", err)
				} else {
					providerMap["aws"] = awsProvider
				}
			}

			if contains(providers, "azure") {
				azureProvider, err := azure.NewProvider(config.Azure)
				if err != nil {
					logrus.Warnf("Failed to initialize Azure provider: %v", err)
				} else {
					providerMap["azure"] = azureProvider
				}
			}

			if contains(providers, "gcp") {
				gcpProvider, err := gcp.NewProvider(config.GCP)
				if err != nil {
					logrus.Warnf("Failed to initialize GCP provider: %v", err)
				} else {
					providerMap["gcp"] = gcpProvider
				}
			}

			if len(providerMap) == 0 {
				return fmt.Errorf("no cloud providers could be initialized")
			}

			// Parse discovery mode
			discoveryMode := core.StandardMode
			switch mode {
			case "quick":
				discoveryMode = core.QuickMode
			case "standard":
				discoveryMode = core.StandardMode
			case "deep":
				discoveryMode = core.DeepMode
			}

			// Create discovery options
			options := core.DiscoveryOptions{
				Mode:           discoveryMode,
				Providers:      providers, // Use the actual providers from command line
				Accounts:       accounts,
				Regions:        regions,
				ResourceTypes:  resourceTypes,
				UseNativeTools: useNativeTools,
				MaxParallel:    maxParallel,
				Timeout:        timeout,
				ProgressHandler: func(progress core.DiscoveryProgress) {
					logrus.Infof("Discovery progress: %d resources found, %d accounts processed",
						progress.ResourcesFound, progress.AccountsProcessed)
				},
			}

			// Create orchestrator
			orchestrator := core.NewDiscoveryOrchestrator(providerMap, storage, options)

			// Start discovery with proper context
			logrus.Info("Starting cloud resource discovery...")
			start := time.Now()

			discoveryCtx := ctxManager.NewOperationContext(core.OperationDiscovery)
			defer discoveryCtx.Cancel()
			
			result, err := orchestrator.Discover(discoveryCtx.Context())
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			duration := time.Since(start)
			logrus.Infof("Discovery completed in %v", duration)
			logrus.Infof("Found %d resources across %d accounts",
				len(result.Resources), len(result.Accounts))

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&providers, "providers", "p", []string{}, "Cloud providers to scan (aws, azure, gcp)")
	cmd.Flags().StringSliceVarP(&accounts, "accounts", "a", []string{}, "Specific accounts to scan")
	cmd.Flags().StringSliceVarP(&regions, "regions", "r", []string{}, "Specific regions to scan")
	cmd.Flags().StringSliceVarP(&resourceTypes, "resource-types", "t", []string{}, "Specific resource types to discover")
	cmd.Flags().StringVarP(&mode, "mode", "m", "standard", "Discovery mode (quick, standard, deep)")
	cmd.Flags().BoolVar(&useNativeTools, "native-tools", true, "Use cloud-native tools when available")
	cmd.Flags().IntVar(&maxParallel, "max-parallel", 10, "Maximum parallel operations")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "Discovery timeout")

	return cmd
}

func createQueryCmd() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query discovered resources",
		Long:  "Query and filter discovered cloud resources using SQL or natural language",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create query engine
			engine := query.NewEngine(storage)

			// Execute query
			queryStr := strings.Join(args, " ")
			if queryStr == "" {
				return fmt.Errorf("query string is required")
			}

			results, err := engine.ExecuteSQL(queryStr)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			// Output results
			if output != "" {
				exporter := export.NewExporter()
				return exporter.Export(results, format, output)
			}

			// Print results
			for _, resource := range results {
				fmt.Printf("%s: %s (%s)\n", resource.Type, resource.Name, resource.Provider)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, csv)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	return cmd
}

func createExportCmd() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export discovered resources",
		Long:  "Export discovered cloud resources to various formats",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Get all resources
			resources, err := storage.GetResources("SELECT * FROM resources", []interface{}{})
			if err != nil {
				return fmt.Errorf("failed to get resources: %w", err)
			}

			// Export resources
			exporter := export.NewExporter()
			return exporter.Export(resources, format, output)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "json", "Export format (json, csv, yaml)")
	cmd.Flags().StringVarP(&output, "output", "o", "resources.json", "Output file path")

	return cmd
}

func createAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask questions about your cloud infrastructure",
		Long:  "Ask natural language questions about your discovered cloud resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create query engine
			engine := query.NewEngine(storage)

			// Process natural language query
			queryStr := strings.Join(args, " ")
			if queryStr == "" {
				return fmt.Errorf("question is required")
			}

			results, err := engine.ExecuteSQL(queryStr)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			// Print results
			fmt.Printf("Found %d resources matching your query:\n", len(results))
			for _, resource := range results {
				fmt.Printf("- %s: %s (%s)\n", resource.Type, resource.Name, resource.Provider)
			}

			return nil
		},
	}

	return cmd
}

func createStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show discovery status and statistics",
		Long:  "Display the current status of cloud resource discovery and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Get discovery status
			status, err := storage.GetDiscoveryStatus()
			if err != nil {
				return fmt.Errorf("failed to get discovery status: %w", err)
			}

			// Print status
			fmt.Printf("Discovery Status: %s\n", status.Status)
			fmt.Printf("Last Discovery: %s\n", status.LastRun.Format(time.RFC3339))
			fmt.Printf("Total Resources: %d\n", status.ResourceCount)
			fmt.Printf("Providers: %s\n", status.Providers)

			return nil
		},
	}

	return cmd
}

func createAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run comprehensive analysis on discovered resources",
		Long:  "Perform dependency mapping, security analysis, and cost estimation",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create analysis orchestrator
			orchestrator := analysis.NewAnalysisOrchestrator(storage)

			// Run comprehensive analysis
			report, err := orchestrator.AnalyzeAll(context.TODO())
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Print analysis results
			fmt.Printf("Analysis completed!\n")
			fmt.Printf("Total Resources: %d\n", report.Summary.TotalResources)
			fmt.Printf("Dependencies: %d\n", report.Summary.TotalDependencies)
			fmt.Printf("Security Findings: %d\n", report.Summary.SecurityFindings)
			fmt.Printf("Monthly Cost: $%.2f\n", report.Summary.TotalMonthlyCost)

			return nil
		},
	}

	return cmd
}

func createSecurityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Run security analysis on discovered resources",
		Long:  "Analyze security posture and identify potential vulnerabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create security analyzer
			analyzer := analysis.NewSecurityAnalyzer(storage)

			// Run security analysis
			report, err := analyzer.AnalyzeSecurity(context.TODO())
			if err != nil {
				return fmt.Errorf("security analysis failed: %w", err)
			}

			// Print security results
			fmt.Printf("Security Analysis completed!\n")
			fmt.Printf("Total Findings: %d\n", report.Summary.TotalFindings)
			fmt.Printf("Critical: %d\n", report.Summary.CriticalFindings)
			fmt.Printf("High: %d\n", report.Summary.HighFindings)
			fmt.Printf("Medium: %d\n", report.Summary.MediumFindings)
			fmt.Printf("Low: %d\n", report.Summary.LowFindings)

			return nil
		},
	}

	return cmd
}

func createCostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Run cost analysis on discovered resources",
		Long:  "Estimate costs and identify optimization opportunities",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create cost analyzer
			analyzer := analysis.NewCostAnalyzer(storage)

			// Run cost analysis
			report, err := analyzer.AnalyzeCost(context.TODO())
			if err != nil {
				return fmt.Errorf("cost analysis failed: %w", err)
			}

			// Print cost results
			fmt.Printf("Cost Analysis completed!\n")
			fmt.Printf("Total Resources: %d\n", report.Summary.TotalResources)
			fmt.Printf("Resources with Cost: %d\n", report.Summary.ResourcesWithCost)
			fmt.Printf("Average Monthly Cost: $%.2f\n", report.Summary.AverageMonthlyCost)
			fmt.Printf("Potential Savings: $%.2f\n", report.PotentialSavings)

			return nil
		},
	}

	return cmd
}

func createDependenciesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Run dependency analysis on discovered resources",
		Long:  "Map resource dependencies and relationships",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create dependency analyzer
			analyzer := analysis.NewDependencyAnalyzer(storage)

			// Run dependency analysis
			graph, err := analyzer.AnalyzeDependencies(context.TODO())
			if err != nil {
				return fmt.Errorf("dependency analysis failed: %w", err)
			}

			// Print dependency results
			fmt.Printf("Dependency Analysis completed!\n")
			fmt.Printf("Total Resources: %d\n", graph.Stats.TotalResources)
			fmt.Printf("Total Dependencies: %d\n", graph.Stats.TotalDependencies)
			fmt.Printf("Cycles: %d\n", graph.Stats.Cycles)
			fmt.Printf("Islands: %d\n", graph.Stats.Islands)

			return nil
		},
	}

	return cmd
}

// createInteractiveCmd creates the interactive CLI command
func createInteractiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive analysis mode",
		Long:  "Launch an interactive CLI for comprehensive cloud analysis with guided workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize storage
			storage, err := storage.NewSQLiteStorage(viper.GetString("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer storage.Close()

			// Create enhanced CLI
			enhancedCLI := cli.NewEnhancedCLI(storage)

			// Start interactive mode
			return enhancedCLI.InteractiveAnalysisMode()
		},
	}
}

// Helper functions
func loadConfig() (*core.Config, error) {
	// Initialize viper
	viper.SetConfigName("cloudrecon")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.cloudrecon")
	viper.AddConfigPath("/etc/cloudrecon")

	// Set default values
	setDefaultConfig()

	// Enable environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CLOUDRECON")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Warn("Config file not found, using defaults")
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		logrus.Infof("Using config file: %s", viper.ConfigFileUsed())
	}

	// Unmarshal config
	var config core.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Debug logging
	logrus.Debugf("Loaded config successfully")

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func setDefaultConfig() {
	// Storage defaults
	viper.SetDefault("storage.database_path", "cloudrecon.db")
	viper.SetDefault("storage.max_connections", 10)
	viper.SetDefault("storage.connection_timeout", "30s")

	// AWS defaults
	viper.SetDefault("aws.regions", []string{"us-east-1", "us-west-2"})
	viper.SetDefault("aws.max_retries", 3)
	viper.SetDefault("aws.timeout", "30s")

	// Azure defaults
	viper.SetDefault("azure.subscriptions", []string{})
	viper.SetDefault("azure.max_retries", 3)
	viper.SetDefault("azure.timeout", "30s")

	// GCP defaults
	viper.SetDefault("gcp.project_id", "")
	viper.SetDefault("gcp.organization_id", "")
	viper.SetDefault("gcp.credentials_path", "")
	viper.SetDefault("gcp.projects", []string{})
	viper.SetDefault("gcp.max_retries", 3)
	viper.SetDefault("gcp.timeout", "30s")
	viper.SetDefault("gcp.discovery_methods", []string{"config", "environment", "gcloud", "metadata", "resource_manager"})

	// Discovery defaults
	viper.SetDefault("discovery.max_parallel", 10)
	viper.SetDefault("discovery.timeout", "300s")
	viper.SetDefault("discovery.use_native_tools", true)

	// Analysis defaults
	viper.SetDefault("analysis.enable_cost_analysis", true)
	viper.SetDefault("analysis.enable_security_analysis", true)
	viper.SetDefault("analysis.enable_dependency_analysis", true)
	viper.SetDefault("analysis.cache_results", true)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
}

func validateConfig(config *core.Config) error {
	// Validate storage config
	if config.Storage.DatabasePath == "" {
		return fmt.Errorf("storage.database_path is required")
	}

	// Validate AWS config
	if len(config.AWS.Regions) == 0 {
		return fmt.Errorf("aws.regions cannot be empty")
	}

	// Validate GCP config
	if config.GCP.ProjectID != "" && len(config.GCP.Projects) > 0 {
		return fmt.Errorf("cannot specify both gcp.project_id and gcp.projects")
	}

	// Validate discovery config
	if config.Discovery.MaxParallel <= 0 {
		return fmt.Errorf("discovery.max_parallel must be greater than 0")
	}

	// Validate analysis config
	if !config.Analysis.EnableCostAnalysis && !config.Analysis.EnableSecurityAnalysis && !config.Analysis.EnableDependencyAnalysis {
		return fmt.Errorf("at least one analysis type must be enabled")
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
