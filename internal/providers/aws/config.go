package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// AWSConfigClient handles AWS Config operations
type AWSConfigClient struct {
	client *configservice.Client
	region string
}

// NewAWSConfigClient creates a new AWS Config client
func NewAWSConfigClient(cfg aws.Config) *AWSConfigClient {
	return &AWSConfigClient{
		client: configservice.NewFromConfig(cfg),
		region: cfg.Region,
	}
}

// IsConfigAvailable checks if AWS Config is enabled and available
func (c *AWSConfigClient) IsConfigAvailable(ctx context.Context) (bool, error) {
	// Check for Configuration Recorders
	recorders, err := c.client.DescribeConfigurationRecorders(ctx, &configservice.DescribeConfigurationRecordersInput{})
	if err != nil {
		return false, fmt.Errorf("failed to check configuration recorders: %w", err)
	}

	if len(recorders.ConfigurationRecorders) == 0 {
		return false, nil
	}

	// Check if at least one recorder is recording
	for _, recorder := range recorders.ConfigurationRecorders {
		if recorder.RecordingGroup != nil && recorder.RecordingGroup.AllSupported {
			return true, nil
		}
	}

	return false, nil
}

// IsConfigAggregatorAvailable checks if Config Aggregator is available
func (c *AWSConfigClient) IsConfigAggregatorAvailable(ctx context.Context) (bool, error) {
	aggregators, err := c.client.DescribeConfigurationAggregators(ctx, &configservice.DescribeConfigurationAggregatorsInput{})
	if err != nil {
		return false, fmt.Errorf("failed to check configuration aggregators: %w", err)
	}

	return len(aggregators.ConfigurationAggregators) > 0, nil
}

// DiscoverWithConfig uses AWS Config for comprehensive resource discovery
func (c *AWSConfigClient) DiscoverWithConfig(ctx context.Context, account core.Account) ([]core.Resource, error) {
	logrus.Info("Starting AWS Config discovery", "account", account.ID, "region", c.region)

	// Try Config Aggregator first (fastest for multi-account)
	if available, err := c.IsConfigAggregatorAvailable(ctx); err == nil && available {
		return c.discoverViaConfigAggregator(ctx, account)
	}

	// Fall back to regular Config
	return c.discoverViaConfig(ctx, account)
}

// discoverViaConfigAggregator uses Config Aggregator for multi-account discovery
func (c *AWSConfigClient) discoverViaConfigAggregator(ctx context.Context, account core.Account) ([]core.Resource, error) {
	logrus.Info("Using AWS Config Aggregator for discovery")

	// Get the first available aggregator
	aggregators, err := c.client.DescribeConfigurationAggregators(ctx, &configservice.DescribeConfigurationAggregatorsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration aggregators: %w", err)
	}

	if len(aggregators.ConfigurationAggregators) == 0 {
		return nil, fmt.Errorf("no configuration aggregators found")
	}

	aggregator := aggregators.ConfigurationAggregators[0]
	logrus.Info("Using Config Aggregator", "name", aws.ToString(aggregator.ConfigurationAggregatorName))

	// Query all resources using the aggregator
	query := `
		SELECT
			resourceId,
			resourceType,
			awsRegion,
			configuration,
			tags,
			resourceCreationTime,
			resourceDeletionTime
		WHERE
			resourceType IN (
				'AWS::EC2::Instance',
				'AWS::EC2::SecurityGroup',
				AWS::EC2::VPC',
				'AWS::EC2::Subnet',
				'AWS::EC2::Volume',
				'AWS::S3::Bucket',
				'AWS::RDS::DBInstance',
				'AWS::RDS::DBCluster',
				'AWS::IAM::User',
				'AWS::IAM::Role',
				'AWS::IAM::Group',
				'AWS::Lambda::Function',
				'AWS::CloudFormation::Stack',
				'AWS::ECS::Cluster',
				'AWS::ECS::Service'
			)
	`

	var resources []core.Resource
	var nextToken *string

	for {
		result, err := c.client.SelectAggregateResourceConfig(ctx, &configservice.SelectAggregateResourceConfigInput{
			ConfigurationAggregatorName: aggregator.ConfigurationAggregatorName,
			Expression:                  aws.String(query),
			NextToken:                   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("config aggregator query failed: %w", err)
		}

		// Parse results
		for _, item := range result.Results {
			resource, err := c.parseConfigResult(item, account)
			if err != nil {
				logrus.Warnf("Failed to parse config result: %v", err)
				continue
			}
			resources = append(resources, resource)
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	logrus.Info("AWS Config Aggregator discovery completed", "resources", len(resources))
	return resources, nil
}

// discoverViaConfig uses regular AWS Config for discovery
func (c *AWSConfigClient) discoverViaConfig(ctx context.Context, account core.Account) ([]core.Resource, error) {
	logrus.Info("Using AWS Config for discovery")

	// Query all resources
	query := `
		SELECT
			resourceId,
			resourceType,
			awsRegion,
			configuration,
			tags,
			resourceCreationTime,
			resourceDeletionTime
		WHERE
			resourceType IN (
				'AWS::EC2::Instance',
				'AWS::EC2::SecurityGroup',
				'AWS::EC2::VPC',
				'AWS::EC2::Subnet',
				'AWS::EC2::Volume',
				'AWS::S3::Bucket',
				'AWS::RDS::DBInstance',
				'AWS::RDS::DBCluster',
				'AWS::IAM::User',
				'AWS::IAM::Role',
				'AWS::IAM::Group',
				'AWS::Lambda::Function',
				'AWS::CloudFormation::Stack',
				'AWS::ECS::Cluster',
				'AWS::ECS::Service'
			)
	`

	var resources []core.Resource
	var nextToken *string

	for {
		result, err := c.client.SelectResourceConfig(ctx, &configservice.SelectResourceConfigInput{
			Expression: aws.String(query),
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("config query failed: %w", err)
		}

		// Parse results
		for _, item := range result.Results {
			resource, err := c.parseConfigResult(item, account)
			if err != nil {
				logrus.Warnf("Failed to parse config result: %v", err)
				continue
			}
			resources = append(resources, resource)
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	logrus.Info("AWS Config discovery completed", "resources", len(resources))
	return resources, nil
}

// parseConfigResult parses a Config API result into a Resource
func (c *AWSConfigClient) parseConfigResult(item string, account core.Account) (core.Resource, error) {
	var configData struct {
		ResourceID           string                 `json:"resourceId"`
		ResourceType         string                 `json:"resourceType"`
		AWSRegion            string                 `json:"awsRegion"`
		Configuration        map[string]interface{} `json:"configuration"`
		Tags                 map[string]string      `json:"tags"`
		ResourceCreationTime *string                `json:"resourceCreationTime"`
		ResourceDeletionTime *string                `json:"resourceDeletionTime"`
	}

	if err := json.Unmarshal([]byte(item), &configData); err != nil {
		return core.Resource{}, fmt.Errorf("failed to unmarshal config result: %w", err)
	}

	// Parse creation time
	var createdAt time.Time
	if configData.ResourceCreationTime != nil {
		if parsed, err := time.Parse(time.RFC3339, *configData.ResourceCreationTime); err == nil {
			createdAt = parsed
		}
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Parse configuration
	configJSON, _ := json.Marshal(configData.Configuration)

	// Extract service and type from resource type
	service, resourceType := c.extractServiceAndType(configData.ResourceType)

	// Generate ARN
	arn := c.generateARN(configData.ResourceType, configData.ResourceID, configData.AWSRegion, account.ID)

	// Note: Resource deletion time is available but not used in current implementation

	resource := core.Resource{
		ID:              configData.ResourceID,
		Provider:        "aws",
		AccountID:       account.ID,
		Region:          configData.AWSRegion,
		Service:         service,
		Type:            resourceType,
		Name:            c.extractResourceName(configData.Configuration),
		ARN:             arn,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt, // Config doesn't provide update time
		DiscoveredAt:    time.Now(),
		DiscoveryMethod: "aws_config",
		Configuration:   configJSON,
		Tags:            configData.Tags,
	}

	// Set security and compliance flags
	c.enrichResourceSecurity(&resource, configData.Configuration)

	return resource, nil
}

// extractServiceAndType extracts service and type from AWS resource type
func (c *AWSConfigClient) extractServiceAndType(resourceType string) (string, string) {
	switch resourceType {
	case "AWS::EC2::Instance":
		return "ec2", "instance"
	case "AWS::EC2::SecurityGroup":
		return "ec2", "security-group"
	case "AWS::EC2::VPC":
		return "ec2", "vpc"
	case "AWS::EC2::Subnet":
		return "ec2", "subnet"
	case "AWS::EC2::Volume":
		return "ec2", "volume"
	case "AWS::S3::Bucket":
		return "s3", "bucket"
	case "AWS::RDS::DBInstance":
		return "rds", "instance"
	case "AWS::RDS::DBCluster":
		return "rds", "cluster"
	case "AWS::IAM::User":
		return "iam", "user"
	case "AWS::IAM::Role":
		return "iam", "role"
	case "AWS::IAM::Group":
		return "iam", "group"
	case "AWS::Lambda::Function":
		return "lambda", "function"
	case "AWS::CloudFormation::Stack":
		return "cloudformation", "stack"
	case "AWS::ECS::Cluster":
		return "ecs", "cluster"
	case "AWS::ECS::Service":
		return "ecs", "service"
	default:
		// Generic parsing for unknown types
		parts := splitResourceType(resourceType)
		if len(parts) >= 3 {
			return parts[1], parts[2]
		}
		return "unknown", "unknown"
	}
}

// extractResourceName extracts resource name from configuration
func (c *AWSConfigClient) extractResourceName(config map[string]interface{}) string {
	// Try common name fields
	nameFields := []string{"Name", "name", "InstanceName", "BucketName", "UserName", "RoleName"}

	for _, field := range nameFields {
		if name, ok := config[field].(string); ok && name != "" {
			return name
		}
	}

	// Try tags
	if tags, ok := config["Tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagMap, ok := tag.(map[string]interface{}); ok {
				if key, ok := tagMap["Key"].(string); ok && key == "Name" {
					if value, ok := tagMap["Value"].(string); ok {
						return value
					}
				}
			}
		}
	}

	return "unknown"
}

// generateARN generates ARN for the resource
func (c *AWSConfigClient) generateARN(resourceType, resourceID, region, accountID string) string {
	service, resourceType := c.extractServiceAndType(resourceType)

	switch service {
	case "ec2":
		return fmt.Sprintf("arn:aws:ec2:%s:%s:%s/%s", region, accountID, resourceType, resourceID)
	case "s3":
		return fmt.Sprintf("arn:aws:s3:::%s", resourceID)
	case "rds":
		return fmt.Sprintf("arn:aws:rds:%s:%s:%s:%s", region, accountID, resourceType, resourceID)
	case "iam":
		return fmt.Sprintf("arn:aws:iam::%s:%s/%s", accountID, resourceType, resourceID)
	case "lambda":
		return fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", region, accountID, resourceID)
	case "cloudformation":
		return fmt.Sprintf("arn:aws:cloudformation:%s:%s:stack/%s", region, accountID, resourceID)
	case "ecs":
		return fmt.Sprintf("arn:aws:ecs:%s:%s:%s/%s", region, accountID, resourceType, resourceID)
	default:
		return fmt.Sprintf("arn:aws:%s:%s:%s:%s/%s", service, region, accountID, resourceType, resourceID)
	}
}

// enrichResourceSecurity enriches resource with security and compliance information
func (c *AWSConfigClient) enrichResourceSecurity(resource *core.Resource, config map[string]interface{}) {
	// Check for public access
	resource.PublicAccess = c.isResourcePublic(resource.Service, resource.Type, config)

	// Check for encryption
	resource.Encrypted = c.isResourceEncrypted(resource.Service, resource.Type, config)

	// Add compliance flags
	resource.Compliance = c.getComplianceFlags(resource.Service, resource.Type, config)
}

// isResourcePublic checks if a resource has public access
func (c *AWSConfigClient) isResourcePublic(service, resourceType string, config map[string]interface{}) bool {
	switch service {
	case "s3":
		if resourceType == "bucket" {
			// Check bucket policy for public access
			if policy, ok := config["Policy"].(string); ok {
				return c.isS3PolicyPublic(policy)
			}
		}
	case "ec2":
		if resourceType == "security-group" {
			// Check security group rules for 0.0.0.0/0
			if rules, ok := config["SecurityGroupRules"].([]interface{}); ok {
				for _, rule := range rules {
					if ruleMap, ok := rule.(map[string]interface{}); ok {
						if cidr, ok := ruleMap["CidrIpv4"].(string); ok {
							if cidr == "0.0.0.0/0" {
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

// isResourceEncrypted checks if a resource is encrypted
func (c *AWSConfigClient) isResourceEncrypted(service, resourceType string, config map[string]interface{}) bool {
	switch service {
	case "s3":
		if resourceType == "bucket" {
			if encryption, ok := config["ServerSideEncryptionConfiguration"].(map[string]interface{}); ok {
				return encryption != nil
			}
		}
	case "rds":
		if resourceType == "instance" || resourceType == "cluster" {
			if encrypted, ok := config["StorageEncrypted"].(bool); ok {
				return encrypted
			}
		}
	case "ec2":
		if resourceType == "volume" {
			if encrypted, ok := config["Encrypted"].(bool); ok {
				return encrypted
			}
		}
	}
	return false
}

// getComplianceFlags returns compliance flags for the resource
func (c *AWSConfigClient) getComplianceFlags(service, resourceType string, config map[string]interface{}) []string {
	var flags []string

	// Check for required tags
	if tags, ok := config["Tags"].([]interface{}); ok {
		hasName := false
		hasEnvironment := false

		for _, tag := range tags {
			if tagMap, ok := tag.(map[string]interface{}); ok {
				if key, ok := tagMap["Key"].(string); ok {
					switch key {
					case "Name":
						hasName = true
					case "Environment", "env":
						hasEnvironment = true
					}
				}
			}
		}

		if !hasName {
			flags = append(flags, "missing-name-tag")
		}
		if !hasEnvironment {
			flags = append(flags, "missing-environment-tag")
		}
	}

	// Check for public access
	if c.isResourcePublic(service, resourceType, config) {
		flags = append(flags, "public-access")
	}

	// Check for encryption
	if !c.isResourceEncrypted(service, resourceType, config) {
		flags = append(flags, "unencrypted")
	}

	return flags
}

// isS3PolicyPublic checks if S3 bucket policy allows public access
func (c *AWSConfigClient) isS3PolicyPublic(policy string) bool {
	// Simple check for public access patterns
	publicPatterns := []string{
		"\"Principal\": \"*\"",
		"\"Principal\": {\"AWS\": \"*\"}",
		"\"Effect\": \"Allow\"",
	}

	for _, pattern := range publicPatterns {
		if contains(policy, pattern) {
			return true
		}
	}

	return false
}

// Helper functions
func splitResourceType(resourceType string) []string {
	// Split "AWS::Service::Type" into ["AWS", "Service", "Type"]
	parts := make([]string, 0)
	current := ""

	for _, char := range resourceType {
		if char == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
