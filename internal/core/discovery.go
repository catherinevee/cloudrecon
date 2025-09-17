package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DiscoveryOrchestrator coordinates multi-cloud discovery
type DiscoveryOrchestrator struct {
	providers map[string]CloudProvider
	storage   Storage
	cache     Cache
	options   DiscoveryOptions
	mu        sync.RWMutex
}

// NewDiscoveryOrchestrator creates a new discovery orchestrator
func NewDiscoveryOrchestrator(
	providers map[string]CloudProvider,
	storage Storage,
	options DiscoveryOptions,
) *DiscoveryOrchestrator {
	return &DiscoveryOrchestrator{
		providers: providers,
		storage:   storage,
		options:   options,
	}
}

// Discover performs multi-cloud discovery
func (d *DiscoveryOrchestrator) Discover(ctx context.Context) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		StartTime: time.Now(),
		Resources: make([]Resource, 0),
		Accounts:  make([]Account, 0),
		Providers: make([]string, 0),
		Mode:      d.options.Mode,
		Errors:    make([]error, 0),
	}

	// Phase 1: Enumerate accounts across all providers
	accounts, err := d.discoverAccounts(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("account discovery failed: %w", err))
		return result, err
	}
	result.Accounts = accounts

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

	// Collect errors
	for err := range errorsChan {
		result.Errors = append(result.Errors, err)
	}

	// Phase 3: Enrich with relationships and metadata
	if d.options.Mode >= StandardMode {
		d.enrichResources(ctx, result.Resources)
	}

	// Phase 4: Store in database
	if err := d.storage.StoreDiscovery(result); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("storage failed: %w", err))
		return result, err
	}

	result.EndTime = time.Now()
	return result, nil
}

// discoverAccounts discovers all accounts across all providers
func (d *DiscoveryOrchestrator) discoverAccounts(ctx context.Context) ([]Account, error) {
	var allAccounts []Account
	var wg sync.WaitGroup
	var mu sync.Mutex

	for providerName, provider := range d.providers {
		wg.Add(1)
		go func(name string, p CloudProvider) {
			defer wg.Done()

			accounts, err := p.DiscoverAccounts(ctx)
			if err != nil {
				// Log error but continue with other providers
				return
			}

			mu.Lock()
			allAccounts = append(allAccounts, accounts...)
			mu.Unlock()
		}(providerName, provider)
	}

	wg.Wait()
	return allAccounts, nil
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

// enrichResources enriches resources with relationships and metadata
func (d *DiscoveryOrchestrator) enrichResources(ctx context.Context, resources []Resource) {
	// TODO: Implement resource enrichment
	// - Map dependencies between resources
	// - Calculate costs
	// - Assess security posture
	// - Check compliance status
}

// DiscoveryProgress represents progress during discovery
type DiscoveryProgress struct {
	ResourcesFound    int
	AccountsProcessed int
	CurrentAccount    string
	CurrentProvider   string
	ElapsedTime       time.Duration
}

// DiscoveryResult contains the results of a discovery run
type DiscoveryResult struct {
	StartTime time.Time
	EndTime   time.Time
	Resources []Resource
	Accounts  []Account
	Providers []string
	Mode      DiscoveryMode
	Errors    []error
}

// DiscoveryStatus represents the status of discovery
type DiscoveryStatus struct {
	LastRun       time.Time
	ResourceCount int
	Providers     string
	Status        string
	Errors        []string
}
