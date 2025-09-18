package analysis

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// PerformanceConfig holds configuration for performance optimization
type PerformanceConfig struct {
	MaxWorkers      int           `json:"max_workers"`
	BatchSize       int           `json:"batch_size"`
	CacheTimeout    time.Duration `json:"cache_timeout"`
	EnableParallel  bool          `json:"enable_parallel"`
	MemoryOptimize  bool          `json:"memory_optimize"`
	EnableProfiling bool          `json:"enable_profiling"`
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		MaxWorkers:      runtime.NumCPU(),
		BatchSize:       100,
		CacheTimeout:    5 * time.Minute,
		EnableParallel:  true,
		MemoryOptimize:  true,
		EnableProfiling: false,
	}
}

// PerformanceOptimizedDependencyAnalyzer is an optimized version of DependencyAnalyzer
type PerformanceOptimizedDependencyAnalyzer struct {
	*DependencyAnalyzer
	config     *PerformanceConfig
	cache      map[string]interface{}
	cacheMutex sync.RWMutex
}

// NewPerformanceOptimizedDependencyAnalyzer creates a new performance-optimized dependency analyzer
func NewPerformanceOptimizedDependencyAnalyzer(storage core.Storage, config *PerformanceConfig) *PerformanceOptimizedDependencyAnalyzer {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	return &PerformanceOptimizedDependencyAnalyzer{
		DependencyAnalyzer: NewDependencyAnalyzer(storage),
		config:             config,
		cache:              make(map[string]interface{}),
	}
}

// AnalyzeDependenciesOptimized performs optimized dependency analysis with parallel processing
func (poda *PerformanceOptimizedDependencyAnalyzer) AnalyzeDependenciesOptimized(ctx context.Context) (*DependencyGraph, error) {
	start := time.Now()
	logrus.Info("Starting optimized dependency analysis")

	// Get all resources with caching
	resources, err := poda.getResourcesCached(ctx)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return &DependencyGraph{
			Resources:    []core.Resource{},
			Dependencies: []Dependency{},
			Stats:        GraphStats{},
		}, nil
	}

	// Group resources by provider for parallel processing
	providerResources := poda.groupResourcesByProvider(resources)

	// Process providers in parallel
	var dependencies []Dependency
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create worker pool
	workerCount := poda.config.MaxWorkers
	if len(providerResources) < workerCount {
		workerCount = len(providerResources)
	}

	providerChan := make(chan providerWork, len(providerResources))

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go poda.worker(ctx, providerChan, &dependencies, &mu, &wg)
	}

	// Send work to workers
	for provider, resources := range providerResources {
		providerChan <- providerWork{
			Provider:  provider,
			Resources: resources,
		}
	}
	close(providerChan)

	// Wait for all workers to complete
	wg.Wait()

	// Analyze cross-provider dependencies in parallel
	crossProviderDeps, err := poda.analyzeCrossProviderDependenciesParallel(ctx, providerResources)
	if err != nil {
		logrus.Warnf("Failed to analyze cross-provider dependencies: %v", err)
	}
	dependencies = append(dependencies, crossProviderDeps...)

	// Calculate graph statistics
	stats := poda.calculateGraphStatsOptimized(resources, dependencies)

	graph := &DependencyGraph{
		Resources:    resources,
		Dependencies: dependencies,
		Stats:        stats,
	}

	duration := time.Since(start)
	logrus.Infof("Optimized dependency analysis completed: %d resources, %d dependencies in %v",
		len(resources), len(dependencies), duration)

	return graph, nil
}

type providerWork struct {
	Provider  string
	Resources []core.Resource
}

// worker processes provider work items
func (poda *PerformanceOptimizedDependencyAnalyzer) worker(ctx context.Context, workChan <-chan providerWork, dependencies *[]Dependency, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	for work := range workChan {
		select {
		case <-ctx.Done():
			return
		default:
			providerDeps, err := poda.analyzeProviderDependencies(ctx, work.Provider, work.Resources)
			if err != nil {
				logrus.Warnf("Failed to analyze dependencies for %s: %v", work.Provider, err)
				continue
			}

			mu.Lock()
			*dependencies = append(*dependencies, providerDeps...)
			mu.Unlock()
		}
	}
}

// getResourcesCached retrieves resources with caching
func (poda *PerformanceOptimizedDependencyAnalyzer) getResourcesCached(ctx context.Context) ([]core.Resource, error) {
	cacheKey := "resources_all"

	// Check cache first
	poda.cacheMutex.RLock()
	if cached, exists := poda.cache[cacheKey]; exists {
		if resources, ok := cached.([]core.Resource); ok {
			poda.cacheMutex.RUnlock()
			return resources, nil
		}
	}
	poda.cacheMutex.RUnlock()

	// Get from storage
	resources, err := poda.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, err
	}

	// Cache the result
	poda.cacheMutex.Lock()
	poda.cache[cacheKey] = resources
	poda.cacheMutex.Unlock()

	return resources, nil
}

// groupResourcesByProvider groups resources by provider efficiently
func (poda *PerformanceOptimizedDependencyAnalyzer) groupResourcesByProvider(resources []core.Resource) map[string][]core.Resource {
	providerResources := make(map[string][]core.Resource)

	for _, resource := range resources {
		providerResources[resource.Provider] = append(providerResources[resource.Provider], resource)
	}

	return providerResources
}

// analyzeCrossProviderDependenciesParallel analyzes cross-provider dependencies in parallel
func (poda *PerformanceOptimizedDependencyAnalyzer) analyzeCrossProviderDependenciesParallel(ctx context.Context, providerResources map[string][]core.Resource) ([]Dependency, error) {
	if len(providerResources) < 2 {
		return []Dependency{}, nil
	}

	// Create all possible provider pairs
	providers := make([]string, 0, len(providerResources))
	for provider := range providerResources {
		providers = append(providers, provider)
	}

	var dependencies []Dependency
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process provider pairs in parallel
	for i := 0; i < len(providers); i++ {
		for j := i + 1; j < len(providers); j++ {
			wg.Add(1)
			go func(provider1, provider2 string) {
				defer wg.Done()

				deps := poda.analyzeProviderPairDependencies(ctx, provider1, provider2, providerResources)

				mu.Lock()
				dependencies = append(dependencies, deps...)
				mu.Unlock()
			}(providers[i], providers[j])
		}
	}

	wg.Wait()
	return dependencies, nil
}

// analyzeProviderPairDependencies analyzes dependencies between two providers
func (poda *PerformanceOptimizedDependencyAnalyzer) analyzeProviderPairDependencies(ctx context.Context, provider1, provider2 string, providerResources map[string][]core.Resource) []Dependency {
	var dependencies []Dependency

	resources1 := providerResources[provider1]
	resources2 := providerResources[provider2]

	// Create lookup maps for efficient searching
	map1 := make(map[string]core.Resource)
	map2 := make(map[string]core.Resource)

	for _, resource := range resources1 {
		map1[resource.ID] = resource
	}
	for _, resource := range resources2 {
		map2[resource.ID] = resource
	}

	// Analyze cross-provider dependencies
	for _, resource1 := range resources1 {
		for _, resource2 := range resources2 {
			if dep := poda.findCrossProviderDependency(resource1, resource2); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		}
	}

	return dependencies
}

// findCrossProviderDependency finds dependencies between resources from different providers
func (poda *PerformanceOptimizedDependencyAnalyzer) findCrossProviderDependency(resource1, resource2 core.Resource) *Dependency {
	// Check for common patterns in cross-provider dependencies
	// This is a simplified implementation - in practice, you'd have more sophisticated logic

	// Only check for explicit references in configuration
	config1 := string(resource1.Configuration)
	if len(config1) > 0 && (strings.Contains(config1, resource2.ID) || strings.Contains(config1, resource2.ARN)) {
		return &Dependency{
			SourceID:     resource1.ID,
			TargetID:     resource2.ID,
			SourceARN:    resource1.ARN,
			TargetARN:    resource2.ARN,
			Relationship: "cross_provider_reference",
			Direction:    "outbound",
			Confidence:   0.7,
			Metadata: map[string]interface{}{
				"source_provider": resource1.Provider,
				"target_provider": resource2.Provider,
			},
		}
	}

	return nil
}

// calculateGraphStatsOptimized calculates graph statistics efficiently
func (poda *PerformanceOptimizedDependencyAnalyzer) calculateGraphStatsOptimized(resources []core.Resource, dependencies []Dependency) GraphStats {
	stats := GraphStats{
		TotalResources:    len(resources),
		TotalDependencies: len(dependencies),
	}

	if len(dependencies) == 0 {
		stats.Islands = len(resources)
		return stats
	}

	// Create dependency graph for analysis
	graph := make(map[string][]string)
	incoming := make(map[string]int)

	for _, dep := range dependencies {
		graph[dep.SourceID] = append(graph[dep.SourceID], dep.TargetID)
		incoming[dep.TargetID]++
	}

	// Calculate islands (resources with no dependencies)
	stats.Islands = 0
	for _, resource := range resources {
		if len(graph[resource.ID]) == 0 && incoming[resource.ID] == 0 {
			stats.Islands++
		}
	}

	// Calculate max depth using BFS
	stats.MaxDepth = poda.calculateMaxDepth(graph, resources)

	// Calculate cycles using DFS
	stats.Cycles = poda.calculateCycles(graph, resources)

	return stats
}

// calculateMaxDepth calculates the maximum depth of the dependency graph
func (poda *PerformanceOptimizedDependencyAnalyzer) calculateMaxDepth(graph map[string][]string, resources []core.Resource) int {
	maxDepth := 0
	visited := make(map[string]bool)

	for _, resource := range resources {
		if !visited[resource.ID] {
			depth := poda.bfsDepth(graph, resource.ID, visited)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}

// bfsDepth calculates depth using BFS
func (poda *PerformanceOptimizedDependencyAnalyzer) bfsDepth(graph map[string][]string, start string, visited map[string]bool) int {
	queue := []string{start}
	depth := 0

	for len(queue) > 0 {
		levelSize := len(queue)
		depth++

		for i := 0; i < levelSize; i++ {
			node := queue[0]
			queue = queue[1:]

			if visited[node] {
				continue
			}
			visited[node] = true

			for _, neighbor := range graph[node] {
				if !visited[neighbor] {
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return depth
}

// calculateCycles calculates the number of cycles in the dependency graph
func (poda *PerformanceOptimizedDependencyAnalyzer) calculateCycles(graph map[string][]string, resources []core.Resource) int {
	cycles := 0
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, resource := range resources {
		if !visited[resource.ID] {
			if poda.hasCycleDFS(graph, resource.ID, visited, recStack) {
				cycles++
			}
		}
	}

	return cycles
}

// hasCycleDFS detects cycles using DFS
func (poda *PerformanceOptimizedDependencyAnalyzer) hasCycleDFS(graph map[string][]string, node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if poda.hasCycleDFS(graph, neighbor, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// ClearCache clears the analysis cache
func (poda *PerformanceOptimizedDependencyAnalyzer) ClearCache() {
	poda.cacheMutex.Lock()
	defer poda.cacheMutex.Unlock()

	poda.cache = make(map[string]interface{})
}

// GetCacheStats returns cache statistics
func (poda *PerformanceOptimizedDependencyAnalyzer) GetCacheStats() map[string]interface{} {
	poda.cacheMutex.RLock()
	defer poda.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cache_size":       len(poda.cache),
		"max_workers":      poda.config.MaxWorkers,
		"batch_size":       poda.config.BatchSize,
		"parallel_enabled": poda.config.EnableParallel,
	}
}
