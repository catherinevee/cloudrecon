package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(databasePath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.Initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return storage, nil
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
	CREATE INDEX IF NOT EXISTS idx_service ON resources(service);
	CREATE INDEX IF NOT EXISTS idx_region ON resources(region);
	
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
	
	CREATE TABLE IF NOT EXISTS resource_relationships (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		relationship TEXT NOT NULL,
		weight INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(source_id) REFERENCES resources(id),
		FOREIGN KEY(target_id) REFERENCES resources(id)
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// StoreResources stores multiple resources
func (s *SQLiteStorage) StoreResources(ctx context.Context, resources []core.Resource) error {
	if len(resources) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log rollback error but don't fail the operation
			_ = rollbackErr
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO resources (
			id, provider, account_id, region, service, type, name, arn,
			created_at, updated_at, discovered_at, discovery_method,
			tags, configuration, public_access, encrypted, monthly_cost
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, resource := range resources {
		tagsJSON, _ := json.Marshal(resource.Tags)
		_, err := stmt.ExecContext(ctx,
			resource.ID, resource.Provider, resource.AccountID, resource.Region,
			resource.Service, resource.Type, resource.Name, resource.ARN,
			resource.CreatedAt, resource.UpdatedAt, resource.DiscoveredAt,
			resource.DiscoveryMethod, string(tagsJSON), string(resource.Configuration),
			resource.PublicAccess, resource.Encrypted, resource.MonthlyCost,
		)
		if err != nil {
			return fmt.Errorf("failed to insert resource %s: %w", resource.ID, err)
		}
	}

	return tx.Commit()
}

// StoreDiscovery stores discovery results with deduplication
func (s *SQLiteStorage) StoreDiscovery(result *core.DiscoveryResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log rollback error but don't fail the operation
			_ = rollbackErr
		}
	}()

	// Insert discovery run
	providers := make([]string, 0, len(result.Providers))
	for _, p := range result.Providers {
		providers = append(providers, p)
	}

	errorsJSON, _ := json.Marshal(result.Errors)

	runResult, err := tx.Exec(`
		INSERT INTO discovery_runs (started_at, completed_at, resource_count, providers, mode, status, errors)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, result.StartTime, result.EndTime, len(result.Resources),
		strings.Join(providers, ","), result.Mode, "completed", string(errorsJSON))

	if err != nil {
		return fmt.Errorf("failed to insert discovery run: %w", err)
	}

	_, _ = runResult.LastInsertId()

	// Prepare statements for efficiency
	insertStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO resources (
			id, provider, account_id, region, service, type, name, arn,
			created_at, updated_at, tags, configuration, public_access,
			encrypted, monthly_cost, dependencies, discovered_at, discovery_method
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	changeStmt, err := tx.Prepare(`
		INSERT INTO resource_changes (resource_id, change_type, old_configuration, new_configuration)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare change statement: %w", err)
	}
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

// GetResources retrieves resources based on query
func (s *SQLiteStorage) GetResources(query string, args ...interface{}) ([]core.Resource, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

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
			if err := json.Unmarshal([]byte(tagsJSON), &resource.Tags); err != nil {
				resource.Tags = make(map[string]string)
			}
		}
		if depsJSON != "" {
			if err := json.Unmarshal([]byte(depsJSON), &resource.Dependencies); err != nil {
				resource.Dependencies = make([]string, 0)
			}
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// GetDiscoveryStatus returns status of last discovery
func (s *SQLiteStorage) GetDiscoveryStatus() (*core.DiscoveryStatus, error) {
	var status core.DiscoveryStatus

	err := s.db.QueryRow(`
		SELECT started_at, resource_count, providers, status, errors
		FROM discovery_runs
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&status.LastRun, &status.ResourceCount, &status.Providers, &status.Status, &status.Errors)

	if err != nil {
		if err == sql.ErrNoRows {
			return &core.DiscoveryStatus{
				LastRun:       time.Time{},
				ResourceCount: 0,
				Providers:     "",
				Status:        "never_run",
				Errors:        []string{},
			}, nil
		}
		return nil, fmt.Errorf("failed to get discovery status: %w", err)
	}

	return &status, nil
}

// Close closes the storage connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// Query executes a raw SQL query
func (s *SQLiteStorage) Query(query string, args ...interface{}) (core.Rows, error) {
	return s.db.Query(query, args...)
}

// Exec executes a raw SQL statement
func (s *SQLiteStorage) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// GetResourceCount returns the total number of resources
func (s *SQLiteStorage) GetResourceCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM resources").Scan(&count)
	return count, err
}

// GetResourceSummary returns aggregated resource statistics
func (s *SQLiteStorage) GetResourceSummary() (*core.ResourceSummary, error) {
	summary := &core.ResourceSummary{
		ByProvider:     make(map[string]int),
		ByService:      make(map[string]int),
		ByType:         make(map[string]int),
		ByRegion:       make(map[string]int),
		CostByProvider: make(map[string]float64),
	}

	// Total resources
	err := s.db.QueryRow("SELECT COUNT(*) FROM resources").Scan(&summary.TotalResources)
	if err != nil {
		return nil, fmt.Errorf("failed to get total resources: %w", err)
	}

	// By provider
	rows, err := s.db.Query("SELECT provider, COUNT(*) FROM resources GROUP BY provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get provider stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			return nil, err
		}
		summary.ByProvider[provider] = count
	}

	// By service
	rows, err = s.db.Query("SELECT service, COUNT(*) FROM resources GROUP BY service")
	if err != nil {
		return nil, fmt.Errorf("failed to get service stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var service string
		var count int
		if err := rows.Scan(&service, &count); err != nil {
			return nil, err
		}
		summary.ByService[service] = count
	}

	// By type
	rows, err = s.db.Query("SELECT type, COUNT(*) FROM resources GROUP BY type")
	if err != nil {
		return nil, fmt.Errorf("failed to get type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var resourceType string
		var count int
		if err := rows.Scan(&resourceType, &count); err != nil {
			return nil, err
		}
		summary.ByType[resourceType] = count
	}

	// By region
	rows, err = s.db.Query("SELECT region, COUNT(*) FROM resources GROUP BY region")
	if err != nil {
		return nil, fmt.Errorf("failed to get region stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var region string
		var count int
		if err := rows.Scan(&region, &count); err != nil {
			return nil, err
		}
		summary.ByRegion[region] = count
	}

	// Total cost
	err = s.db.QueryRow("SELECT SUM(monthly_cost) FROM resources").Scan(&summary.TotalCost)
	if err != nil {
		return nil, fmt.Errorf("failed to get total cost: %w", err)
	}

	// Cost by provider
	rows, err = s.db.Query("SELECT provider, SUM(monthly_cost) FROM resources GROUP BY provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get cost by provider: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var cost float64
		if err := rows.Scan(&provider, &cost); err != nil {
			return nil, err
		}
		summary.CostByProvider[provider] = cost
	}

	// Security issues
	err = s.db.QueryRow("SELECT COUNT(*) FROM resources WHERE public_access = true").Scan(&summary.SecurityIssues)
	if err != nil {
		return nil, fmt.Errorf("failed to get security issues: %w", err)
	}

	return summary, nil
}
