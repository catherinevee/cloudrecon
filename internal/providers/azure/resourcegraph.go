package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// AzureResourceGraphClient handles Azure Resource Graph operations
type AzureResourceGraphClient struct {
	client *armresourcegraph.Client
}

// NewAzureResourceGraphClient creates a new Azure Resource Graph client
func NewAzureResourceGraphClient() (*AzureResourceGraphClient, error) {
	// Use default Azure credential chain
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	client, err := armresourcegraph.NewClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Resource Graph client: %w", err)
	}

	return &AzureResourceGraphClient{
		client: client,
	}, nil
}

// IsResourceGraphAvailable checks if Azure Resource Graph is available
func (c *AzureResourceGraphClient) IsResourceGraphAvailable(ctx context.Context) (bool, error) {
	// Try a simple query to test availability
	query := "Resources | limit 1"

	_, err := c.client.Resources(ctx, armresourcegraph.QueryRequest{
		Query: &query,
	}, nil)

	if err != nil {
		return false, fmt.Errorf("Resource Graph not available: %w", err)
	}

	return true, nil
}

// DiscoverWithResourceGraph uses Azure Resource Graph for comprehensive discovery
func (c *AzureResourceGraphClient) DiscoverWithResourceGraph(ctx context.Context, account core.Account) ([]core.Resource, error) {
	logrus.Info("Starting Azure Resource Graph discovery", "subscription", account.ID)

	// Comprehensive KQL query for all major resource types
	query := `
		Resources
		| where type in (
			'microsoft.compute/virtualmachines',
			'microsoft.compute/disks',
			'microsoft.compute/images',
			'microsoft.storage/storageaccounts',
			'microsoft.sql/servers',
			'microsoft.sql/databases',
			'microsoft.web/sites',
			'microsoft.web/serverfarms',
			'microsoft.keyvault/vaults',
			'microsoft.network/virtualnetworks',
			'microsoft.network/networksecuritygroups',
			'microsoft.network/loadbalancers',
			'microsoft.network/publicipaddresses',
			'microsoft.containerservice/managedclusters',
			'microsoft.containerregistry/registries',
			'microsoft.logic/workflows',
			'microsoft.servicebus/namespaces',
			'microsoft.eventhub/namespaces',
			'microsoft.cdn/profiles',
			'microsoft.cache/redis',
			'microsoft.documentdb/databaseaccounts',
			'microsoft.insights/components',
			'microsoft.automation/automationaccounts',
			'microsoft.batch/batchaccounts',
			'microsoft.datafactory/factories',
			'microsoft.databricks/workspaces',
			'microsoft.machinelearningservices/workspaces',
			'microsoft.synapse/workspaces',
			'microsoft.apimanagement/service',
			'microsoft.search/searchservices',
			'microsoft.cognitiveservices/accounts',
			'microsoft.analysisservices/servers',
			'microsoft.powerbi/workspacecollections',
			'microsoft.streamanalytics/streamingjobs',
			'microsoft.hdinsight/clusters',
			'microsoft.iotcentral/iotapps',
			'microsoft.devices/iothubs',
			'microsoft.servicefabric/clusters',
			'microsoft.managedidentity/userassignedidentities',
			'microsoft.authorization/roleassignments',
			'microsoft.authorization/policyassignments',
			'microsoft.resources/resourcegroups'
		)
		| project
			id,
			name,
			type,
			location,
			resourceGroup,
			subscriptionId,
			tags,
			properties,
			kind,
			managedBy,
			sku,
			plan,
			identity,
			zones,
			etag,
			createdTime = properties.createdTime,
			changedTime = properties.changedTime,
			provisioningState = properties.provisioningState
		| order by name asc
	`

	result, err := c.client.Resources(ctx, armresourcegraph.QueryRequest{
		Query: &query,
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("Resource Graph query failed: %w", err)
	}

	var resources []core.Resource

	// Parse the results
	if result.QueryResponse.Data != nil {
		// The Data field contains the query results
		if data, ok := result.QueryResponse.Data.([]interface{}); ok {
			for _, item := range data {
				resource, err := c.parseResourceGraphResult(item, account)
				if err != nil {
					logrus.Warnf("Failed to parse Resource Graph result: %v", err)
					continue
				}
				resources = append(resources, resource)
			}
		}
	}

	logrus.Info("Azure Resource Graph discovery completed", "resources", len(resources))
	return resources, nil
}

// parseResourceGraphResult parses a Resource Graph result into a Resource
func (c *AzureResourceGraphClient) parseResourceGraphResult(item interface{}, account core.Account) (core.Resource, error) {
	// Convert interface{} to map[string]interface{}
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return core.Resource{}, fmt.Errorf("invalid result format")
	}

	// Extract basic fields
	id, _ := itemMap["id"].(string)
	name, _ := itemMap["name"].(string)
	resourceType, _ := itemMap["type"].(string)
	location, _ := itemMap["location"].(string)
	resourceGroup, _ := itemMap["resourceGroup"].(string)
	subscriptionID, _ := itemMap["subscriptionId"].(string)

	// Parse tags
	tags := make(map[string]string)
	if tagsRaw, ok := itemMap["tags"].(map[string]interface{}); ok {
		for k, v := range tagsRaw {
			if vStr, ok := v.(string); ok {
				tags[k] = vStr
			}
		}
	}

	// Parse properties
	var properties map[string]interface{}
	if propsRaw, ok := itemMap["properties"].(map[string]interface{}); ok {
		properties = propsRaw
	}

	// Parse timestamps
	var createdAt time.Time
	if createdTimeStr, ok := itemMap["createdTime"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, createdTimeStr); err == nil {
			createdAt = parsed
		}
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Extract service and type from resource type
	service, resourceTypeShort := c.extractServiceAndType(resourceType)

	// Parse configuration
	configJSON, _ := json.Marshal(itemMap)

	resource := core.Resource{
		ID:              id,
		Provider:        "azure",
		AccountID:       subscriptionID,
		Region:          location,
		Service:         service,
		Type:            resourceTypeShort,
		Name:            name,
		ARN:             id, // Azure uses full resource ID as ARN
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt, // Resource Graph doesn't provide update time
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "azure_resource_graph",
		Configuration:   configJSON,
		Tags:            tags,
	}

	// Add resource group as a tag
	if resourceGroup != "" {
		resource.Tags["ResourceGroup"] = resourceGroup
	}

	// Enrich with security and compliance information
	c.enrichResourceSecurity(&resource, properties)

	return resource, nil
}

// extractServiceAndType extracts service and type from Azure resource type
func (c *AzureResourceGraphClient) extractServiceAndType(resourceType string) (string, string) {
	// Remove 'microsoft.' prefix and split by '/'
	if len(resourceType) > 10 && resourceType[:10] == "microsoft." {
		resourceType = resourceType[10:]
	}

	parts := splitString(resourceType, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return "unknown", "unknown"
}

// enrichResourceSecurity enriches resource with security and compliance information
func (c *AzureResourceGraphClient) enrichResourceSecurity(resource *core.Resource, properties map[string]interface{}) {
	// Check for public access
	resource.PublicAccess = c.isResourcePublic(resource.Service, resource.Type, properties)

	// Check for encryption
	resource.Encrypted = c.isResourceEncrypted(resource.Service, resource.Type, properties)

	// Add compliance flags
	resource.Compliance = c.getComplianceFlags(resource.Service, resource.Type, properties)
}

// isResourcePublic checks if a resource has public access
func (c *AzureResourceGraphClient) isResourcePublic(service, resourceType string, properties map[string]interface{}) bool {
	switch service {
	case "storage":
		if resourceType == "storageaccounts" {
			// Check if storage account allows public access
			if networkRules, ok := properties["networkAcls"].(map[string]interface{}); ok {
				if defaultAction, ok := networkRules["defaultAction"].(string); ok {
					return defaultAction == "Allow"
				}
			}
		}
	case "web":
		if resourceType == "sites" {
			// Check if app service has public access
			if siteConfig, ok := properties["siteConfig"].(map[string]interface{}); ok {
				if httpsOnly, ok := siteConfig["httpsOnly"].(bool); ok {
					return !httpsOnly
				}
			}
		}
	case "network":
		if resourceType == "publicipaddresses" {
			// Public IP addresses are inherently public
			return true
		}
	}
	return false
}

// isResourceEncrypted checks if a resource is encrypted
func (c *AzureResourceGraphClient) isResourceEncrypted(service, resourceType string, properties map[string]interface{}) bool {
	switch service {
	case "storage":
		if resourceType == "storageaccounts" {
			if encryption, ok := properties["encryption"].(map[string]interface{}); ok {
				if services, ok := encryption["services"].(map[string]interface{}); ok {
					if blob, ok := services["blob"].(map[string]interface{}); ok {
						if enabled, ok := blob["enabled"].(bool); ok {
							return enabled
						}
					}
				}
			}
		}
	case "compute":
		if resourceType == "disks" {
			if encryptionSettings, ok := properties["encryptionSettings"].(map[string]interface{}); ok {
				if enabled, ok := encryptionSettings["enabled"].(bool); ok {
					return enabled
				}
			}
		}
	case "keyvault":
		if resourceType == "vaults" {
			if enabledForDiskEncryption, ok := properties["enabledForDiskEncryption"].(bool); ok {
				return enabledForDiskEncryption
			}
		}
	}
	return false
}

// getComplianceFlags returns compliance flags for the resource
func (c *AzureResourceGraphClient) getComplianceFlags(service, resourceType string, properties map[string]interface{}) []string {
	var flags []string

	// Check for required tags (this would be based on your organization's policy)
	// For now, we'll add some basic compliance checks

	// Check for public access
	if c.isResourcePublic(service, resourceType, properties) {
		flags = append(flags, "public-access")
	}

	// Check for encryption
	if !c.isResourceEncrypted(service, resourceType, properties) {
		flags = append(flags, "unencrypted")
	}

	// Check for resource group naming convention
	if resourceGroup, ok := properties["resourceGroup"].(string); ok {
		if len(resourceGroup) < 3 {
			flags = append(flags, "invalid-resource-group-name")
		}
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
