package core

import (
	"encoding/json"
	"time"
)

// Resource represents a cloud resource
type Resource struct {
	// Core fields
	ID        string `json:"id" db:"id"`
	ARN       string `json:"arn,omitempty" db:"arn"`
	Provider  string `json:"provider" db:"provider"`
	AccountID string `json:"account_id" db:"account_id"`
	Region    string `json:"region" db:"region"`
	Service   string `json:"service" db:"service"`
	Type      string `json:"type" db:"type"`
	Name      string `json:"name" db:"name"`

	// Metadata
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
	Tags      map[string]string `json:"tags" db:"tags"`

	// Configuration
	Configuration json.RawMessage `json:"configuration" db:"configuration"`

	// Security & Compliance
	PublicAccess bool     `json:"public_access" db:"public_access"`
	Encrypted    bool     `json:"encrypted" db:"encrypted"`
	Compliance   []string `json:"compliance" db:"compliance"`

	// Cost
	MonthlyCost float64 `json:"monthly_cost" db:"monthly_cost"`

	// Relationships
	Dependencies []string `json:"dependencies" db:"dependencies"`
	Dependents   []string `json:"dependents" db:"dependents"`

	// Discovery metadata
	DiscoveredAt    time.Time `json:"discovered_at" db:"discovered_at"`
	DiscoveryMethod string    `json:"discovery_method" db:"discovery_method"`
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	ID               int             `json:"id" db:"id"`
	ResourceID       string          `json:"resource_id" db:"resource_id"`
	ChangeType       string          `json:"change_type" db:"change_type"` // created, updated, deleted
	ChangedAt        time.Time       `json:"changed_at" db:"changed_at"`
	OldConfiguration json.RawMessage `json:"old_configuration" db:"old_configuration"`
	NewConfiguration json.RawMessage `json:"new_configuration" db:"new_configuration"`
}

// ResourceFilter represents filters for querying resources
type ResourceFilter struct {
	Providers     []string
	Accounts      []string
	Regions       []string
	Services      []string
	Types         []string
	PublicAccess  *bool
	Encrypted     *bool
	MinCost       *float64
	MaxCost       *float64
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Tags          map[string]string
}

// ResourceSummary represents aggregated resource statistics
type ResourceSummary struct {
	TotalResources   int                `json:"total_resources"`
	ByProvider       map[string]int     `json:"by_provider"`
	ByService        map[string]int     `json:"by_service"`
	ByType           map[string]int     `json:"by_type"`
	ByRegion         map[string]int     `json:"by_region"`
	TotalCost        float64            `json:"total_cost"`
	CostByProvider   map[string]float64 `json:"cost_by_provider"`
	SecurityIssues   int                `json:"security_issues"`
	ComplianceIssues int                `json:"compliance_issues"`
}

// ResourceQuery represents a query for resources
type ResourceQuery struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// ResourceTemplate represents a predefined query template
type ResourceTemplate struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SQL         string          `json:"sql"`
	Parameters  []TemplateParam `json:"parameters"`
	Category    string          `json:"category"`
}

// TemplateParam represents a parameter for a query template
type TemplateParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
}

// ResourceRelationship represents a relationship between resources
type ResourceRelationship struct {
	SourceID     string `json:"source_id"`
	TargetID     string `json:"target_id"`
	Relationship string `json:"relationship"` // depends_on, contains, uses, etc.
	Weight       int    `json:"weight"`       // Relationship strength
}

// ResourceMetrics represents performance metrics for resources
type ResourceMetrics struct {
	ResourceID     string    `json:"resource_id"`
	CPUUtilization float64   `json:"cpu_utilization"`
	MemoryUsage    float64   `json:"memory_usage"`
	NetworkIn      float64   `json:"network_in"`
	NetworkOut     float64   `json:"network_out"`
	DiskUsage      float64   `json:"disk_usage"`
	LastActivity   time.Time `json:"last_activity"`
	Uptime         float64   `json:"uptime"`
}

// ResourceCost represents cost information for a resource
type ResourceCost struct {
	ResourceID    string             `json:"resource_id"`
	Service       string             `json:"service"`
	ResourceType  string             `json:"resource_type"`
	Region        string             `json:"region"`
	MonthlyCost   float64            `json:"monthly_cost"`
	DailyCost     float64            `json:"daily_cost"`
	HourlyCost    float64            `json:"hourly_cost"`
	Currency      string             `json:"currency"`
	LastUpdated   time.Time          `json:"last_updated"`
	CostBreakdown map[string]float64 `json:"cost_breakdown"`
}

// ResourceSecurity represents security information for a resource
type ResourceSecurity struct {
	ResourceID       string    `json:"resource_id"`
	PublicAccess     bool      `json:"public_access"`
	Encrypted        bool      `json:"encrypted"`
	EncryptionKey    string    `json:"encryption_key,omitempty"`
	SecurityGroups   []string  `json:"security_groups,omitempty"`
	IAMRoles         []string  `json:"iam_roles,omitempty"`
	ComplianceStatus string    `json:"compliance_status"`
	Vulnerabilities  []string  `json:"vulnerabilities,omitempty"`
	LastScanned      time.Time `json:"last_scanned"`
}

// ResourceCompliance represents compliance information for a resource
type ResourceCompliance struct {
	ResourceID string              `json:"resource_id"`
	Standards  []string            `json:"standards"` // CIS, SOC2, PCI-DSS, etc.
	Status     map[string]string   `json:"status"`    // standard -> status
	LastAudit  time.Time           `json:"last_audit"`
	Auditor    string              `json:"auditor"`
	Findings   []ComplianceFinding `json:"findings"`
}

// ComplianceFinding represents a compliance finding
type ComplianceFinding struct {
	ID          string `json:"id"`
	Standard    string `json:"standard"`
	Rule        string `json:"rule"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Description string `json:"description"`
	Remediation string `json:"remediation"`
	Status      string `json:"status"` // pass, fail, not_applicable
}
