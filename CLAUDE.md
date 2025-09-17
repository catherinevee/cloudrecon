# CloudRecon: Unified Multi-Cloud Discovery CLI - Technical Specification

## Executive Summary

CloudRecon is a Go-based CLI tool that discovers and inventories cloud infrastructure across AWS, Azure, and GCP in a single command. Unlike existing solutions that cost $50,000+ annually or require complex setup, CloudRecon provides instant visibility into all cloud resources with zero infrastructure requirements.

**Core Innovation**: Intelligently combines cloud-native discovery tools (AWS Systems Manager, Azure Resource Graph, GCP Cloud Asset Inventory) with direct API calls to provide the most comprehensive and efficient discovery possible.

## Problem Statement

### Current Pain Points
- **Invisible Resources**: 30% of cloud resources are "shadow IT" - unknown to central IT teams
- **Cost Hemorrhaging**: $61 billion wasted annually on unused/forgotten cloud resources
- **Security Blindspots**: 82% of breaches involve cloud assets teams didn't know existed
- **Tool Fragmentation**: Teams juggle 3-6 different tools per cloud provider
- **Manual Discovery**: Takes 40+ hours to manually inventory a medium-sized cloud estate

### Why Existing Solutions Fail

| Solution | Fatal Flaw | Result |
|----------|------------|--------|
| **Steampipe** | Live API queries only, no persistent storage | Too slow for large environments (30+ min for full scan) |
| **CloudQuery** | Requires PostgreSQL setup and maintenance | 66% of teams abandon during setup |
| **ServiceNow Discovery** | Enterprise complexity, $100k+ pricing | Inaccessible to 95% of organizations |
| **Cloud-native tools** | Single-cloud only, no aggregation | Requires 3 separate tools and manual correlation |
| **Manual scripts** | Maintenance burden, no error handling | Break with API changes, miss resources |

## Solution Architecture

### Core Design Principles

1. **Hybrid Approach**: Leverage cloud-native tools when superior, use direct APIs when necessary
2. **Zero Infrastructure**: Single binary, no databases, no agents, no servers
3. **Offline-First**: Cache everything locally for instant queries without API calls
4. **Progressive Enhancement**: Basic discovery in 30 seconds, deep discovery optional
5. **Cloud-Native Integration**: Use AWS Config, Azure Resource Graph, and GCP Asset Inventory when available

### Technical Architecture

```yaml
CloudRecon Architecture:
  Discovery Layer:
    Cloud-Native Tools:
      AWS:
        - AWS Config (when configured): Instant resource inventory
        - Systems Manager Inventory: EC2 detailed metadata
        - Organizations API: Account enumeration
        - Cost Explorer API: Cost allocation
      Azure:
        - Resource Graph: Lightning-fast cross-subscription queries
        - Azure Arc: Hybrid resource discovery
        - Cost Management API: Spending analysis
      GCP:
        - Cloud Asset Inventory API: Real-time resource state
        - Resource Manager API: Project hierarchy
        - Billing API: Cost breakdown
    
    Direct API Fallback:
      - When cloud tools unavailable
      - For resources not covered by native tools
      - For cross-region aggregation
      
  Storage Layer:
    Primary: Embedded SQLite database
    Cache: Local JSON files for raw responses
    Export: CSV, JSON, YAML, Terraform state
    
  Query Layer:
    SQL: Full SQLite syntax support
    DSL: Simple query language for common patterns
    Natural Language: LLM integration for plain English queries
```

### Smart Discovery Strategy

```go
// Intelligent discovery selection
type DiscoveryStrategy struct {
    // Use cloud-native tools when available
    PreferNativeTools bool
    
    // Fallback strategies
    UseDirectAPI     bool
    UseTerraformState bool
    UseCloudFormation bool
    
    // Performance optimization
    ParallelAccounts int
    RateLimitPerSec  int
}

// Example: AWS Discovery Logic
func (d *AWSDiscoverer) Discover(ctx context.Context) ([]Resource, error) {
    // 1. Try AWS Config first (fastest, most complete)
    if d.hasAWSConfig() {
        return d.discoverViaConfig(ctx)
    }
    
    // 2. Try Systems Manager Inventory for EC2
    if d.hasSystemsManager() {
        resources = append(resources, d.discoverViaSSM(ctx))
    }
    
    // 3. Fall back to direct API calls
    return d.discoverViaAPI(ctx)
}
```

## Feature Specifications

### 1. Intelligent Account Discovery

**AWS Implementation:**
```bash
# Automatically discover all AWS accounts
cloudrecon discover --auto-detect

# Uses in order:
# 1. AWS Organizations API (finds all member accounts)
# 2. AWS SSO/Identity Center (finds assumed role accounts)  
# 3. Config Aggregator (finds accounts sending data)
# 4. ~/.aws/config parsing (local profiles)
```

**Azure Implementation:**
```bash
# Discover all Azure subscriptions
cloudrecon discover --provider azure

# Uses:
# 1. Azure Resource Graph (instant cross-subscription query)
# 2. Management Groups API (hierarchy traversal)
# 3. Azure Arc (hybrid resources)
```

**GCP Implementation:**
```bash
# Discover all GCP projects
cloudrecon discover --provider gcp

# Uses:
# 1. Cloud Asset Inventory (if enabled)
# 2. Resource Manager API (project enumeration)
# 3. Folder/Organization traversal
```

### 2. Resource Discovery Priorities

```yaml
Critical Resources (Always Discovered):
  Security:
    - Public S3 buckets, storage containers
    - Open security groups/firewalls
    - Unencrypted databases
    - Public IPs without protection
    - IAM users with excessive permissions
    
  Cost:
    - Idle compute instances (0% CPU for 7+ days)
    - Unattached volumes/disks
    - Unused elastic IPs
    - Oversized databases
    - Forgotten snapshots
    
  Compliance:
    - Resources missing required tags
    - Resources in non-approved regions
    - Non-compliant configurations
    
Extended Discovery (Optional):
  - Lambda functions and configurations
  - API Gateway endpoints
  - Container registries
  - ML model endpoints
  - IoT devices
```

### 3. Query Interface

**Built-in Query Templates:**
```sql
-- Find all public resources
SELECT * FROM resources 
WHERE public_access = true;

-- Identify unused resources costing money
SELECT * FROM resources 
WHERE last_used < datetime('now', '-30 days') 
AND monthly_cost > 0;

-- Security audit query
SELECT * FROM resources
WHERE (type = 'SecurityGroup' AND ingress_rules LIKE '%0.0.0.0/0%')
   OR (type = 'S3Bucket' AND public_access = true)
   OR (type = 'Database' AND encrypted = false);

-- Cross-cloud cost analysis
SELECT provider, 
       SUM(monthly_cost) as total_cost,
       COUNT(*) as resource_count
FROM resources 
GROUP BY provider;
```

**Natural Language Queries (via LLM):**
```bash
cloudrecon ask "show me all databases without backups"
cloudrecon ask "which resources were created last week?"
cloudrecon ask "find potential cost savings over $1000/month"
```

### 4. Integration Points

**Native Cloud Tool Integration:**
```yaml
AWS Config:
  When Available: Use for instant, complete inventory
  Benefits: 
    - 100x faster than API calls
    - Historical data included
    - Compliance rules evaluation
  Integration: 
    - Check if Config recorder is enabled
    - Use Config API for bulk export
    - Fall back to direct APIs if unavailable

Azure Resource Graph:
  When Available: Always (built into Azure)
  Benefits:
    - Query millions of resources in seconds
    - KQL query language support
    - Cross-subscription by default
  Integration:
    - Single API call for entire estate
    - Export to CloudRecon format

GCP Asset Inventory:
  When Available: When enabled on projects
  Benefits:
    - Real-time asset state
    - 35-day history included
    - IAM policy analysis
  Integration:
    - Export to BigQuery or Cloud Storage
    - Stream to CloudRecon via Pub/Sub
```

**External Tool Integration:**
```bash
# Import from existing tools
cloudrecon import --from steampipe
cloudrecon import --from terraform-state s3://bucket/terraform.tfstate
cloudrecon import --from aws-config s3://config-bucket/

# Export to other tools
cloudrecon export --format grafana  # Grafana dashboard JSON
cloudrecon export --format datadog  # Datadog metrics
cloudrecon export --format splunk   # Splunk events
```

## Implementation Roadmap

### Phase 1: MVP (Weeks 1-6)
```yaml
Week 1-2: Core Framework
  - Multi-cloud credential management
  - Plugin architecture for providers
  - SQLite schema design
  
Week 3-4: AWS Discovery
  - AWS Config integration
  - Organizations enumeration
  - Core services (EC2, S3, RDS, IAM)
  
Week 5-6: Azure & GCP Basics
  - Resource Graph queries
  - Cloud Asset Inventory integration
  - Basic resource types
```

### Phase 2: Intelligence Layer (Weeks 7-12)
```yaml
Week 7-8: Smart Discovery
  - Native tool detection and selection
  - Fallback strategies
  - Parallel processing optimization
  
Week 9-10: Query Engine
  - SQL query optimization
  - Query templates library
  - Export formats
  
Week 11-12: Polish
  - Terminal UI with progress bars
  - Error handling and retry logic
  - Comprehensive documentation
```

### Phase 3: Advanced Features (Weeks 13-16)
```yaml
Week 13-14: Continuous Monitoring
  - Daemon mode for real-time updates
  - Change detection and alerts
  - Slack/Teams notifications
  
Week 15-16: Compliance & Security
  - CIS benchmark checks
  - Custom policy engine
  - Remediation scripts
```

## Competitive Advantages

### 1. Speed Through Intelligence
```yaml
Traditional Approach (Steampipe):
  - 500+ API calls per AWS account
  - 30-45 minutes for full scan
  - No caching between runs

CloudRecon Approach:
  - 1 AWS Config API call (if available)
  - 30 seconds for full scan
  - Intelligent caching and incremental updates
```

### 2. Zero Setup Complexity
```yaml
CloudQuery Setup:
  1. Install PostgreSQL
  2. Configure database
  3. Install CloudQuery
  4. Write configuration files
  5. Run sync command
  6. Query database
  Time: 2-4 hours

CloudRecon Setup:
  1. Download binary
  2. Run: cloudrecon discover
  Time: 2 minutes
```

### 3. Cost Efficiency
```yaml
Resource Requirements:
  Steampipe: 4GB RAM for large queries
  CloudQuery: PostgreSQL server + storage
  ServiceNow: Dedicated infrastructure
  
  CloudRecon: 256MB RAM, 100MB disk
```

## Monetization Model

### Open Source Core (Free)
- Basic discovery across all clouds
- Local SQLite storage
- Standard query templates
- Community support

### Pro Edition ($99/month)
- Continuous monitoring daemon
- Advanced query builder
- Slack/Teams integration
- Policy engine with 100+ rules
- Email support

### Enterprise Edition ($499/month)
- SSO/SAML integration
- Custom policies and rules
- API access for automation
- Audit logging
- SLA with priority support
- Remediation playbooks

### Cloud Edition ($999/month)
- Hosted web interface
- Team collaboration
- Historical trending
- Cost optimization recommendations
- Security posture scoring
- Executive dashboards

## Success Metrics

### Launch Goals (Month 1)
- 1,000 GitHub stars
- 100 production users
- 5 enterprise trials

### Growth Goals (Month 6)
- 10,000 active users
- 50 paying customers
- $15,000 MRR
- 3 cloud provider partnerships

### Technical Performance Targets
- Discover 10,000 resources in < 60 seconds
- Support 100+ AWS accounts in single scan
- < 500MB memory usage
- 99.9% discovery accuracy

## Risk Mitigation

### Technical Risks
- **API Rate Limits**: Use native tools, implement exponential backoff
- **Credential Management**: Leverage cloud SDK credential chains
- **Schema Changes**: Versioned schema, automatic migrations

### Business Risks
- **Competition from Cloud Providers**: Focus on multi-cloud value
- **Enterprise Security Concerns**: SOC2 compliance, security audits
- **Support Burden**: Comprehensive docs, community forum

## Code Architecture & Business Logic

### Project Structure
```
cloudrecon/
├── cmd/
│   └── cloudrecon/
│       └── main.go                 # Entry point, CLI setup
├── internal/
│   ├── core/
│   │   ├── discovery.go            # Discovery interfaces and orchestration
│   │   ├── resource.go             # Resource model definitions
│   │   ├── account.go              # Account/subscription/project models
│   │   └── errors.go               # Custom error types
│   ├── providers/
│   │   ├── aws/
│   │   │   ├── discovery.go        # AWS discovery implementation
│   │   │   ├── config.go           # AWS Config integration
│   │   │   ├── organizations.go    # Organizations API
│   │   │   ├── services/           # Service-specific discovery
│   │   │   │   ├── ec2.go
│   │   │   │   ├── s3.go
│   │   │   │   ├── rds.go
│   │   │   │   └── iam.go
│   │   │   └── credentials.go      # AWS credential management
│   │   ├── azure/
│   │   │   ├── discovery.go        # Azure discovery implementation
│   │   │   ├── resourcegraph.go    # Resource Graph queries
│   │   │   ├── subscriptions.go    # Subscription enumeration
│   │   │   └── services/
│   │   └── gcp/
│   │       ├── discovery.go        # GCP discovery implementation
│   │       ├── assetinventory.go   # Cloud Asset Inventory
│   │       └── services/
│   ├── storage/
│   │   ├── sqlite.go               # SQLite operations
│   │   ├── schema.go               # Database schema
│   │   ├── migrations.go           # Schema migrations
│   │   └── cache.go                # Caching layer
│   ├── query/
│   │   ├── engine.go               # Query execution
│   │   ├── parser.go               # Query parsing
│   │   ├── templates.go            # Pre-built queries
│   │   └── natural.go              # Natural language processing
│   ├── export/
│   │   ├── json.go
│   │   ├── csv.go
│   │   ├── terraform.go
│   │   └── grafana.go
│   └── monitor/
│       ├── daemon.go               # Continuous monitoring
│       ├── diff.go                 # Change detection
│       └── alerts.go               # Alert notifications
├── pkg/
│   ├── ratelimit/                  # Rate limiting utilities
│   ├── parallel/                   # Parallel execution helpers
│   ├── retry/                      # Retry logic with backoff
│   └── progress/                   # Progress bars and UI
├── config/
│   └── config.yaml                 # Default configuration
├── scripts/
│   ├── install.sh                  # Installation script
│   └── build.sh                    # Build script
└── tests/
    ├── integration/
    └── unit/
```

### Core Business Logic

#### 1. Discovery Orchestrator
```go
// internal/core/discovery.go

package core

import (
    "context"
    "sync"
    "time"
)

// DiscoveryMode defines how aggressive discovery should be
type DiscoveryMode int

const (
    QuickMode DiscoveryMode = iota  // Critical resources only (30 sec)
    StandardMode                      // Most resources (2 min)
    DeepMode                         // Everything including dependencies (10 min)
)

// DiscoveryOptions configures the discovery process
type DiscoveryOptions struct {
    Mode            DiscoveryMode
    Providers       []string      // aws, azure, gcp
    Accounts        []string      // Specific accounts to scan
    Regions         []string      // Specific regions (empty = all)
    ResourceTypes   []string      // Specific resource types
    UseNativeTools  bool          // Prefer cloud-native tools
    MaxParallel     int           // Max parallel operations
    Timeout         time.Duration
    ProgressHandler func(DiscoveryProgress)
}

// DiscoveryOrchestrator coordinates multi-cloud discovery
type DiscoveryOrchestrator struct {
    providers map[string]CloudProvider
    storage   Storage
    cache     Cache
    options   DiscoveryOptions
    mu        sync.RWMutex
}

// Discover performs multi-cloud discovery
func (d *DiscoveryOrchestrator) Discover(ctx context.Context) (*DiscoveryResult, error) {
    result := &DiscoveryResult{
        StartTime: time.Now(),
        Resources: make([]Resource, 0),
    }
    
    // Phase 1: Enumerate accounts across all providers
    accounts, err := d.discoverAccounts(ctx)
    if err != nil {
        return nil, fmt.Errorf("account discovery failed: %w", err)
    }
    
    // Phase 2: Parallel discovery across accounts
    var wg sync.WaitGroup
    sem := make(chan struct{}, d.options.MaxParallel)
    errorsChan := make(chan error, len(accounts))
    resourcesChan := make(chan []Resource, len(accounts))
    
    for _, account := range accounts {
        wg.Add(1)
        go func(acc Account) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire semaphore
            defer func() { <-sem }() // Release semaphore
            
            resources, err := d.discoverAccountResources(ctx, acc)
            if err != nil {
                errorsChan <- fmt.Errorf("account %s: %w", acc.ID, err)
                return
            }
            resourcesChan <- resources
        }(account)
    }
    
    // Wait for all discoveries to complete
    go func() {
        wg.Wait()
        close(errorsChan)
        close(resourcesChan)
    }()
    
    // Collect results
    for resources := range resourcesChan {
        result.Resources = append(result.Resources, resources...)
    }
    
    // Phase 3: Enrich with relationships and metadata
    if d.options.Mode >= StandardMode {
        d.enrichResources(ctx, result.Resources)
    }
    
    // Phase 4: Store in database
    if err := d.storage.StoreDiscovery(result); err != nil {
        return nil, fmt.Errorf("storage failed: %w", err)
    }
    
    result.EndTime = time.Now()
    return result, nil
}

// discoverAccountResources discovers resources for a single account
func (d *DiscoveryOrchestrator) discoverAccountResources(
    ctx context.Context, 
    account Account,
) ([]Resource, error) {
    provider := d.providers[account.Provider]
    
    // Try native tools first
    if d.options.UseNativeTools {
        if nativeProvider, ok := provider.(NativeToolProvider); ok {
            if available, _ := nativeProvider.IsNativeToolAvailable(ctx, account); available {
                return nativeProvider.DiscoverWithNativeTool(ctx, account)
            }
        }
    }
    
    // Fall back to direct API discovery
    return provider.DiscoverResources(ctx, account, d.options)
}
```

#### 2. Provider Interface with Native Tool Support
```go
// internal/core/provider.go

package core

// CloudProvider is the base interface for all cloud providers
type CloudProvider interface {
    // Name returns the provider name (aws, azure, gcp)
    Name() string
    
    // DiscoverAccounts finds all accounts/subscriptions/projects
    DiscoverAccounts(ctx context.Context) ([]Account, error)
    
    // DiscoverResources discovers resources in an account
    DiscoverResources(ctx context.Context, account Account, opts DiscoveryOptions) ([]Resource, error)
    
    // ValidateCredentials checks if credentials are valid
    ValidateCredentials(ctx context.Context) error
}

// NativeToolProvider supports cloud-native discovery tools
type NativeToolProvider interface {
    CloudProvider
    
    // IsNativeToolAvailable checks if native tool is configured
    IsNativeToolAvailable(ctx context.Context, account Account) (bool, error)
    
    // DiscoverWithNativeTool uses cloud-native tools for discovery
    DiscoverWithNativeTool(ctx context.Context, account Account) ([]Resource, error)
}

// Resource represents a cloud resource
type Resource struct {
    // Core fields
    ID           string            `json:"id" db:"id"`
    ARN          string            `json:"arn,omitempty" db:"arn"`
    Provider     string            `json:"provider" db:"provider"`
    AccountID    string            `json:"account_id" db:"account_id"`
    Region       string            `json:"region" db:"region"`
    Service      string            `json:"service" db:"service"`
    Type         string            `json:"type" db:"type"`
    Name         string            `json:"name" db:"name"`
    
    // Metadata
    CreatedAt    time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
    Tags         map[string]string `json:"tags" db:"tags"`
    
    // Configuration
    Configuration json.RawMessage  `json:"configuration" db:"configuration"`
    
    // Security & Compliance
    PublicAccess  bool             `json:"public_access" db:"public_access"`
    Encrypted     bool             `json:"encrypted" db:"encrypted"`
    Compliance    []string         `json:"compliance" db:"compliance"`
    
    // Cost
    MonthlyCost   float64          `json:"monthly_cost" db:"monthly_cost"`
    
    // Relationships
    Dependencies  []string         `json:"dependencies" db:"dependencies"`
    Dependents    []string         `json:"dependents" db:"dependents"`
    
    // Discovery metadata
    DiscoveredAt  time.Time        `json:"discovered_at" db:"discovered_at"`
    DiscoveryMethod string         `json:"discovery_method" db:"discovery_method"`
}
```

#### 3. AWS Provider with Intelligent Discovery
```go
// internal/providers/aws/discovery.go

package aws

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/config"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/organizations"
)

type AWSProvider struct {
    config      aws.Config
    credentials CredentialManager
    cache       Cache
}

// DiscoverWithNativeTool uses AWS Config when available
func (p *AWSProvider) DiscoverWithNativeTool(ctx context.Context, account Account) ([]Resource, error) {
    // Use AWS Config for comprehensive discovery
    configClient := config.NewFromConfig(p.config)
    
    // Check for Config aggregator (fastest option)
    aggregators, err := configClient.DescribeConfigurationAggregators(ctx, &config.DescribeConfigurationAggregatorsInput{})
    if err == nil && len(aggregators.ConfigurationAggregators) > 0 {
        return p.discoverViaConfigAggregator(ctx, aggregators.ConfigurationAggregators[0])
    }
    
    // Fall back to regular Config
    recorders, err := configClient.DescribeConfigurationRecorders(ctx, &config.DescribeConfigurationRecordersInput{})
    if err == nil && len(recorders.ConfigurationRecorders) > 0 {
        return p.discoverViaConfig(ctx, account)
    }
    
    // Config not available, return error to trigger fallback
    return nil, ErrNativeToolUnavailable
}

// discoverViaConfig uses AWS Config for discovery
func (p *AWSProvider) discoverViaConfig(ctx context.Context, account Account) ([]Resource, error) {
    configClient := config.NewFromConfig(p.config)
    
    var resources []Resource
    var nextToken *string
    
    for {
        // Use SelectResourceConfig for bulk retrieval
        query := `
            SELECT
                resourceId,
                resourceType,
                awsRegion,
                configuration,
                tags,
                resourceCreationTime
            WHERE
                resourceType IN ('AWS::EC2::Instance', 'AWS::S3::Bucket', 'AWS::RDS::DBInstance')
        `
        
        result, err := configClient.SelectResourceConfig(ctx, &config.SelectResourceConfigInput{
            Expression: aws.String(query),
            NextToken:  nextToken,
        })
        
        if err != nil {
            return nil, fmt.Errorf("config query failed: %w", err)
        }
        
        // Parse results
        for _, item := range result.Results {
            resource := p.parseConfigResult(item)
            resources = append(resources, resource)
        }
        
        if result.NextToken == nil {
            break
        }
        nextToken = result.NextToken
    }
    
    return resources, nil
}

// DiscoverResources falls back to direct API calls
func (p *AWSProvider) DiscoverResources(
    ctx context.Context,
    account Account,
    opts DiscoveryOptions,
) ([]Resource, error) {
    var resources []Resource
    
    // Parallel discovery across regions
    regions := opts.Regions
    if len(regions) == 0 {
        regions = p.getAllRegions(ctx)
    }
    
    // Create worker pool for parallel region discovery
    type regionResult struct {
        region    string
        resources []Resource
        err       error
    }
    
    resultChan := make(chan regionResult, len(regions))
    
    for _, region := range regions {
        go func(r string) {
            regionalResources, err := p.discoverRegionalResources(ctx, r, opts)
            resultChan <- regionResult{
                region:    r,
                resources: regionalResources,
                err:       err,
            }
        }(region)
    }
    
    // Collect results
    for i := 0; i < len(regions); i++ {
        result := <-resultChan
        if result.err != nil {
            // Log error but continue with other regions
            continue
        }
        resources = append(resources, result.resources...)
    }
    
    return resources, nil
}

// discoverRegionalResources discovers resources in a specific region
func (p *AWSProvider) discoverRegionalResources(
    ctx context.Context,
    region string,
    opts DiscoveryOptions,
) ([]Resource, error) {
    // Configure regional client
    regionalConfig := p.config.Copy()
    regionalConfig.Region = region
    
    var resources []Resource
    
    // Discover based on mode
    switch opts.Mode {
    case QuickMode:
        // Only critical resources
        resources = append(resources, p.discoverEC2Instances(ctx, regionalConfig, true)...)
        resources = append(resources, p.discoverPublicS3Buckets(ctx, regionalConfig)...)
        resources = append(resources, p.discoverRDSInstances(ctx, regionalConfig, true)...)
        
    case StandardMode:
        // Most resources
        resources = append(resources, p.discoverEC2Resources(ctx, regionalConfig)...)
        resources = append(resources, p.discoverS3Resources(ctx, regionalConfig)...)
        resources = append(resources, p.discoverRDSResources(ctx, regionalConfig)...)
        resources = append(resources, p.discoverLambdaFunctions(ctx, regionalConfig)...)
        
    case DeepMode:
        // Everything including dependencies
        resources = append(resources, p.discoverAllResources(ctx, regionalConfig)...)
        p.mapDependencies(ctx, resources)
    }
    
    return resources, nil
}
```

#### 4. Storage Layer with SQLite
```go
// internal/storage/sqlite.go

package storage

import (
    "database/sql"
    "encoding/json"
    _ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
    db *sql.DB
}

// Initialize creates tables and indexes
func (s *SQLiteStorage) Initialize() error {
    schema := `
    CREATE TABLE IF NOT EXISTS resources (
        id TEXT PRIMARY KEY,
        provider TEXT NOT NULL,
        account_id TEXT NOT NULL,
        region TEXT,
        service TEXT NOT NULL,
        type TEXT NOT NULL,
        name TEXT,
        arn TEXT,
        created_at DATETIME,
        updated_at DATETIME,
        tags TEXT,
        configuration TEXT,
        public_access BOOLEAN DEFAULT FALSE,
        encrypted BOOLEAN DEFAULT FALSE,
        monthly_cost REAL DEFAULT 0,
        dependencies TEXT,
        discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        discovery_method TEXT,
        UNIQUE(provider, account_id, id)
    );
    
    CREATE INDEX IF NOT EXISTS idx_provider ON resources(provider);
    CREATE INDEX IF NOT EXISTS idx_account ON resources(account_id);
    CREATE INDEX IF NOT EXISTS idx_type ON resources(type);
    CREATE INDEX IF NOT EXISTS idx_public ON resources(public_access);
    CREATE INDEX IF NOT EXISTS idx_cost ON resources(monthly_cost);
    
    CREATE TABLE IF NOT EXISTS discovery_runs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        started_at DATETIME NOT NULL,
        completed_at DATETIME,
        resource_count INTEGER DEFAULT 0,
        providers TEXT,
        mode TEXT,
        status TEXT,
        errors TEXT
    );
    
    CREATE TABLE IF NOT EXISTS resource_changes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        resource_id TEXT NOT NULL,
        change_type TEXT NOT NULL, -- created, updated, deleted
        changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        old_configuration TEXT,
        new_configuration TEXT,
        FOREIGN KEY(resource_id) REFERENCES resources(id)
    );
    `
    
    _, err := s.db.Exec(schema)
    return err
}

// StoreDiscovery stores discovery results with deduplication
func (s *SQLiteStorage) StoreDiscovery(result *DiscoveryResult) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Insert discovery run
    runResult, err := tx.Exec(`
        INSERT INTO discovery_runs (started_at, completed_at, resource_count, providers, mode, status)
        VALUES (?, ?, ?, ?, ?, ?)
    `, result.StartTime, result.EndTime, len(result.Resources), 
       strings.Join(result.Providers, ","), result.Mode, "completed")
    
    if err != nil {
        return err
    }
    
    runID, _ := runResult.LastInsertId()
    
    // Prepare statements for efficiency
    insertStmt, _ := tx.Prepare(`
        INSERT OR REPLACE INTO resources (
            id, provider, account_id, region, service, type, name, arn,
            created_at, updated_at, tags, configuration, public_access,
            encrypted, monthly_cost, dependencies, discovered_at, discovery_method
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
    defer insertStmt.Close()
    
    changeStmt, _ := tx.Prepare(`
        INSERT INTO resource_changes (resource_id, change_type, old_configuration, new_configuration)
        VALUES (?, ?, ?, ?)
    `)
    defer changeStmt.Close()
    
    // Process each resource
    for _, resource := range result.Resources {
        // Check if resource exists
        var existingConfig string
        err := tx.QueryRow("SELECT configuration FROM resources WHERE id = ?", resource.ID).
            Scan(&existingConfig)
        
        changeType := "created"
        if err == nil {
            changeType = "updated"
            // Detect actual changes
            if existingConfig != string(resource.Configuration) {
                _, _ = changeStmt.Exec(resource.ID, changeType, existingConfig, resource.Configuration)
            }
        }
        
        // Store resource
        tagsJSON, _ := json.Marshal(resource.Tags)
        depsJSON, _ := json.Marshal(resource.Dependencies)
        
        _, err = insertStmt.Exec(
            resource.ID,
            resource.Provider,
            resource.AccountID,
            resource.Region,
            resource.Service,
            resource.Type,
            resource.Name,
            resource.ARN,
            resource.CreatedAt,
            resource.UpdatedAt,
            string(tagsJSON),
            string(resource.Configuration),
            resource.PublicAccess,
            resource.Encrypted,
            resource.MonthlyCost,
            string(depsJSON),
            resource.DiscoveredAt,
            resource.DiscoveryMethod,
        )
        
        if err != nil {
            return fmt.Errorf("failed to store resource %s: %w", resource.ID, err)
        }
    }
    
    return tx.Commit()
}
```

#### 5. Query Engine
```go
// internal/query/engine.go

package query

type QueryEngine struct {
    storage Storage
    cache   Cache
}

// ExecuteSQL runs raw SQL queries
func (e *QueryEngine) ExecuteSQL(query string, args ...interface{}) ([]Resource, error) {
    // Add safety checks
    if err := e.validateQuery(query); err != nil {
        return nil, err
    }
    
    // Check cache
    cacheKey := fmt.Sprintf("%s:%v", query, args)
    if cached, ok := e.cache.Get(cacheKey); ok {
        return cached.([]Resource), nil
    }
    
    // Execute query
    rows, err := e.storage.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Parse results
    var resources []Resource
    for rows.Next() {
        var r Resource
        if err := rows.Scan(&r); err != nil {
            return nil, err
        }
        resources = append(resources, r)
    }
    
    // Cache results
    e.cache.Set(cacheKey, resources, 5*time.Minute)
    
    return resources, nil
}

// ExecuteTemplate runs predefined query templates
func (e *QueryEngine) ExecuteTemplate(templateName string, params map[string]interface{}) ([]Resource, error) {
    templates := map[string]string{
        "public_resources": `
            SELECT * FROM resources 
            WHERE public_access = true 
            ORDER BY monthly_cost DESC
        `,
        "unused_resources": `
            SELECT * FROM resources 
            WHERE type IN ('AWS::EC2::Instance', 'AWS::RDS::DBInstance')
            AND json_extract(configuration, '$.State') = 'stopped'
            AND datetime(updated_at) < datetime('now', '-7 days')
        `,
        "unencrypted_databases": `
            SELECT * FROM resources
            WHERE service IN ('rds', 'dynamodb', 'redshift')
            AND encrypted = false
        `,
        "cost_optimization": `
            SELECT provider, service, type, 
                   COUNT(*) as count,
                   SUM(monthly_cost) as total_cost
            FROM resources
            WHERE monthly_cost > 0
            GROUP BY provider, service, type
            HAVING total_cost > 100
            ORDER BY total_cost DESC
        `,
    }
    
    query, ok := templates[templateName]
    if !ok {
        return nil, fmt.Errorf("unknown template: %s", templateName)
    }
    
    return e.ExecuteSQL(query)
}
```

## Anti-Patterns to Avoid

### 1. Performance Anti-Patterns

#### ❌ **AVOID: Sequential API Calls**
```go
// BAD: Sequential discovery takes forever
for _, region := range regions {
    resources := discoverRegion(region)  // Blocks until complete
    allResources = append(allResources, resources...)
}
```

#### ✅ **DO: Parallel Discovery with Controlled Concurrency**
```go
// GOOD: Parallel with semaphore to prevent API throttling
sem := make(chan struct{}, 10) // Max 10 concurrent
var wg sync.WaitGroup

for _, region := range regions {
    wg.Add(1)
    go func(r string) {
        defer wg.Done()
        sem <- struct{}{}        // Acquire
        defer func() { <-sem }() // Release
        
        resources := discoverRegion(r)
        mu.Lock()
        allResources = append(allResources, resources...)
        mu.Unlock()
    }(region)
}
wg.Wait()
```

### 2. Error Handling Anti-Patterns

#### ❌ **AVOID: Failing Entire Discovery on Single Error**
```go
// BAD: One failed region kills everything
for _, region := range regions {
    resources, err := discoverRegion(region)
    if err != nil {
        return nil, err  // Stops entire discovery
    }
}
```

#### ✅ **DO: Graceful Degradation with Error Collection**
```go
// GOOD: Continue discovery, collect errors
var errors []error
for _, region := range regions {
    resources, err := discoverRegion(region)
    if err != nil {
        errors = append(errors, fmt.Errorf("region %s: %w", region, err))
        continue  // Keep going
    }
    allResources = append(allResources, resources...)
}

if len(errors) > 0 {
    return allResources, &PartialError{Errors: errors}
}
```

### 3. Memory Anti-Patterns

#### ❌ **AVOID: Loading Everything into Memory**
```go
// BAD: OOM for large environments
allResources := []Resource{}
for _, account := range accounts {
    resources := discoverAccount(account)  // Could be 100k+ resources
    allResources = append(allResources, resources...)
}
return allResources  // Boom! OOM
```

#### ✅ **DO: Stream Results to Storage**
```go
// GOOD: Stream to database as discovered
writer := storage.NewBatchWriter()
for _, account := range accounts {
    resources := discoverAccount(account)
    if err := writer.Write(resources); err != nil {
        return err
    }
    // Resources are freed after writing
}
writer.Flush()
```

### 4. Credential Anti-Patterns

#### ❌ **AVOID: Hardcoding Credentials**
```go
// BAD: Never do this
awsClient := aws.NewClient("AKIAIOSFODNN7EXAMPLE", "SECRET")
```

#### ✅ **DO: Use SDK Credential Chain**
```go
// GOOD: Use SDK's credential chain
cfg, err := config.LoadDefaultConfig(context.TODO())
// Tries: Environment → Shared Config → IAM Role → etc.
```

### 5. API Rate Limit Anti-Patterns

#### ❌ **AVOID: Aggressive Retry Without Backoff**
```go
// BAD: Gets you banned
for retries := 0; retries < 100; retries++ {
    resp, err := api.Call()
    if err == nil {
        return resp
    }
    // Immediately retry - BAD!
}
```

#### ✅ **DO: Exponential Backoff with Jitter**
```go
// GOOD: Respectful retry strategy
backoff := retry.NewExponentialBackoff(
    retry.WithMaxRetries(5),
    retry.WithInitialInterval(1*time.Second),
    retry.WithMaxInterval(30*time.Second),
    retry.WithJitter(0.1),
)

return backoff.Retry(func() error {
    return api.Call()
})
```

### 6. Database Anti-Patterns

#### ❌ **AVOID: N+1 Query Problem**
```go
// BAD: Separate query for each resource
for _, resource := range resources {
    tags := db.Query("SELECT * FROM tags WHERE resource_id = ?", resource.ID)
    resource.Tags = tags
}
```

#### ✅ **DO: Batch Operations**
```go
// GOOD: Single query with JOIN or batch fetch
resources := db.Query(`
    SELECT r.*, t.key, t.value 
    FROM resources r 
    LEFT JOIN tags t ON r.id = t.resource_id
`)
```

### 7. Configuration Anti-Patterns

#### ❌ **AVOID: Scattered Configuration**
```go
// BAD: Config everywhere
awsRegions := []string{"us-east-1", "us-west-2"}  // Hardcoded
maxRetries := 3  // Magic number
timeout := 30    // What unit?
```

#### ✅ **DO: Centralized, Typed Configuration**
```go
// GOOD: Structured config
type Config struct {
    AWS struct {
        Regions    []string      `yaml:"regions"`
        MaxRetries int          `yaml:"max_retries"`
        Timeout    time.Duration `yaml:"timeout"`
    } `yaml:"aws"`
}

config := LoadConfig("config.yaml")
```

### 8. Testing Anti-Patterns

#### ❌ **AVOID: No Interface Boundaries**
```go
// BAD: Untestable
func DiscoverResources() []Resource {
    client := ec2.NewClient()  // Direct instantiation
    return client.DescribeInstances()
}
```

#### ✅ **DO: Dependency Injection**
```go
// GOOD: Testable with mocks
type EC2Client interface {
    DescribeInstances() []Instance
}

func DiscoverResources(client EC2Client) []Resource {
    return client.DescribeInstances()
}
```

### 9. Logging Anti-Patterns

#### ❌ **AVOID: fmt.Println Debugging**
```go
// BAD: No context, no levels
fmt.Println("Starting discovery")
fmt.Println("Found resources:", len(resources))
```

#### ✅ **DO: Structured Logging**
```go
// GOOD: Structured, leveled, contextual
logger.Info("Starting discovery",
    zap.String("provider", "aws"),
    zap.String("account", accountID),
    zap.Int("regions", len(regions)),
)
```

### 10. Concurrency Anti-Patterns

#### ❌ **AVOID: Goroutine Leaks**
```go
// BAD: Goroutine leak
for {
    go func() {
        discover()  // Never exits
    }()
}
```

#### ✅ **DO: Proper Goroutine Lifecycle**
```go
// GOOD: Context cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

go func() {
    select {
    case <-ctx.Done():
        return  // Clean exit
    default:
        discover()
    }
}()
```

## Claude Code Optimization Guide

### For Maximum Claude Code Assistance:

1. **Clear Function Signatures**
```go
// Claude can better understand and extend this:
// DiscoverEC2Instances retrieves all EC2 instances in a region.
// It returns ErrNoCredentials if AWS credentials are not configured.
func DiscoverEC2Instances(ctx context.Context, region string) ([]Resource, error)
```

2. **Explicit Error Types**
```go
// Define custom errors for Claude to handle appropriately
var (
    ErrNoCredentials = errors.New("no valid credentials found")
    ErrRateLimited   = errors.New("API rate limit exceeded")
    ErrPartialResult = errors.New("discovery partially completed")
)
```

3. **Well-Defined Interfaces**
```go
// Claude can implement these interfaces consistently
type Discoverer interface {
    Discover(context.Context) ([]Resource, error)
}

type Enricher interface {
    Enrich(context.Context, []Resource) error
}
```

4. **Comprehensive Test Stubs**
```go
// Provide test structure for Claude to fill in
func TestDiscoverEC2Instances(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() 
        want    []Resource
        wantErr error
    }{
        // Claude: Add test cases here
    }
}
```

5. **TODO Comments with Context**
```go
// TODO(claude): Implement exponential backoff with jitter
// Requirements: Max 5 retries, initial delay 1s, max delay 30s
// Should handle rate limit errors specifically
```

## Conclusion

CloudRecon's architecture emphasizes intelligent discovery through cloud-native tool integration, efficient parallel processing, and robust error handling. By avoiding common anti-patterns and following the structured approach outlined here, the tool can deliver 10x better performance than existing solutions while maintaining simplicity and reliability.

The hybrid approach of leveraging cloud-native tools when available, combined with direct API fallbacks, ensures both optimal performance and complete coverage. This positions CloudRecon as the definitive solution for cloud discovery - simple enough for startups, powerful enough for enterprises.