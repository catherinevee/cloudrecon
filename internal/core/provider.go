package core

import (
	"context"
	"time"
)

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

// Storage interface for persisting discovery results
type Storage interface {
	// Initialize sets up the storage backend
	Initialize() error

	// StoreDiscovery stores discovery results
	StoreDiscovery(result *DiscoveryResult) error

	// GetResources retrieves resources based on query
	GetResources(query string, args ...interface{}) ([]Resource, error)

	// GetDiscoveryStatus returns status of last discovery
	GetDiscoveryStatus() (*DiscoveryStatus, error)

	// Query executes a raw SQL query
	Query(query string, args ...interface{}) (Rows, error)

	// GetResourceCount returns the total number of resources
	GetResourceCount() (int, error)

	// GetResourceSummary returns aggregated resource statistics
	GetResourceSummary() (*ResourceSummary, error)

	// Close closes the storage connection
	Close() error
}

// Rows represents database rows
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
}

// Cache interface for caching query results
type Cache interface {
	// Get retrieves a value from cache
	Get(key string) (interface{}, bool)

	// Set stores a value in cache with TTL
	Set(key string, value interface{}, ttl time.Duration)

	// Delete removes a value from cache
	Delete(key string)

	// Clear removes all values from cache
	Clear()
}

// Account represents a cloud account/subscription/project
type Account struct {
	ID          string            `json:"id"`
	Provider    string            `json:"provider"`
	Name        string            `json:"name"`
	Type        string            `json:"type"` // account, subscription, project
	Region      string            `json:"region,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Credentials Credentials       `json:"-"`
}

// Credentials represents cloud provider credentials
type Credentials struct {
	AccessKey    string `json:"access_key,omitempty"`
	SecretKey    string `json:"secret_key,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	KeyFile      string `json:"key_file,omitempty"`
}

// DiscoveryMode defines how aggressive discovery should be
type DiscoveryMode int

const (
	QuickMode    DiscoveryMode = iota // Critical resources only (30 sec)
	StandardMode                      // Most resources (2 min)
	DeepMode                          // Everything including dependencies (10 min)
)

// DiscoveryOptions configures the discovery process
type DiscoveryOptions struct {
	Mode            DiscoveryMode
	Providers       []string // aws, azure, gcp
	Accounts        []string // Specific accounts to scan
	Regions         []string // Specific regions (empty = all)
	ResourceTypes   []string // Specific resource types
	UseNativeTools  bool     // Prefer cloud-native tools
	MaxParallel     int      // Max parallel operations
	Timeout         time.Duration
	ProgressHandler func(DiscoveryProgress)
}

// Config represents the application configuration
type Config struct {
	Storage StorageConfig `yaml:"storage"`
	AWS     AWSConfig     `yaml:"aws"`
	Azure   AzureConfig   `yaml:"azure"`
	GCP     GCPConfig     `yaml:"gcp"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	DatabasePath string `yaml:"database_path"`
	CacheSize    int    `yaml:"cache_size"`
}

// AWSConfig represents AWS configuration
type AWSConfig struct {
	Regions    []string `yaml:"regions"`
	MaxRetries int      `yaml:"max_retries"`
	Timeout    string   `yaml:"timeout"`
}

// AzureConfig represents Azure configuration
type AzureConfig struct {
	Subscriptions []string `yaml:"subscriptions"`
	MaxRetries    int      `yaml:"max_retries"`
	Timeout       string   `yaml:"timeout"`
}

// GCPConfig represents GCP configuration
type GCPConfig struct {
	Projects   []string `yaml:"projects"`
	MaxRetries int      `yaml:"max_retries"`
	Timeout    string   `yaml:"timeout"`
}
