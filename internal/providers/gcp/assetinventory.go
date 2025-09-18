package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	asset "cloud.google.com/go/asset/apiv1"
	"cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

// GCPAssetInventoryClient handles GCP Cloud Asset Inventory operations
type GCPAssetInventoryClient struct {
	client *asset.Client
}

// NewGCPAssetInventoryClient creates a new GCP Asset Inventory client
func NewGCPAssetInventoryClient(ctx context.Context) (*GCPAssetInventoryClient, error) {
	client, err := asset.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Asset Inventory client: %w", err)
	}

	return &GCPAssetInventoryClient{
		client: client,
	}, nil
}

// IsAssetInventoryAvailable checks if Cloud Asset Inventory is available
func (c *GCPAssetInventoryClient) IsAssetInventoryAvailable(ctx context.Context, projectID string) (bool, error) {
	// Try to list assets to test availability
	req := &assetpb.ListAssetsRequest{
		Parent: fmt.Sprintf("projects/%s", projectID),
	}

	it := c.client.ListAssets(ctx, req)
	_, err := it.Next()

	if err != nil && err != iterator.Done {
		return false, fmt.Errorf("Asset Inventory not available: %w", err)
	}

	return true, nil
}

// DiscoverWithAssetInventory uses Cloud Asset Inventory for comprehensive discovery
func (c *GCPAssetInventoryClient) DiscoverWithAssetInventory(ctx context.Context, account core.Account) ([]core.Resource, error) {
	logrus.Info("Starting GCP Asset Inventory discovery", "project", account.ID)

	projectID := account.ID
	parent := fmt.Sprintf("projects/%s", projectID)

	// List all assets
	req := &assetpb.ListAssetsRequest{
		Parent: parent,
		AssetTypes: []string{
			"compute.googleapis.com/Instance",
			"compute.googleapis.com/Disk",
			"compute.googleapis.com/Network",
			"compute.googleapis.com/Subnetwork",
			"compute.googleapis.com/Image",
			"compute.googleapis.com/Snapshot",
			"storage.googleapis.com/Bucket",
			"sqladmin.googleapis.com/Instance",
			"sqladmin.googleapis.com/Database",
			"cloudresourcemanager.googleapis.com/Project",
			"iam.googleapis.com/ServiceAccount",
			"iam.googleapis.com/Role",
			"run.googleapis.com/Service",
			"container.googleapis.com/Cluster",
			"container.googleapis.com/NodePool",
			"pubsub.googleapis.com/Topic",
			"pubsub.googleapis.com/Subscription",
			"bigquery.googleapis.com/Dataset",
			"bigquery.googleapis.com/Table",
			"cloudfunctions.googleapis.com/CloudFunction",
			"appengine.googleapis.com/Application",
			"appengine.googleapis.com/Service",
			"appengine.googleapis.com/Version",
		},
	}

	var resources []core.Resource
	it := c.client.ListAssets(ctx, req)

	for {
		asset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logrus.Warnf("Failed to get asset: %v", err)
			continue
		}

		resource, err := c.parseAssetToResource(asset, projectID)
		if err != nil {
			logrus.Warnf("Failed to parse asset: %v", err)
			continue
		}

		resources = append(resources, resource)
	}

	logrus.Info("GCP Asset Inventory discovery completed", "resources", len(resources))
	return resources, nil
}

// parseAssetToResource parses a Cloud Asset to a Resource
func (c *GCPAssetInventoryClient) parseAssetToResource(asset *assetpb.Asset, projectID string) (core.Resource, error) {
	// Extract service and type from asset type
	service, resourceType := c.extractServiceAndType(asset.AssetType)

	// Parse creation time
	var createdAt time.Time
	if asset.UpdateTime != nil {
		createdAt = asset.UpdateTime.AsTime()
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Parse configuration
	configJSON, _ := json.Marshal(asset)

	// Extract name from asset name
	name := c.extractResourceName(asset)

	// Generate ARN (GCP uses full resource name as ARN)
	arn := asset.Name

	// Parse labels as tags (from resource data)
	tags := make(map[string]string)
	if asset.Resource != nil && asset.Resource.Data != nil {
		// Extract labels from resource data
		if labels, ok := asset.Resource.Data.Fields["labels"]; ok {
			labelsStruct := labels.GetStructValue()
			if labelsStruct != nil {
				for k, v := range labelsStruct.Fields {
					if v.GetStringValue() != "" {
						tags[k] = v.GetStringValue()
					}
				}
			}
		}
	}

	resource := core.Resource{
		ID:              asset.Name,
		Provider:        "gcp",
		AccountID:       projectID,
		Region:          c.extractRegionFromAsset(asset),
		Service:         service,
		Type:            resourceType,
		Name:            name,
		ARN:             arn,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt, // Asset Inventory doesn't provide update time
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "gcp_asset_inventory",
		Configuration:   configJSON,
		Tags:            tags,
	}

	// Enrich with security and compliance information
	c.enrichResourceSecurity(&resource, asset)

	return resource, nil
}

// extractServiceAndType extracts service and type from GCP asset type
func (c *GCPAssetInventoryClient) extractServiceAndType(assetType string) (string, string) {
	// Remove 'googleapis.com/' suffix and split by '/'
	if len(assetType) > 15 && assetType[len(assetType)-15:] == ".googleapis.com/" {
		assetType = assetType[:len(assetType)-15]
	}

	parts := splitString(assetType, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return "unknown", "unknown"
}

// extractResourceName extracts resource name from asset
func (c *GCPAssetInventoryClient) extractResourceName(asset *assetpb.Asset) string {
	// Try to get name from resource data
	if asset.Resource != nil && asset.Resource.Data != nil {
		if name, ok := asset.Resource.Data.Fields["name"]; ok {
			if nameStr := name.GetStringValue(); nameStr != "" {
				return nameStr
			}
		}
	}

	// Fall back to last part of asset name
	parts := splitString(asset.Name, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown"
}

// extractRegionFromAsset extracts region from asset
func (c *GCPAssetInventoryClient) extractRegionFromAsset(asset *assetpb.Asset) string {
	// Try to get region from resource data
	if asset.Resource != nil && asset.Resource.Data != nil {
		if region, ok := asset.Resource.Data.Fields["region"]; ok {
			if regionStr := region.GetStringValue(); regionStr != "" {
				return regionStr
			}
		}
		if zone, ok := asset.Resource.Data.Fields["zone"]; ok {
			if zoneStr := zone.GetStringValue(); zoneStr != "" {
				// Extract region from zone (e.g., "us-central1-a" -> "us-central1")
				parts := splitString(zoneStr, "-")
				if len(parts) >= 3 {
					return parts[0] + "-" + parts[1]
				}
			}
		}
	}

	// Fall back to global
	return "global"
}

// enrichResourceSecurity enriches resource with security and compliance information
func (c *GCPAssetInventoryClient) enrichResourceSecurity(resource *core.Resource, asset *assetpb.Asset) {
	// Check for public access
	resource.PublicAccess = c.isResourcePublic(resource.Service, resource.Type, asset)

	// Check for encryption
	resource.Encrypted = c.isResourceEncrypted(resource.Service, resource.Type, asset)

	// Add compliance flags
	resource.Compliance = c.getComplianceFlags(resource.Service, resource.Type, asset)
}

// isResourcePublic checks if a resource has public access
func (c *GCPAssetInventoryClient) isResourcePublic(service, resourceType string, asset *assetpb.Asset) bool {
	switch service {
	case "storage":
		if resourceType == "Bucket" {
			// Check if bucket allows public access via IAM policy
			if asset.IamPolicy != nil {
				for _, binding := range asset.IamPolicy.Bindings {
					if binding.Role == "roles/storage.objectViewer" || binding.Role == "roles/storage.objectAdmin" {
						for _, member := range binding.Members {
							if member == "allUsers" || member == "allAuthenticatedUsers" {
								return true
							}
						}
					}
				}
			}
		}
	case "compute":
		if resourceType == "Instance" {
			// Check if instance has external IP
			if asset.Resource != nil && asset.Resource.Data != nil {
				if networkInterfaces, ok := asset.Resource.Data.Fields["networkInterfaces"]; ok {
					niList := networkInterfaces.GetListValue()
					if niList != nil {
						for _, ni := range niList.Values {
							niStruct := ni.GetStructValue()
							if niStruct != nil {
								if accessConfigs, ok := niStruct.Fields["accessConfigs"]; ok {
									acList := accessConfigs.GetListValue()
									if acList != nil {
										for _, ac := range acList.Values {
											acStruct := ac.GetStructValue()
											if acStruct != nil {
												if natIP, ok := acStruct.Fields["natIP"]; ok {
													if natIPStr := natIP.GetStringValue(); natIPStr != "" {
														return true
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

// isResourceEncrypted checks if a resource is encrypted
func (c *GCPAssetInventoryClient) isResourceEncrypted(service, resourceType string, asset *assetpb.Asset) bool {
	switch service {
	case "storage":
		if resourceType == "Bucket" {
			// Check if bucket has default encryption
			if asset.Resource != nil && asset.Resource.Data != nil {
				if encryption, ok := asset.Resource.Data.Fields["encryption"]; ok {
					encryptionStruct := encryption.GetStructValue()
					if encryptionStruct != nil {
						if defaultKmsKeyName, ok := encryptionStruct.Fields["defaultKmsKeyName"]; ok {
							if kmsKeyName := defaultKmsKeyName.GetStringValue(); kmsKeyName != "" {
								return true
							}
						}
					}
				}
			}
		}
	case "compute":
		if resourceType == "Disk" {
			// Check if disk is encrypted
			if asset.Resource != nil && asset.Resource.Data != nil {
				if diskEncryptionKey, ok := asset.Resource.Data.Fields["diskEncryptionKey"]; ok {
					dekStruct := diskEncryptionKey.GetStructValue()
					if dekStruct != nil {
						if kmsKeyName, ok := dekStruct.Fields["kmsKeyName"]; ok {
							if kmsKeyNameStr := kmsKeyName.GetStringValue(); kmsKeyNameStr != "" {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// getComplianceFlags returns compliance flags for the resource
func (c *GCPAssetInventoryClient) getComplianceFlags(service, resourceType string, asset *assetpb.Asset) []string {
	var flags []string

	// Check for public access
	if c.isResourcePublic(service, resourceType, asset) {
		flags = append(flags, "public-access")
	}

	// Check for encryption
	if !c.isResourceEncrypted(service, resourceType, asset) {
		flags = append(flags, "unencrypted")
	}

	// Check for required labels (this would be based on your organization's policy)
	hasLabels := false
	if asset.Resource != nil && asset.Resource.Data != nil {
		if labels, ok := asset.Resource.Data.Fields["labels"]; ok {
			labelsStruct := labels.GetStructValue()
			if labelsStruct != nil {
				hasLabels = len(labelsStruct.Fields) > 0
			}
		}
	}
	if !hasLabels {
		flags = append(flags, "missing-labels")
	}

	return flags
}

// Helper function to split string by delimiter
func splitString(s, delimiter string) []string {
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
