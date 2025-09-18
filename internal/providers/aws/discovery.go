package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

type AWSProvider struct {
	config       aws.Config
	credentials  core.Credentials //nolint:unused
	cache        core.Cache
	configClient *AWSConfigClient
}

// NewProvider creates a new AWS provider
func NewProvider(cfg core.AWSConfig) (*AWSProvider, error) {
	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSProvider{
		config:       awsConfig,
		cache:        &memoryCache{},
		configClient: NewAWSConfigClient(awsConfig),
	}, nil
}

// Name returns the provider name
func (p *AWSProvider) Name() string {
	return "aws"
}

// DiscoverAccounts discovers all AWS accounts
func (p *AWSProvider) DiscoverAccounts(ctx context.Context) ([]core.Account, error) {
	var accounts []core.Account

	// Try Organizations API first
	orgClient := organizations.NewFromConfig(p.config)

	// Check if we have organizations access
	_, err := orgClient.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err == nil {
		// We have organizations access, list all accounts
		accounts, err = p.discoverAccountsViaOrganizations(ctx, orgClient)
		if err != nil {
			logrus.Warnf("Failed to discover accounts via organizations: %v", err)
		}
	}

	// Always try to get current account
	currentAccount, err := p.discoverCurrentAccount(ctx)
	if err != nil {
		logrus.Warnf("Failed to discover current account: %v", err)
	} else {
		// Add current account if not already present
		found := false
		for _, account := range accounts {
			if account.ID == currentAccount.ID {
				found = true
				break
			}
		}
		if !found {
			accounts = append(accounts, currentAccount)
		}
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no AWS accounts discovered")
	}

	return accounts, nil
}

// DiscoverResources discovers resources in an account
func (p *AWSProvider) DiscoverResources(
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
func (p *AWSProvider) ValidateCredentials(ctx context.Context) error {
	// Try to get caller identity
	stsClient := sts.NewFromConfig(p.config)
	_, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("invalid AWS credentials: %w", err)
	}
	return nil
}

// IsNativeToolAvailable checks if AWS Config is available
func (p *AWSProvider) IsNativeToolAvailable(ctx context.Context, account core.Account) (bool, error) {
	return p.configClient.IsConfigAvailable(ctx)
}

// DiscoverWithNativeTool uses AWS Config for discovery
func (p *AWSProvider) DiscoverWithNativeTool(ctx context.Context, account core.Account) ([]core.Resource, error) {
	return p.configClient.DiscoverWithConfig(ctx, account)
}

// discoverAccountsViaOrganizations discovers accounts via Organizations API
func (p *AWSProvider) discoverAccountsViaOrganizations(ctx context.Context, client *organizations.Client) ([]core.Account, error) {
	var accounts []core.Account

	paginator := organizations.NewListAccountsPaginator(client, &organizations.ListAccountsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, account := range page.Accounts {
			accounts = append(accounts, core.Account{
				ID:       aws.ToString(account.Id),
				Provider: "aws",
				Name:     aws.ToString(account.Name),
				Type:     "account",
				Tags: map[string]string{
					"email":  aws.ToString(account.Email),
					"status": string(account.Status),
				},
			})
		}
	}

	return accounts, nil
}

// discoverCurrentAccount discovers the current account
func (p *AWSProvider) discoverCurrentAccount(ctx context.Context) (core.Account, error) {
	stsClient := sts.NewFromConfig(p.config)

	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return core.Account{}, fmt.Errorf("failed to get caller identity: %w", err)
	}

	return core.Account{
		ID:       aws.ToString(identity.Account),
		Provider: "aws",
		Name:     "Current Account",
		Type:     "account",
		Tags: map[string]string{
			"user_id": aws.ToString(identity.UserId),
			"arn":     aws.ToString(identity.Arn),
		},
	}, nil
}

// discoverViaDirectAPI falls back to direct API calls
func (p *AWSProvider) discoverViaDirectAPI(ctx context.Context, account core.Account, opts core.DiscoveryOptions) ([]core.Resource, error) {
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
			regionalResources, err := p.discoverRegionalResources(ctx, r, opts)
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
func (p *AWSProvider) discoverRegionalResources(
	ctx context.Context,
	region string,
	opts core.DiscoveryOptions,
) ([]core.Resource, error) {
	// Configure regional client
	regionalConfig := p.config.Copy()
	regionalConfig.Region = region

	var resources []core.Resource

	// Discover based on mode
	switch opts.Mode {
	case core.QuickMode:
		// Only critical resources
		resources = append(resources, p.discoverEC2Instances(ctx, regionalConfig, true)...)
		resources = append(resources, p.discoverPublicS3Buckets(ctx, regionalConfig)...)
		resources = append(resources, p.discoverRDSInstances(ctx, regionalConfig, true)...)

	case core.StandardMode:
		// Most resources
		resources = append(resources, p.discoverEC2Resources(ctx, regionalConfig)...)
		resources = append(resources, p.discoverS3Resources(ctx, regionalConfig)...)
		resources = append(resources, p.discoverRDSResources(ctx, regionalConfig)...)

	case core.DeepMode:
		// Everything including dependencies
		resources = append(resources, p.discoverAllResources(ctx, regionalConfig)...)
		p.mapDependencies(ctx, resources)
	}

	return resources, nil
}

// getAllRegions returns all available AWS regions
func (p *AWSProvider) getAllRegions(ctx context.Context) []string {
	// Return common regions for now
	return []string{
		"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		"us-east-2", "us-west-1", "eu-central-1", "ap-northeast-1",
	}
}

// discoverEC2Resources discovers comprehensive EC2 resources
func (p *AWSProvider) discoverEC2Resources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover instances
	resources = append(resources, p.discoverEC2Instances(ctx, config, false)...)

	// Discover security groups
	resources = append(resources, p.discoverSecurityGroups(ctx, config)...)

	// Discover volumes
	resources = append(resources, p.discoverVolumes(ctx, config)...)

	// Discover VPCs
	resources = append(resources, p.discoverVPCs(ctx, config)...)

	// Discover subnets
	resources = append(resources, p.discoverSubnets(ctx, config)...)

	return resources
}

// discoverAllResources discovers all resources
func (p *AWSProvider) discoverAllResources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover all resource types
	resources = append(resources, p.discoverEC2Resources(ctx, config)...)
	resources = append(resources, p.discoverS3Resources(ctx, config)...)
	resources = append(resources, p.discoverRDSResources(ctx, config)...)
	resources = append(resources, p.discoverIAMResources(ctx, config)...)
	resources = append(resources, p.discoverLambdaResources(ctx, config)...)

	// Service-specific discoveries
	if cfResources, err := p.DiscoverCloudFormationStacks(ctx, config.Region); err == nil {
		resources = append(resources, cfResources...)
	}
	if ecsResources, err := p.DiscoverECSServices(ctx, config.Region); err == nil {
		resources = append(resources, ecsResources...)
	}
	if elasticacheResources, err := p.DiscoverElastiCacheClusters(ctx, config.Region); err == nil {
		resources = append(resources, elasticacheResources...)
	}
	if lbResources, err := p.DiscoverLoadBalancers(ctx, config.Region); err == nil {
		resources = append(resources, lbResources...)
	}
	if snsResources, err := p.DiscoverSNSTopics(ctx, config.Region); err == nil {
		resources = append(resources, snsResources...)
	}
	if sqsResources, err := p.DiscoverSQSQueues(ctx, config.Region); err == nil {
		resources = append(resources, sqsResources...)
	}

	// Global services (Route53)
	if route53Resources, err := p.DiscoverRoute53Zones(ctx, config.Region); err == nil {
		resources = append(resources, route53Resources...)
	}

	return resources
}

// Helper methods
func (p *AWSProvider) getAccountIDFromConfig(config aws.Config) string {
	// This is a placeholder - in a real implementation, you'd get this from STS
	return "placeholder-account-id"
}

func (p *AWSProvider) mapDependencies(ctx context.Context, resources []core.Resource) {
	// TODO: Implement dependency mapping
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
