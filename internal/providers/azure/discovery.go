package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
)

type AzureProvider struct {
}

// NewProvider creates a new Azure provider
func NewProvider(cfg core.AzureConfig) (*AzureProvider, error) {
	return &AzureProvider{}, nil
}

// Name returns the provider name
func (p *AzureProvider) Name() string {
	return "azure"
}

// DiscoverAccounts discovers all Azure subscriptions
func (p *AzureProvider) DiscoverAccounts(ctx context.Context) ([]core.Account, error) {
	var accounts []core.Account

	// For now, return a placeholder account
	account := core.Account{
		ID:       "placeholder-subscription-id",
		Provider: "azure",
		Name:     "Placeholder Subscription",
		Type:     "subscription",
		Tags: map[string]string{
			"tenant_id": "placeholder-tenant-id",
		},
	}

	accounts = append(accounts, account)
	return accounts, nil
}

// DiscoverResources discovers resources in a subscription
func (p *AzureProvider) DiscoverResources(
	ctx context.Context,
	account core.Account,
	opts core.DiscoveryOptions,
) ([]core.Resource, error) {
	// For now, return a simple implementation
	return p.discoverViaDirectAPI(ctx, account, opts)
}

// ValidateCredentials checks if credentials are valid
func (p *AzureProvider) ValidateCredentials(ctx context.Context) error {
	// For now, always return success
	return nil
}

// IsNativeToolAvailable checks if Azure Resource Graph is available
func (p *AzureProvider) IsNativeToolAvailable(ctx context.Context, account core.Account) (bool, error) {
	// For now, return true to use Resource Graph placeholder
	return true, nil
}

// DiscoverWithNativeTool uses Azure Resource Graph for discovery
func (p *AzureProvider) DiscoverWithNativeTool(ctx context.Context, account core.Account) ([]core.Resource, error) {
	return p.discoverViaResourceGraph(ctx, account)
}

// discoverViaDirectAPI falls back to direct API calls
func (p *AzureProvider) discoverViaDirectAPI(ctx context.Context, account core.Account, opts core.DiscoveryOptions) ([]core.Resource, error) {
	var resources []core.Resource

	// For now, return a simple placeholder resource
	resource := core.Resource{
		ID:              "placeholder-resource",
		Provider:        "azure",
		AccountID:       account.ID,
		Region:          "eastus",
		Service:         "compute",
		Type:            "virtual-machine",
		Name:            "Placeholder VM",
		ARN:             fmt.Sprintf("/subscriptions/%s/resourceGroups/rg-placeholder/providers/Microsoft.Compute/virtualMachines/placeholder-vm", account.ID),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "direct_api",
		Tags: map[string]string{
			"Environment": "dev",
			"Project":     "cloudrecon",
		},
	}

	resources = append(resources, resource)
	return resources, nil
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
