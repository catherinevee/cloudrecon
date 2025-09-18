package gcp

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/asset/apiv1/assetpb"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

type GCPProvider struct {
	assetInventoryClient *GCPAssetInventoryClient
}

// NewProvider creates a new GCP provider
func NewProvider(cfg core.GCPConfig) (*GCPProvider, error) {
	ctx := context.Background()
	assetInventoryClient, err := NewGCPAssetInventoryClient(ctx)
	if err != nil {
		logrus.Warnf("Failed to create Asset Inventory client: %v", err)
		// Continue without Asset Inventory client
	}

	return &GCPProvider{
		assetInventoryClient: assetInventoryClient,
	}, nil
}

// Name returns the provider name
func (p *GCPProvider) Name() string {
	return "gcp"
}

// DiscoverAccounts discovers all GCP projects
func (p *GCPProvider) DiscoverAccounts(ctx context.Context) ([]core.Account, error) {
	var accounts []core.Account

	// Try Resource Manager API first
	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		logrus.Warnf("Failed to create Resource Manager client: %v", err)
		return p.discoverCurrentProject(ctx)
	}
	defer client.Close()

	// List all projects
	req := &resourcemanagerpb.ListProjectsRequest{}
	it := client.ListProjects(ctx, req)

	for {
		project, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logrus.Warnf("Failed to list projects: %v", err)
			continue
		}

		account := core.Account{
			ID:       project.ProjectId,
			Provider: "gcp",
			Name:     project.DisplayName,
			Type:     "project",
			Tags: map[string]string{
				"lifecycle_state": project.State.String(),
				"project_id":      project.ProjectId,
			},
		}

		accounts = append(accounts, account)
	}

	if len(accounts) == 0 {
		return p.discoverCurrentProject(ctx)
	}

	return accounts, nil
}

// DiscoverResources discovers resources in a project
func (p *GCPProvider) DiscoverResources(
	ctx context.Context,
	account core.Account,
	opts core.DiscoveryOptions,
) ([]core.Resource, error) {
	// Try native tools first
	if opts.UseNativeTools {
		if available, _ := p.IsNativeToolAvailable(ctx, account); available {
			return p.DiscoverWithNativeTool(ctx, account)
		}
	}

	// Fall back to direct API discovery
	return p.discoverViaDirectAPI(ctx, account, opts)
}

// ValidateCredentials checks if credentials are valid
func (p *GCPProvider) ValidateCredentials(ctx context.Context) error {
	// Try to create a Resource Manager client
	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return fmt.Errorf("invalid GCP credentials: %w", err)
	}
	defer client.Close()

	// Try to list projects
	req := &resourcemanagerpb.ListProjectsRequest{}
	it := client.ListProjects(ctx, req)
	_, err = it.Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("invalid GCP credentials: %w", err)
	}

	return nil
}

// IsNativeToolAvailable checks if Cloud Asset Inventory is available
func (p *GCPProvider) IsNativeToolAvailable(ctx context.Context, account core.Account) (bool, error) {
	if p.assetInventoryClient == nil {
		return false, fmt.Errorf("Asset Inventory client not initialized")
	}

	return p.assetInventoryClient.IsAssetInventoryAvailable(ctx, account.ID)
}

// DiscoverWithNativeTool uses Cloud Asset Inventory for discovery
func (p *GCPProvider) DiscoverWithNativeTool(ctx context.Context, account core.Account) ([]core.Resource, error) {
	if p.assetInventoryClient == nil {
		return nil, fmt.Errorf("Asset Inventory client not initialized")
	}

	return p.assetInventoryClient.DiscoverWithAssetInventory(ctx, account)
}

// discoverCurrentProject discovers the current project
func (p *GCPProvider) discoverCurrentProject(ctx context.Context) ([]core.Account, error) {
	// Try to get current project from metadata server or environment
	// For now, return a placeholder
	account := core.Account{
		ID:       "placeholder-project-id",
		Provider: "gcp",
		Name:     "Placeholder Project",
		Type:     "project",
		Tags: map[string]string{
			"project_number": "123456789",
		},
	}

	return []core.Account{account}, nil
}

// discoverViaDirectAPI falls back to direct API calls
func (p *GCPProvider) discoverViaDirectAPI(ctx context.Context, account core.Account, opts core.DiscoveryOptions) ([]core.Resource, error) {
	var resources []core.Resource

	// Get regions to scan
	regions := opts.Regions
	if len(regions) == 0 {
		regions = p.getAllRegions(ctx)
	}

	// Discover resources in parallel across regions
	type regionResult struct {
		region    string
		resources []core.Resource
		err       error
	}

	resultChan := make(chan regionResult, len(regions))

	for _, region := range regions {
		go func(r string) {
			regionalResources, err := p.discoverRegionalResources(ctx, r, account, opts)
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
			logrus.Warnf("Failed to discover resources in region %s: %v", result.region, result.err)
			continue
		}
		resources = append(resources, result.resources...)
	}

	return resources, nil
}

// discoverRegionalResources discovers resources in a specific region
func (p *GCPProvider) discoverRegionalResources(
	ctx context.Context,
	region string,
	account core.Account,
	opts core.DiscoveryOptions,
) ([]core.Resource, error) {
	var resources []core.Resource

	// Discover based on mode
	switch opts.Mode {
	case core.QuickMode:
		// Only critical resources
		resources = append(resources, p.discoverComputeResources(ctx, account.ID)...)
		resources = append(resources, p.discoverPublicStorageBuckets(ctx, region, account)...)
		resources = append(resources, p.discoverSQLInstances(ctx, region, account, true)...)

	case core.StandardMode:
		// Most resources
		resources = append(resources, p.discoverComputeResources(ctx, account.ID)...)
		resources = append(resources, p.discoverStorageResources(ctx, region, account)...)
		resources = append(resources, p.discoverSQLResources(ctx, region, account)...)
		resources = append(resources, p.discoverCloudFunctions(ctx, region, account)...)

	case core.DeepMode:
		// Everything including dependencies
		resources = append(resources, p.discoverAllResources(ctx, region, account)...)
		p.mapDependencies(ctx, resources)
	}

	return resources, nil
}

// getAllRegions returns all available GCP regions
func (p *GCPProvider) getAllRegions(ctx context.Context) []string {
	// Return common regions for now
	return []string{
		"us-central1", "us-east1", "us-west1", "europe-west1",
		"asia-east1", "asia-southeast1", "australia-southeast1",
	}
}

// discoverPublicStorageBuckets discovers public Cloud Storage buckets
func (p *GCPProvider) discoverPublicStorageBuckets(ctx context.Context, region string, account core.Account) []core.Resource {
	// TODO: Implement Cloud Storage bucket discovery
	return []core.Resource{}
}

// discoverSQLInstances discovers Cloud SQL instances
func (p *GCPProvider) discoverSQLInstances(ctx context.Context, region string, account core.Account, criticalOnly bool) []core.Resource {
	// TODO: Implement Cloud SQL instance discovery
	return []core.Resource{}
}

// discoverStorageResources discovers comprehensive Cloud Storage resources
func (p *GCPProvider) discoverStorageResources(ctx context.Context, region string, account core.Account) []core.Resource {
	// TODO: Implement comprehensive Cloud Storage discovery
	return []core.Resource{}
}

// discoverSQLResources discovers comprehensive Cloud SQL resources
func (p *GCPProvider) discoverSQLResources(ctx context.Context, region string, account core.Account) []core.Resource {
	// TODO: Implement comprehensive Cloud SQL discovery
	return []core.Resource{}
}

// discoverCloudFunctions discovers Cloud Functions
func (p *GCPProvider) discoverCloudFunctions(ctx context.Context, region string, account core.Account) []core.Resource {
	// TODO: Implement Cloud Functions discovery
	return []core.Resource{}
}

// discoverAllResources discovers all resources
func (p *GCPProvider) discoverAllResources(ctx context.Context, region string, account core.Account) []core.Resource {
	var resources []core.Resource

	// Discover all resource types
	resources = append(resources, p.discoverComputeResources(ctx, account.ID)...)
	resources = append(resources, p.discoverAssetInventoryResources(ctx, account.ID)...)

	return resources
}

// discoverAssetInventoryResources discovers resources using Cloud Asset Inventory
func (p *GCPProvider) discoverAssetInventoryResources(ctx context.Context, projectID string) []core.Resource {
	// TODO: Implement Cloud Asset Inventory integration
	// For now, return placeholder resources
	var resources []core.Resource

	placeholderResources := []struct {
		name         string
		service      string
		resourceType string
	}{
		{"Cloud Storage Bucket", "storage", "bucket"},
		{"Cloud SQL Instance", "sql", "instance"},
		{"Cloud Function", "functions", "function"},
		{"Service Account", "iam", "serviceAccount"},
	}

	for _, res := range placeholderResources {
		resource := core.Resource{
			ID:              fmt.Sprintf("placeholder-%s", res.name),
			Provider:        "gcp",
			AccountID:       projectID,
			Region:          "us-central1",
			Service:         res.service,
			Type:            res.resourceType,
			Name:            res.name,
			ARN:             fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s/%s/placeholder", projectID, res.resourceType),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "cloud_asset_inventory",
			Tags: map[string]string{
				"Environment": "dev",
				"Project":     "cloudrecon",
			},
		}
		resources = append(resources, resource)
	}

	return resources
}

// Helper methods for resource discovery

// parseAssetToResource converts a Cloud Asset to a Resource
//
//nolint:unused
func (p *GCPProvider) parseAssetToResource(asset *assetpb.Asset, projectID string) core.Resource {
	resource := core.Resource{
		ID:              asset.Name,
		Provider:        "gcp",
		AccountID:       projectID,
		Region:          p.extractRegionFromAsset(asset),
		Service:         p.extractServiceFromAsset(asset),
		Type:            p.extractTypeFromAsset(asset),
		Name:            p.extractNameFromAsset(asset),
		ARN:             asset.Name,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "cloud_asset_inventory",
	}

	// Parse labels
	resource.Tags = make(map[string]string)
	// TODO: Parse asset labels when available

	// Parse configuration
	resource.Configuration = []byte(asset.AssetType)

	return resource
}

// Helper methods for parsing assets
//
//nolint:unused
func (p *GCPProvider) extractRegionFromAsset(asset *assetpb.Asset) string {
	// Extract region from asset name
	// Format: projects/{project}/locations/{location}/...
	// TODO: Implement proper region extraction
	return "global"
}

//nolint:unused
func (p *GCPProvider) extractServiceFromAsset(asset *assetpb.Asset) string {
	// Extract service from asset type
	// Format: {service}.googleapis.com/{resource}
	// TODO: Implement proper service extraction
	return "unknown"
}

//nolint:unused
func (p *GCPProvider) extractTypeFromAsset(asset *assetpb.Asset) string {
	// Extract type from asset type
	// Format: {service}.googleapis.com/{resource}
	// TODO: Implement proper type extraction
	return "unknown"
}

//nolint:unused
func (p *GCPProvider) extractNameFromAsset(asset *assetpb.Asset) string {
	// Extract name from asset name
	// TODO: Implement proper name extraction
	return asset.Name
}

func (p *GCPProvider) mapDependencies(ctx context.Context, resources []core.Resource) {
	// TODO: Implement dependency mapping
}
