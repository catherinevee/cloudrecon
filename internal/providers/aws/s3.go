package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverS3Resources discovers comprehensive S3 resources
func (p *AWSProvider) discoverS3Resources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover all buckets (not just public ones)
	client := s3.NewFromConfig(config)

	// List buckets directly
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		logrus.Warnf("Failed to list S3 buckets: %v", err)
		return resources
	}

	for _, bucket := range result.Buckets {
		resource := core.Resource{
			ID:              aws.ToString(bucket.Name),
			Provider:        "aws",
			AccountID:       p.getAccountIDFromConfig(config),
			Region:          config.Region,
			Service:         "s3",
			Type:            "bucket",
			Name:            aws.ToString(bucket.Name),
			ARN:             fmt.Sprintf("arn:aws:s3:::%s", aws.ToString(bucket.Name)),
			CreatedAt:       aws.ToTime(bucket.CreationDate),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
		}

		// Check if bucket is public
		isPublic, _ := p.isBucketPublic(ctx, client, aws.ToString(bucket.Name))
		resource.PublicAccess = isPublic

		// Check if bucket is encrypted
		resource.Encrypted = p.isBucketEncrypted(ctx, client, aws.ToString(bucket.Name))

		// Parse configuration
		configJSON, _ := json.Marshal(bucket)
		resource.Configuration = configJSON

		// Estimate cost
		resource.MonthlyCost = p.estimateBucketCost(ctx, client, aws.ToString(bucket.Name))

		resources = append(resources, resource)
	}

	return resources
}

// discoverPublicS3Buckets discovers public S3 buckets
func (p *AWSProvider) discoverPublicS3Buckets(ctx context.Context, config aws.Config) []core.Resource {
	client := s3.NewFromConfig(config)
	var resources []core.Resource

	// List buckets directly
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		logrus.Warnf("Failed to list S3 buckets: %v", err)
		return resources
	}

	for _, bucket := range result.Buckets {
		// Check if bucket is public
		isPublic, err := p.isBucketPublic(ctx, client, aws.ToString(bucket.Name))
		if err != nil {
			logrus.Warnf("Failed to check bucket public access: %v", err)
			continue
		}

		if !isPublic {
			continue
		}

		resource := core.Resource{
			ID:              aws.ToString(bucket.Name),
			Provider:        "aws",
			AccountID:       p.getAccountIDFromConfig(config),
			Region:          config.Region,
			Service:         "s3",
			Type:            "bucket",
			Name:            aws.ToString(bucket.Name),
			ARN:             fmt.Sprintf("arn:aws:s3:::%s", aws.ToString(bucket.Name)),
			CreatedAt:       aws.ToTime(bucket.CreationDate),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
			PublicAccess:    true,
		}

		// Parse configuration
		configJSON, _ := json.Marshal(bucket)
		resource.Configuration = configJSON

		resources = append(resources, resource)
	}

	return resources
}

// Helper methods for S3 resources
func (p *AWSProvider) isBucketPublic(ctx context.Context, client *s3.Client, bucketName string) (bool, error) {
	// Check bucket public access block
	result, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// If public access block is not configured, bucket might be public
		return true, nil
	}

	// Check if all public access is blocked
	return !(aws.ToBool(result.PublicAccessBlockConfiguration.BlockPublicAcls) &&
		aws.ToBool(result.PublicAccessBlockConfiguration.BlockPublicPolicy) &&
		aws.ToBool(result.PublicAccessBlockConfiguration.IgnorePublicAcls) &&
		aws.ToBool(result.PublicAccessBlockConfiguration.RestrictPublicBuckets)), nil
}

func (p *AWSProvider) isBucketEncrypted(ctx context.Context, client *s3.Client, bucketName string) bool {
	// Check bucket encryption configuration
	result, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// If encryption is not configured, bucket is not encrypted
		return false
	}

	// Check if any encryption rule is configured
	return len(result.ServerSideEncryptionConfiguration.Rules) > 0
}

func (p *AWSProvider) estimateBucketCost(ctx context.Context, client *s3.Client, bucketName string) float64 {
	// TODO: Implement bucket cost estimation
	// This would integrate with AWS Cost Explorer API or use pricing data
	return 0.0
}
