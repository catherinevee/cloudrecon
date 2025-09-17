package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverEC2Instances discovers EC2 instances
func (p *AWSProvider) discoverEC2Instances(ctx context.Context, config aws.Config, criticalOnly bool) []core.Resource {
	client := ec2.NewFromConfig(config)
	var resources []core.Resource

	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe EC2 instances: %v", err)
			continue
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				// Skip terminated instances unless in deep mode
				if instance.State != nil && instance.State.Name == ec2Types.InstanceStateNameTerminated && criticalOnly {
					continue
				}

				resource := core.Resource{
					ID:              aws.ToString(instance.InstanceId),
					Provider:        "aws",
					AccountID:       p.getAccountIDFromConfig(config),
					Region:          config.Region,
					Service:         "ec2",
					Type:            "instance",
					Name:            p.getInstanceName(instance),
					ARN:             p.getInstanceARN(instance, config.Region),
					CreatedAt:       aws.ToTime(instance.LaunchTime),
					UpdatedAt:       time.Now(),
					DiscoveredAt:    time.Now(),
					DiscoveryMethod: "direct_api",
				}

				// Parse tags
				resource.Tags = p.parseInstanceTags(instance)

				// Parse configuration
				configJSON, _ := json.Marshal(instance)
				resource.Configuration = configJSON

				// Set security and cost flags
				resource.PublicAccess = p.isInstancePublic(instance)
				resource.Encrypted = p.isInstanceEncrypted(instance)
				resource.MonthlyCost = p.estimateInstanceCost(instance)

				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// discoverSecurityGroups discovers security groups
func (p *AWSProvider) discoverSecurityGroups(ctx context.Context, config aws.Config) []core.Resource {
	client := ec2.NewFromConfig(config)
	var resources []core.Resource

	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe security groups: %v", err)
			continue
		}

		for _, sg := range page.SecurityGroups {
			resource := core.Resource{
				ID:              aws.ToString(sg.GroupId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ec2",
				Type:            "security-group",
				Name:            aws.ToString(sg.GroupName),
				ARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:security-group/%s", config.Region, p.getAccountIDFromConfig(config), aws.ToString(sg.GroupId)),
				CreatedAt:       time.Now(), // Security groups don't have creation time
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseSecurityGroupTags(sg)

			// Parse configuration
			configJSON, _ := json.Marshal(sg)
			resource.Configuration = configJSON

			// Check for public access
			resource.PublicAccess = p.isSecurityGroupPublic(sg)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverVolumes discovers EBS volumes
func (p *AWSProvider) discoverVolumes(ctx context.Context, config aws.Config) []core.Resource {
	client := ec2.NewFromConfig(config)
	var resources []core.Resource

	paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe volumes: %v", err)
			continue
		}

		for _, volume := range page.Volumes {
			resource := core.Resource{
				ID:              aws.ToString(volume.VolumeId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ec2",
				Type:            "volume",
				Name:            aws.ToString(volume.VolumeId),
				ARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:volume/%s", config.Region, p.getAccountIDFromConfig(config), aws.ToString(volume.VolumeId)),
				CreatedAt:       aws.ToTime(volume.CreateTime),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseVolumeTags(volume)

			// Parse configuration
			configJSON, _ := json.Marshal(volume)
			resource.Configuration = configJSON

			// Set security and cost flags
			resource.Encrypted = volume.Encrypted != nil && *volume.Encrypted
			resource.MonthlyCost = p.estimateVolumeCost(volume)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverVPCs discovers VPCs
func (p *AWSProvider) discoverVPCs(ctx context.Context, config aws.Config) []core.Resource {
	client := ec2.NewFromConfig(config)
	var resources []core.Resource

	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe VPCs: %v", err)
			continue
		}

		for _, vpc := range page.Vpcs {
			resource := core.Resource{
				ID:              aws.ToString(vpc.VpcId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ec2",
				Type:            "vpc",
				Name:            p.getVPCName(vpc),
				ARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:vpc/%s", config.Region, p.getAccountIDFromConfig(config), aws.ToString(vpc.VpcId)),
				CreatedAt:       time.Now(), // VPCs don't have creation time
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseVPCTags(vpc)

			// Parse configuration
			configJSON, _ := json.Marshal(vpc)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverSubnets discovers subnets
func (p *AWSProvider) discoverSubnets(ctx context.Context, config aws.Config) []core.Resource {
	client := ec2.NewFromConfig(config)
	var resources []core.Resource

	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to describe subnets: %v", err)
			continue
		}

		for _, subnet := range page.Subnets {
			resource := core.Resource{
				ID:              aws.ToString(subnet.SubnetId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ec2",
				Type:            "subnet",
				Name:            p.getSubnetName(subnet),
				ARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:subnet/%s", config.Region, p.getAccountIDFromConfig(config), aws.ToString(subnet.SubnetId)),
				CreatedAt:       time.Now(), // Subnets don't have creation time
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseSubnetTags(subnet)

			// Parse configuration
			configJSON, _ := json.Marshal(subnet)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// Helper methods for EC2 resources
func (p *AWSProvider) getInstanceName(instance ec2Types.Instance) string {
	for _, tag := range instance.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return aws.ToString(instance.InstanceId)
}

func (p *AWSProvider) getInstanceARN(instance ec2Types.Instance, region string) string {
	accountID := p.getAccountIDFromConfig(aws.Config{Region: region})
	return fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, accountID, aws.ToString(instance.InstanceId))
}

func (p *AWSProvider) parseInstanceTags(instance ec2Types.Instance) map[string]string {
	tags := make(map[string]string)
	for _, tag := range instance.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) isInstancePublic(instance ec2Types.Instance) bool {
	// Check if instance has a public IP
	return instance.PublicIpAddress != nil && aws.ToString(instance.PublicIpAddress) != ""
}

func (p *AWSProvider) isInstanceEncrypted(instance ec2Types.Instance) bool {
	// Check if root volume is encrypted
	for _, blockDevice := range instance.BlockDeviceMappings {
		if blockDevice.Ebs != nil {
			// TODO: Check encryption status from volume details
			return false
		}
	}
	return false
}

func (p *AWSProvider) estimateInstanceCost(instance ec2Types.Instance) float64 {
	// TODO: Implement cost estimation based on instance type
	// This would integrate with AWS Pricing API or use a cost database
	return 0.0
}

func (p *AWSProvider) parseSecurityGroupTags(sg ec2Types.SecurityGroup) map[string]string {
	tags := make(map[string]string)
	for _, tag := range sg.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) isSecurityGroupPublic(sg ec2Types.SecurityGroup) bool {
	// Check if security group allows traffic from 0.0.0.0/0
	for _, rule := range sg.IpPermissions {
		for _, ipRange := range rule.IpRanges {
			if aws.ToString(ipRange.CidrIp) == "0.0.0.0/0" {
				return true
			}
		}
	}
	return false
}

func (p *AWSProvider) parseVolumeTags(volume ec2Types.Volume) map[string]string {
	tags := make(map[string]string)
	for _, tag := range volume.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) estimateVolumeCost(volume ec2Types.Volume) float64 {
	// TODO: Implement volume cost estimation
	return 0.0
}

func (p *AWSProvider) getVPCName(vpc ec2Types.Vpc) string {
	for _, tag := range vpc.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return aws.ToString(vpc.VpcId)
}

func (p *AWSProvider) parseVPCTags(vpc ec2Types.Vpc) map[string]string {
	tags := make(map[string]string)
	for _, tag := range vpc.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) getSubnetName(subnet ec2Types.Subnet) string {
	for _, tag := range subnet.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return aws.ToString(subnet.SubnetId)
}

func (p *AWSProvider) parseSubnetTags(subnet ec2Types.Subnet) map[string]string {
	tags := make(map[string]string)
	for _, tag := range subnet.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}
