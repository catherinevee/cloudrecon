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

// discoverS3Objects discovers S3 objects (for deep mode)
func (p *AWSProvider) discoverS3Objects(ctx context.Context, config aws.Config, bucketName string) []core.Resource {
	client := s3.NewFromConfig(config)
	var resources []core.Resource

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list objects in bucket %s: %v", bucketName, err)
			continue
		}

		for _, object := range page.Contents {
			resource := core.Resource{
				ID:              fmt.Sprintf("%s/%s", bucketName, aws.ToString(object.Key)),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "s3",
				Type:            "object",
				Name:            aws.ToString(object.Key),
				ARN:             fmt.Sprintf("arn:aws:s3:::%s/%s", bucketName, aws.ToString(object.Key)),
				CreatedAt:       aws.ToTime(object.LastModified),
				UpdatedAt:       aws.ToTime(object.LastModified),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse configuration
			configJSON, _ := json.Marshal(object)
			resource.Configuration = configJSON

			// Set size and storage class
			resource.Tags = map[string]string{
				"size":          fmt.Sprintf("%d", aws.ToInt64(object.Size)),
				"storage_class": string(object.StorageClass),
			}

			resources = append(resources, resource)
		}
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

// discoverS3BucketPolicies discovers S3 bucket policies
func (p *AWSProvider) discoverS3BucketPolicies(ctx context.Context, config aws.Config) []core.Resource {
	client := s3.NewFromConfig(config)
	var resources []core.Resource

	// List all buckets first
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		logrus.Warnf("Failed to list S3 buckets: %v", err)
		return resources
	}

	for _, bucket := range result.Buckets {
		// Get bucket policy
		policy, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			// Bucket might not have a policy
			continue
		}

		resource := core.Resource{
			ID:              fmt.Sprintf("%s-policy", aws.ToString(bucket.Name)),
			Provider:        "aws",
			AccountID:       p.getAccountIDFromConfig(config),
			Region:          config.Region,
			Service:         "s3",
			Type:            "bucket-policy",
			Name:            fmt.Sprintf("%s Policy", aws.ToString(bucket.Name)),
			ARN:             fmt.Sprintf("arn:aws:s3:::%s", aws.ToString(bucket.Name)),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
		}

		// Parse configuration
		configJSON, _ := json.Marshal(policy)
		resource.Configuration = configJSON

		resources = append(resources, resource)
	}

	return resources
}

// discoverS3BucketACLs discovers S3 bucket ACLs
func (p *AWSProvider) discoverS3BucketACLs(ctx context.Context, config aws.Config) []core.Resource {
	client := s3.NewFromConfig(config)
	var resources []core.Resource

	// List all buckets first
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		logrus.Warnf("Failed to list S3 buckets: %v", err)
		return resources
	}

	for _, bucket := range result.Buckets {
		// Get bucket ACL
		acl, err := client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			logrus.Warnf("Failed to get ACL for bucket %s: %v", aws.ToString(bucket.Name), err)
			continue
		}

		resource := core.Resource{
			ID:              fmt.Sprintf("%s-acl", aws.ToString(bucket.Name)),
			Provider:        "aws",
			AccountID:       p.getAccountIDFromConfig(config),
			Region:          config.Region,
			Service:         "s3",
			Type:            "bucket-acl",
			Name:            fmt.Sprintf("%s ACL", aws.ToString(bucket.Name)),
			ARN:             fmt.Sprintf("arn:aws:s3:::%s", aws.ToString(bucket.Name)),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "direct_api",
		}

		// Parse configuration
		configJSON, _ := json.Marshal(acl)
		resource.Configuration = configJSON

		resources = append(resources, resource)
	}

	return resources
}
