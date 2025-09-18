package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// DependencyAnalyzer handles resource dependency analysis
type DependencyAnalyzer struct {
	storage core.Storage
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(storage core.Storage) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		storage: storage,
	}
}

// Dependency represents a relationship between two resources
type Dependency struct {
	SourceID     string                 `json:"source_id"`
	TargetID     string                 `json:"target_id"`
	SourceARN    string                 `json:"source_arn"`
	TargetARN    string                 `json:"target_arn"`
	Relationship string                 `json:"relationship"`
	Direction    string                 `json:"direction"`  // "outbound", "inbound", "bidirectional"
	Confidence   float64                `json:"confidence"` // 0.0 to 1.0
	Metadata     map[string]interface{} `json:"metadata"`
}

// DependencyGraph represents the complete dependency graph
type DependencyGraph struct {
	Resources    []core.Resource `json:"resources"`
	Dependencies []Dependency    `json:"dependencies"`
	Stats        GraphStats      `json:"stats"`
}

// GraphStats provides statistics about the dependency graph
type GraphStats struct {
	TotalResources    int `json:"total_resources"`
	TotalDependencies int `json:"total_dependencies"`
	MaxDepth          int `json:"max_depth"`
	Cycles            int `json:"cycles"`
	Islands           int `json:"islands"` // Resources with no dependencies
}

// AnalyzeDependencies analyzes dependencies for all resources
func (da *DependencyAnalyzer) AnalyzeDependencies(ctx context.Context) (*DependencyGraph, error) {
	logrus.Info("Starting dependency analysis")

	// Get all resources
	resources, err := da.storage.GetResources("SELECT * FROM resources")
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	// Analyze dependencies
	dependencies := make([]Dependency, 0)

	// Group resources by provider for cross-provider analysis
	providerResources := make(map[string][]core.Resource)
	for _, resource := range resources {
		providerResources[resource.Provider] = append(providerResources[resource.Provider], resource)
	}

	// Analyze dependencies within each provider
	for provider, providerResources := range providerResources {
		logrus.Infof("Analyzing dependencies for %s resources", provider)
		providerDeps, err := da.analyzeProviderDependencies(ctx, provider, providerResources)
		if err != nil {
			logrus.Warnf("Failed to analyze dependencies for %s: %v", provider, err)
			continue
		}
		dependencies = append(dependencies, providerDeps...)
	}

	// Analyze cross-provider dependencies
	crossProviderDeps, err := da.analyzeCrossProviderDependencies(ctx, providerResources)
	if err != nil {
		logrus.Warnf("Failed to analyze cross-provider dependencies: %v", err)
	}
	dependencies = append(dependencies, crossProviderDeps...)

	// Calculate graph statistics
	stats := da.calculateGraphStats(resources, dependencies)

	graph := &DependencyGraph{
		Resources:    resources,
		Dependencies: dependencies,
		Stats:        stats,
	}

	logrus.Infof("Dependency analysis completed: %d resources, %d dependencies",
		len(resources), len(dependencies))

	return graph, nil
}

// analyzeProviderDependencies analyzes dependencies within a single provider
func (da *DependencyAnalyzer) analyzeProviderDependencies(ctx context.Context, provider string, resources []core.Resource) ([]Dependency, error) {
	var dependencies []Dependency

	switch provider {
	case "aws":
		dependencies = da.analyzeAWSDependencies(resources)
	case "azure":
		dependencies = da.analyzeAzureDependencies(resources)
	case "gcp":
		dependencies = da.analyzeGCPDependencies(resources)
	default:
		logrus.Warnf("Unknown provider for dependency analysis: %s", provider)
	}

	return dependencies, nil
}

// analyzeAWSDependencies analyzes AWS-specific dependencies
func (da *DependencyAnalyzer) analyzeAWSDependencies(resources []core.Resource) []Dependency {
	var dependencies []Dependency

	// Create resource lookup map
	resourceMap := make(map[string]core.Resource)
	for _, resource := range resources {
		resourceMap[resource.ID] = resource
	}

	for _, resource := range resources {
		switch resource.Service {
		case "ec2":
			deps := da.analyzeEC2Dependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		case "rds":
			deps := da.analyzeRDSDependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		case "lambda":
			deps := da.analyzeLambdaDependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		case "s3":
			deps := da.analyzeS3Dependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		}
	}

	return dependencies
}

// analyzeEC2Dependencies analyzes EC2 instance dependencies
func (da *DependencyAnalyzer) analyzeEC2Dependencies(instance core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	// Parse instance configuration to find dependencies
	config := string(instance.Configuration)

	// Look for security groups
	if strings.Contains(config, "SecurityGroupIds") {
		// Extract security group IDs and create dependencies
		// This is a simplified example - in practice, you'd parse the JSON properly
		deps := da.findResourcesByPattern(resourceMap, "security-group", "ec2")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     instance.ID,
				TargetID:     dep.ID,
				SourceARN:    instance.ARN,
				TargetARN:    dep.ARN,
				Relationship: "uses_security_group",
				Direction:    "outbound",
				Confidence:   0.9,
				Metadata: map[string]interface{}{
					"service": "ec2",
					"type":    "security_group",
				},
			})
		}
	}

	// Look for VPC dependencies
	if strings.Contains(config, "VpcId") {
		deps := da.findResourcesByPattern(resourceMap, "vpc", "ec2")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     instance.ID,
				TargetID:     dep.ID,
				SourceARN:    instance.ARN,
				TargetARN:    dep.ARN,
				Relationship: "runs_in_vpc",
				Direction:    "outbound",
				Confidence:   0.95,
				Metadata: map[string]interface{}{
					"service": "ec2",
					"type":    "vpc",
				},
			})
		}
	}

	// Look for IAM role dependencies
	if strings.Contains(config, "IamInstanceProfile") {
		deps := da.findResourcesByPattern(resourceMap, "instance-profile", "iam")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     instance.ID,
				TargetID:     dep.ID,
				SourceARN:    instance.ARN,
				TargetARN:    dep.ARN,
				Relationship: "assumes_role",
				Direction:    "outbound",
				Confidence:   0.8,
				Metadata: map[string]interface{}{
					"service": "iam",
					"type":    "instance_profile",
				},
			})
		}
	}

	return dependencies
}

// analyzeRDSDependencies analyzes RDS instance dependencies
func (da *DependencyAnalyzer) analyzeRDSDependencies(rds core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	config := string(rds.Configuration)

	// Look for subnet group dependencies
	if strings.Contains(config, "DBSubnetGroupName") {
		deps := da.findResourcesByPattern(resourceMap, "subnet-group", "rds")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     rds.ID,
				TargetID:     dep.ID,
				SourceARN:    rds.ARN,
				TargetARN:    dep.ARN,
				Relationship: "uses_subnet_group",
				Direction:    "outbound",
				Confidence:   0.9,
				Metadata: map[string]interface{}{
					"service": "rds",
					"type":    "subnet_group",
				},
			})
		}
	}

	// Look for parameter group dependencies
	if strings.Contains(config, "DBParameterGroupName") {
		deps := da.findResourcesByPattern(resourceMap, "parameter-group", "rds")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     rds.ID,
				TargetID:     dep.ID,
				SourceARN:    rds.ARN,
				TargetARN:    dep.ARN,
				Relationship: "uses_parameter_group",
				Direction:    "outbound",
				Confidence:   0.9,
				Metadata: map[string]interface{}{
					"service": "rds",
					"type":    "parameter_group",
				},
			})
		}
	}

	return dependencies
}

// analyzeLambdaDependencies analyzes Lambda function dependencies
func (da *DependencyAnalyzer) analyzeLambdaDependencies(lambda core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	config := string(lambda.Configuration)

	// Look for VPC dependencies
	if strings.Contains(config, "VpcConfig") {
		deps := da.findResourcesByPattern(resourceMap, "vpc", "ec2")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     lambda.ID,
				TargetID:     dep.ID,
				SourceARN:    lambda.ARN,
				TargetARN:    dep.ARN,
				Relationship: "runs_in_vpc",
				Direction:    "outbound",
				Confidence:   0.8,
				Metadata: map[string]interface{}{
					"service": "lambda",
					"type":    "vpc",
				},
			})
		}
	}

	// Look for IAM role dependencies
	if strings.Contains(config, "Role") {
		deps := da.findResourcesByPattern(resourceMap, "role", "iam")
		for _, dep := range deps {
			dependencies = append(dependencies, Dependency{
				SourceID:     lambda.ID,
				TargetID:     dep.ID,
				SourceARN:    lambda.ARN,
				TargetARN:    dep.ARN,
				Relationship: "assumes_role",
				Direction:    "outbound",
				Confidence:   0.95,
				Metadata: map[string]interface{}{
					"service": "iam",
					"type":    "role",
				},
			})
		}
	}

	return dependencies
}

// analyzeS3Dependencies analyzes S3 bucket dependencies
func (da *DependencyAnalyzer) analyzeS3Dependencies(s3 core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	// S3 buckets typically don't have many direct dependencies
	// but they might be referenced by other resources
	// This would be handled in reverse dependency analysis

	return dependencies
}

// analyzeAzureDependencies analyzes Azure-specific dependencies
func (da *DependencyAnalyzer) analyzeAzureDependencies(resources []core.Resource) []Dependency {
	var dependencies []Dependency

	// Create resource lookup map
	resourceMap := make(map[string]core.Resource)
	for _, resource := range resources {
		resourceMap[resource.ID] = resource
	}

	for _, resource := range resources {
		switch resource.Service {
		case "compute":
			deps := da.analyzeAzureComputeDependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		case "storage":
			deps := da.analyzeAzureStorageDependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		}
	}

	return dependencies
}

// analyzeAzureComputeDependencies analyzes Azure compute dependencies
func (da *DependencyAnalyzer) analyzeAzureComputeDependencies(vm core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	// Look for network interface dependencies
	deps := da.findResourcesByPattern(resourceMap, "network-interface", "network")
	for _, dep := range deps {
		dependencies = append(dependencies, Dependency{
			SourceID:     vm.ID,
			TargetID:     dep.ID,
			SourceARN:    vm.ARN,
			TargetARN:    dep.ARN,
			Relationship: "uses_network_interface",
			Direction:    "outbound",
			Confidence:   0.9,
			Metadata: map[string]interface{}{
				"service": "compute",
				"type":    "network_interface",
			},
		})
	}

	return dependencies
}

// analyzeAzureStorageDependencies analyzes Azure storage dependencies
func (da *DependencyAnalyzer) analyzeAzureStorageDependencies(storage core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	// Storage accounts typically don't have many direct dependencies
	return dependencies
}

// analyzeGCPDependencies analyzes GCP-specific dependencies
func (da *DependencyAnalyzer) analyzeGCPDependencies(resources []core.Resource) []Dependency {
	var dependencies []Dependency

	// Create resource lookup map
	resourceMap := make(map[string]core.Resource)
	for _, resource := range resources {
		resourceMap[resource.ID] = resource
	}

	for _, resource := range resources {
		switch resource.Service {
		case "compute":
			deps := da.analyzeGCPComputeDependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		case "storage":
			deps := da.analyzeGCPStorageDependencies(resource, resourceMap)
			dependencies = append(dependencies, deps...)
		}
	}

	return dependencies
}

// analyzeGCPComputeDependencies analyzes GCP compute dependencies
func (da *DependencyAnalyzer) analyzeGCPComputeDependencies(instance core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	// Look for network dependencies
	deps := da.findResourcesByPattern(resourceMap, "network", "compute")
	for _, dep := range deps {
		dependencies = append(dependencies, Dependency{
			SourceID:     instance.ID,
			TargetID:     dep.ID,
			SourceARN:    instance.ARN,
			TargetARN:    dep.ARN,
			Relationship: "runs_in_network",
			Direction:    "outbound",
			Confidence:   0.9,
			Metadata: map[string]interface{}{
				"service": "compute",
				"type":    "network",
			},
		})
	}

	return dependencies
}

// analyzeGCPStorageDependencies analyzes GCP storage dependencies
func (da *DependencyAnalyzer) analyzeGCPStorageDependencies(storage core.Resource, resourceMap map[string]core.Resource) []Dependency {
	var dependencies []Dependency

	// Storage buckets typically don't have many direct dependencies
	return dependencies
}

// analyzeCrossProviderDependencies analyzes dependencies across different cloud providers
func (da *DependencyAnalyzer) analyzeCrossProviderDependencies(ctx context.Context, providerResources map[string][]core.Resource) ([]Dependency, error) {
	var dependencies []Dependency

	// Look for common patterns that indicate cross-cloud dependencies
	// This is a simplified implementation - in practice, you'd use more sophisticated analysis

	// Example: Look for resources with similar names or tags that might be related
	allResources := make([]core.Resource, 0)
	for _, resources := range providerResources {
		allResources = append(allResources, resources...)
	}

	// Group by common tags or naming patterns
	resourceGroups := da.groupResourcesByPatterns(allResources)

	// Create dependencies within each group
	for _, group := range resourceGroups {
		if len(group) > 1 {
			// Create bidirectional dependencies between resources in the same group
			for i := 0; i < len(group); i++ {
				for j := i + 1; j < len(group); j++ {
					dependencies = append(dependencies, Dependency{
						SourceID:     group[i].ID,
						TargetID:     group[j].ID,
						SourceARN:    group[i].ARN,
						TargetARN:    group[j].ARN,
						Relationship: "cross_cloud_related",
						Direction:    "bidirectional",
						Confidence:   0.6,
						Metadata: map[string]interface{}{
							"reason": "similar_naming_or_tags",
						},
					})
				}
			}
		}
	}

	return dependencies, nil
}

// findResourcesByPattern finds resources matching a pattern
func (da *DependencyAnalyzer) findResourcesByPattern(resourceMap map[string]core.Resource, pattern, service string) []core.Resource {
	var matches []core.Resource

	for _, resource := range resourceMap {
		if resource.Service == service &&
			(strings.Contains(strings.ToLower(resource.Type), pattern) ||
				strings.Contains(strings.ToLower(resource.Name), pattern)) {
			matches = append(matches, resource)
		}
	}

	return matches
}

// groupResourcesByPatterns groups resources by common patterns
func (da *DependencyAnalyzer) groupResourcesByPatterns(resources []core.Resource) [][]core.Resource {
	groups := make(map[string][]core.Resource)

	for _, resource := range resources {
		// Group by common tags
		key := da.generateGroupKey(resource)
		groups[key] = append(groups[key], resource)
	}

	var result [][]core.Resource
	for _, group := range groups {
		if len(group) > 1 {
			result = append(result, group)
		}
	}

	return result
}

// generateGroupKey generates a key for grouping resources
func (da *DependencyAnalyzer) generateGroupKey(resource core.Resource) string {
	// Use common tags or naming patterns to group resources
	key := resource.Provider + ":" + resource.Service

	// Add common tags to the key
	if resource.Tags != nil {
		if env, ok := resource.Tags["Environment"]; ok {
			key += ":" + env
		}
		if project, ok := resource.Tags["Project"]; ok {
			key += ":" + project
		}
	}

	return key
}

// calculateGraphStats calculates statistics for the dependency graph
func (da *DependencyAnalyzer) calculateGraphStats(resources []core.Resource, dependencies []Dependency) GraphStats {
	stats := GraphStats{
		TotalResources:    len(resources),
		TotalDependencies: len(dependencies),
	}

	// Calculate max depth (simplified)
	stats.MaxDepth = da.calculateMaxDepth(resources, dependencies)

	// Calculate cycles (simplified)
	stats.Cycles = da.detectCycles(resources, dependencies)

	// Calculate islands (resources with no dependencies)
	stats.Islands = da.countIslands(resources, dependencies)

	return stats
}

// calculateMaxDepth calculates the maximum depth of the dependency graph
func (da *DependencyAnalyzer) calculateMaxDepth(resources []core.Resource, dependencies []Dependency) int {
	// This is a simplified implementation
	// In practice, you'd use a proper graph traversal algorithm
	return 3 // Placeholder
}

// detectCycles detects cycles in the dependency graph
func (da *DependencyAnalyzer) detectCycles(resources []core.Resource, dependencies []Dependency) int {
	// This is a simplified implementation
	// In practice, you'd use DFS to detect cycles
	return 0 // Placeholder
}

// countIslands counts resources with no dependencies
func (da *DependencyAnalyzer) countIslands(resources []core.Resource, dependencies []Dependency) int {
	// Create a set of resources that have dependencies
	hasDeps := make(map[string]bool)
	for _, dep := range dependencies {
		hasDeps[dep.SourceID] = true
		hasDeps[dep.TargetID] = true
	}

	// Count resources without dependencies
	islands := 0
	for _, resource := range resources {
		if !hasDeps[resource.ID] {
			islands++
		}
	}

	return islands
}

// splitString is a helper method to split string by delimiter
func (da *DependencyAnalyzer) splitString(s, delimiter string) []string {
	var result []string
	start := 0

	for i := 0; i < len(s); i++ {
		if i+len(delimiter) <= len(s) && s[i:i+len(delimiter)] == delimiter {
			result = append(result, s[start:i])
			start = i + len(delimiter)
			i += len(delimiter) - 1
		}
	}

	if start < len(s) {
		result = append(result, s[start:])
	}

	return result
}
