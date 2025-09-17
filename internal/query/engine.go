package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
)

type QueryEngine struct {
	storage core.Storage
	cache   core.Cache
}

// NewEngine creates a new query engine
func NewEngine(storage core.Storage) *QueryEngine {
	return &QueryEngine{
		storage: storage,
		cache:   &memoryCache{},
	}
}

// ExecuteSQL runs raw SQL queries
func (e *QueryEngine) ExecuteSQL(query string, args ...interface{}) ([]core.Resource, error) {
	// Add safety checks
	if err := e.validateQuery(query); err != nil {
		return nil, err
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s:%v", query, args)
	if cached, ok := e.cache.Get(cacheKey); ok {
		return cached.([]core.Resource), nil
	}

	// Execute query
	rows, err := e.storage.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Parse results
	var resources []core.Resource
	for rows.Next() {
		var resource core.Resource
		var tagsJSON, depsJSON string

		err := rows.Scan(
			&resource.ID,
			&resource.Provider,
			&resource.AccountID,
			&resource.Region,
			&resource.Service,
			&resource.Type,
			&resource.Name,
			&resource.ARN,
			&resource.CreatedAt,
			&resource.UpdatedAt,
			&tagsJSON,
			&resource.Configuration,
			&resource.PublicAccess,
			&resource.Encrypted,
			&resource.MonthlyCost,
			&depsJSON,
			&resource.DiscoveredAt,
			&resource.DiscoveryMethod,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}

		// Parse JSON fields
		if tagsJSON != "" {
			// TODO: Parse tags JSON
		}
		if depsJSON != "" {
			// TODO: Parse dependencies JSON
		}

		resources = append(resources, resource)
	}

	// Cache results
	e.cache.Set(cacheKey, resources, 5*time.Minute)

	return resources, nil
}

// ExecuteTemplate runs predefined query templates
func (e *QueryEngine) ExecuteTemplate(templateName string, params map[string]interface{}) ([]core.Resource, error) {
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
		"security_issues": `
			SELECT * FROM resources
			WHERE (type = 'SecurityGroup' AND json_extract(configuration, '$.IngressRules') LIKE '%0.0.0.0/0%')
			   OR (type = 'S3Bucket' AND public_access = true)
			   OR (type = 'Database' AND encrypted = false)
		`,
		"recent_resources": `
			SELECT * FROM resources
			WHERE datetime(created_at) > datetime('now', '-7 days')
			ORDER BY created_at DESC
		`,
		"high_cost_resources": `
			SELECT * FROM resources
			WHERE monthly_cost > 1000
			ORDER BY monthly_cost DESC
		`,
		"by_provider": `
			SELECT provider, COUNT(*) as count, SUM(monthly_cost) as total_cost
			FROM resources
			GROUP BY provider
			ORDER BY total_cost DESC
		`,
		"by_service": `
			SELECT service, COUNT(*) as count, SUM(monthly_cost) as total_cost
			FROM resources
			GROUP BY service
			ORDER BY total_cost DESC
		`,
		"by_region": `
			SELECT region, COUNT(*) as count, SUM(monthly_cost) as total_cost
			FROM resources
			GROUP BY region
			ORDER BY total_cost DESC
		`,
	}

	query, ok := templates[templateName]
	if !ok {
		return nil, fmt.Errorf("unknown template: %s", templateName)
	}

	return e.ExecuteSQL(query)
}

// validateQuery performs basic SQL validation
func (e *QueryEngine) validateQuery(query string) error {
	// Basic safety checks
	query = strings.ToLower(query)

	// Prevent dangerous operations
	dangerousOps := []string{
		"drop", "delete", "insert", "update", "alter", "create", "truncate",
		"exec", "execute", "sp_", "xp_", "cmdshell",
	}

	for _, op := range dangerousOps {
		if strings.Contains(query, op) {
			return fmt.Errorf("dangerous operation not allowed: %s", op)
		}
	}

	// Must be a SELECT query
	if !strings.HasPrefix(strings.TrimSpace(query), "select") {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	return nil
}

// GetResourceSummary returns aggregated resource statistics
func (e *QueryEngine) GetResourceSummary() (*core.ResourceSummary, error) {
	// Check cache first
	if cached, ok := e.cache.Get("resource_summary"); ok {
		return cached.(*core.ResourceSummary), nil
	}

	// Get summary from storage
	summary, err := e.storage.GetResourceSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to get resource summary: %w", err)
	}

	// Cache for 1 minute
	e.cache.Set("resource_summary", summary, 1*time.Minute)

	return summary, nil
}

// GetResourceCount returns the total number of resources
func (e *QueryEngine) GetResourceCount() (int, error) {
	// Check cache first
	if cached, ok := e.cache.Get("resource_count"); ok {
		return cached.(int), nil
	}

	count, err := e.storage.GetResourceCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get resource count: %w", err)
	}

	// Cache for 1 minute
	e.cache.Set("resource_count", count, 1*time.Minute)

	return count, nil
}

// SearchResources searches resources by name, type, or tags
func (e *QueryEngine) SearchResources(query string) ([]core.Resource, error) {
	sqlQuery := `
		SELECT * FROM resources
		WHERE name LIKE ? 
		   OR type LIKE ?
		   OR service LIKE ?
		   OR tags LIKE ?
		ORDER BY name
	`

	searchTerm := "%" + query + "%"
	return e.ExecuteSQL(sqlQuery, searchTerm, searchTerm, searchTerm, searchTerm)
}

// GetResourcesByProvider returns resources for a specific provider
func (e *QueryEngine) GetResourcesByProvider(provider string) ([]core.Resource, error) {
	sqlQuery := "SELECT * FROM resources WHERE provider = ? ORDER BY name"
	return e.ExecuteSQL(sqlQuery, provider)
}

// GetResourcesByService returns resources for a specific service
func (e *QueryEngine) GetResourcesByService(service string) ([]core.Resource, error) {
	sqlQuery := "SELECT * FROM resources WHERE service = ? ORDER BY name"
	return e.ExecuteSQL(sqlQuery, service)
}

// GetResourcesByType returns resources for a specific type
func (e *QueryEngine) GetResourcesByType(resourceType string) ([]core.Resource, error) {
	sqlQuery := "SELECT * FROM resources WHERE type = ? ORDER BY name"
	return e.ExecuteSQL(sqlQuery, resourceType)
}

// GetResourcesByRegion returns resources for a specific region
func (e *QueryEngine) GetResourcesByRegion(region string) ([]core.Resource, error) {
	sqlQuery := "SELECT * FROM resources WHERE region = ? ORDER BY name"
	return e.ExecuteSQL(sqlQuery, region)
}

// GetPublicResources returns all public resources
func (e *QueryEngine) GetPublicResources() ([]core.Resource, error) {
	return e.ExecuteTemplate("public_resources", nil)
}

// GetUnusedResources returns potentially unused resources
func (e *QueryEngine) GetUnusedResources() ([]core.Resource, error) {
	return e.ExecuteTemplate("unused_resources", nil)
}

// GetUnencryptedDatabases returns unencrypted databases
func (e *QueryEngine) GetUnencryptedDatabases() ([]core.Resource, error) {
	return e.ExecuteTemplate("unencrypted_databases", nil)
}

// GetSecurityIssues returns resources with security issues
func (e *QueryEngine) GetSecurityIssues() ([]core.Resource, error) {
	return e.ExecuteTemplate("security_issues", nil)
}

// GetHighCostResources returns high-cost resources
func (e *QueryEngine) GetHighCostResources() ([]core.Resource, error) {
	return e.ExecuteTemplate("high_cost_resources", nil)
}

// GetRecentResources returns recently created resources
func (e *QueryEngine) GetRecentResources() ([]core.Resource, error) {
	return e.ExecuteTemplate("recent_resources", nil)
}

// memoryCache is a simple in-memory cache implementation
type memoryCache struct {
	data map[string]interface{}
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
	value, ok := c.data[key]
	return value, ok
}

func (c *memoryCache) Set(key string, value interface{}, ttl time.Duration) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}

func (c *memoryCache) Delete(key string) {
	delete(c.data, key)
}

func (c *memoryCache) Clear() {
	c.data = make(map[string]interface{})
}
