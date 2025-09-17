package aws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverRDSResources discovers comprehensive RDS resources
func (p *AWSProvider) discoverRDSResources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover DB instances
	resources = append(resources, p.discoverRDSInstances(ctx, config, false)...)

	// Discover DB clusters
	resources = append(resources, p.discoverRDSClusters(ctx, config)...)

	// Discover DB snapshots
	resources = append(resources, p.discoverRDSSnapshots(ctx, config)...)

	// Discover DB parameter groups
	resources = append(resources, p.discoverRDSParameterGroups(ctx, config)...)

	// Discover DB subnet groups
	resources = append(resources, p.discoverRDSSubnetGroups(ctx, config)...)

	return resources
}

// discoverRDSInstances discovers RDS instances
func (p *AWSProvider) discoverRDSInstances(ctx context.Context, config aws.Config, criticalOnly bool) []core.Resource {
	client := rds.NewFromConfig(config)
	var resources []core.Resource

	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe RDS instances: %v", err)
			continue
		}

		for _, instance := range page.DBInstances {
			// Skip if not critical and instance is not available
			if criticalOnly && instance.DBInstanceStatus != nil && *instance.DBInstanceStatus != "available" {
				continue
			}

			resource := core.Resource{
				ID:              aws.ToString(instance.DBInstanceIdentifier),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "rds",
				Type:            "db-instance",
				Name:            aws.ToString(instance.DBInstanceIdentifier),
				ARN:             aws.ToString(instance.DBInstanceArn),
				CreatedAt:       aws.ToTime(instance.InstanceCreateTime),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseRDSTags(instance)

			// Parse configuration
			configJSON, _ := json.Marshal(instance)
			resource.Configuration = configJSON

			// Set security and cost flags
			resource.Encrypted = instance.StorageEncrypted != nil && *instance.StorageEncrypted
			resource.PublicAccess = instance.PubliclyAccessible != nil && *instance.PubliclyAccessible
			resource.MonthlyCost = p.estimateRDSInstanceCost(instance)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverRDSClusters discovers RDS clusters
func (p *AWSProvider) discoverRDSClusters(ctx context.Context, config aws.Config) []core.Resource {
	client := rds.NewFromConfig(config)
	var resources []core.Resource

	paginator := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe RDS clusters: %v", err)
			continue
		}

		for _, cluster := range page.DBClusters {
			resource := core.Resource{
				ID:              aws.ToString(cluster.DBClusterIdentifier),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "rds",
				Type:            "db-cluster",
				Name:            aws.ToString(cluster.DBClusterIdentifier),
				ARN:             aws.ToString(cluster.DBClusterArn),
				CreatedAt:       aws.ToTime(cluster.ClusterCreateTime),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseRDSClusterTags(cluster)

			// Parse configuration
			configJSON, _ := json.Marshal(cluster)
			resource.Configuration = configJSON

			// Set security and cost flags
			resource.Encrypted = cluster.StorageEncrypted != nil && *cluster.StorageEncrypted
			resource.PublicAccess = cluster.PubliclyAccessible != nil && *cluster.PubliclyAccessible
			resource.MonthlyCost = p.estimateRDSClusterCost(cluster)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverRDSSnapshots discovers RDS snapshots
func (p *AWSProvider) discoverRDSSnapshots(ctx context.Context, config aws.Config) []core.Resource {
	client := rds.NewFromConfig(config)
	var resources []core.Resource

	paginator := rds.NewDescribeDBSnapshotsPaginator(client, &rds.DescribeDBSnapshotsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe RDS snapshots: %v", err)
			continue
		}

		for _, snapshot := range page.DBSnapshots {
			resource := core.Resource{
				ID:              aws.ToString(snapshot.DBSnapshotIdentifier),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "rds",
				Type:            "db-snapshot",
				Name:            aws.ToString(snapshot.DBSnapshotIdentifier),
				ARN:             aws.ToString(snapshot.DBSnapshotArn),
				CreatedAt:       aws.ToTime(snapshot.SnapshotCreateTime),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseRDSSnapshotTags(snapshot)

			// Parse configuration
			configJSON, _ := json.Marshal(snapshot)
			resource.Configuration = configJSON

			// Set security and cost flags
			resource.Encrypted = snapshot.Encrypted != nil && *snapshot.Encrypted
			resource.MonthlyCost = p.estimateRDSSnapshotCost(snapshot)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverRDSParameterGroups discovers RDS parameter groups
func (p *AWSProvider) discoverRDSParameterGroups(ctx context.Context, config aws.Config) []core.Resource {
	client := rds.NewFromConfig(config)
	var resources []core.Resource

	paginator := rds.NewDescribeDBParameterGroupsPaginator(client, &rds.DescribeDBParameterGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe RDS parameter groups: %v", err)
			continue
		}

		for _, paramGroup := range page.DBParameterGroups {
			resource := core.Resource{
				ID:              aws.ToString(paramGroup.DBParameterGroupName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "rds",
				Type:            "db-parameter-group",
				Name:            aws.ToString(paramGroup.DBParameterGroupName),
				ARN:             aws.ToString(paramGroup.DBParameterGroupArn),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = make(map[string]string)

			// Parse configuration
			configJSON, _ := json.Marshal(paramGroup)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverRDSSubnetGroups discovers RDS subnet groups
func (p *AWSProvider) discoverRDSSubnetGroups(ctx context.Context, config aws.Config) []core.Resource {
	client := rds.NewFromConfig(config)
	var resources []core.Resource

	paginator := rds.NewDescribeDBSubnetGroupsPaginator(client, &rds.DescribeDBSubnetGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe RDS subnet groups: %v", err)
			continue
		}

		for _, subnetGroup := range page.DBSubnetGroups {
			resource := core.Resource{
				ID:              aws.ToString(subnetGroup.DBSubnetGroupName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "rds",
				Type:            "db-subnet-group",
				Name:            aws.ToString(subnetGroup.DBSubnetGroupName),
				ARN:             aws.ToString(subnetGroup.DBSubnetGroupArn),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = make(map[string]string)

			// Parse configuration
			configJSON, _ := json.Marshal(subnetGroup)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// Helper methods for RDS resources
func (p *AWSProvider) parseRDSTags(instance rdsTypes.DBInstance) map[string]string {
	tags := make(map[string]string)
	for _, tag := range instance.TagList {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) estimateRDSInstanceCost(instance rdsTypes.DBInstance) float64 {
	// TODO: Implement RDS cost estimation
	// This would integrate with AWS Cost Explorer API or use pricing data
	return 0.0
}

func (p *AWSProvider) parseRDSClusterTags(cluster rdsTypes.DBCluster) map[string]string {
	tags := make(map[string]string)
	for _, tag := range cluster.TagList {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) estimateRDSClusterCost(cluster rdsTypes.DBCluster) float64 {
	// TODO: Implement RDS cluster cost estimation
	return 0.0
}

func (p *AWSProvider) parseRDSSnapshotTags(snapshot rdsTypes.DBSnapshot) map[string]string {
	tags := make(map[string]string)
	for _, tag := range snapshot.TagList {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) estimateRDSSnapshotCost(snapshot rdsTypes.DBSnapshot) float64 {
	// TODO: Implement RDS snapshot cost estimation
	return 0.0
}
