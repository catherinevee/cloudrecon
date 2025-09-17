package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverIAMResources discovers comprehensive IAM resources
func (p *AWSProvider) discoverIAMResources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover users
	resources = append(resources, p.discoverIAMUsers(ctx, config)...)

	// Discover roles
	resources = append(resources, p.discoverIAMRoles(ctx, config)...)

	// Discover groups
	resources = append(resources, p.discoverIAMGroups(ctx, config)...)

	// Discover policies
	resources = append(resources, p.discoverIAMPolicies(ctx, config)...)

	// Discover access keys
	resources = append(resources, p.discoverIAMAccessKeys(ctx, config)...)

	return resources
}

// discoverIAMUsers discovers IAM users
func (p *AWSProvider) discoverIAMUsers(ctx context.Context, config aws.Config) []core.Resource {
	client := iam.NewFromConfig(config)
	var resources []core.Resource

	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list IAM users: %v", err)
			continue
		}

		for _, user := range page.Users {
			resource := core.Resource{
				ID:              aws.ToString(user.UserName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          "global", // IAM is global
				Service:         "iam",
				Type:            "user",
				Name:            aws.ToString(user.UserName),
				ARN:             aws.ToString(user.Arn),
				CreatedAt:       aws.ToTime(user.CreateDate),
				UpdatedAt:       aws.ToTime(user.PasswordLastUsed),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseIAMUserTags(user)

			// Parse configuration
			configJSON, _ := json.Marshal(user)
			resource.Configuration = configJSON

			// Check for security issues
			resource.PublicAccess = p.isUserPublic(user)
			resource.Encrypted = p.isUserEncrypted(user)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverIAMRoles discovers IAM roles
func (p *AWSProvider) discoverIAMRoles(ctx context.Context, config aws.Config) []core.Resource {
	client := iam.NewFromConfig(config)
	var resources []core.Resource

	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list IAM roles: %v", err)
			continue
		}

		for _, role := range page.Roles {
			resource := core.Resource{
				ID:              aws.ToString(role.RoleName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          "global", // IAM is global
				Service:         "iam",
				Type:            "role",
				Name:            aws.ToString(role.RoleName),
				ARN:             aws.ToString(role.Arn),
				CreatedAt:       aws.ToTime(role.CreateDate),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseIAMRoleTags(role)

			// Parse configuration
			configJSON, _ := json.Marshal(role)
			resource.Configuration = configJSON

			// Check for security issues
			resource.PublicAccess = p.isRolePublic(role)
			resource.Encrypted = p.isRoleEncrypted(role)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverIAMGroups discovers IAM groups
func (p *AWSProvider) discoverIAMGroups(ctx context.Context, config aws.Config) []core.Resource {
	client := iam.NewFromConfig(config)
	var resources []core.Resource

	paginator := iam.NewListGroupsPaginator(client, &iam.ListGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list IAM groups: %v", err)
			continue
		}

		for _, group := range page.Groups {
			resource := core.Resource{
				ID:              aws.ToString(group.GroupName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          "global", // IAM is global
				Service:         "iam",
				Type:            "group",
				Name:            aws.ToString(group.GroupName),
				ARN:             aws.ToString(group.Arn),
				CreatedAt:       aws.ToTime(group.CreateDate),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseIAMGroupTags(group)

			// Parse configuration
			configJSON, _ := json.Marshal(group)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverIAMPolicies discovers IAM policies
func (p *AWSProvider) discoverIAMPolicies(ctx context.Context, config aws.Config) []core.Resource {
	client := iam.NewFromConfig(config)
	var resources []core.Resource

	// Discover customer managed policies
	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
		Scope: iamTypes.PolicyScopeTypeLocal,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list IAM policies: %v", err)
			continue
		}

		for _, policy := range page.Policies {
			resource := core.Resource{
				ID:              aws.ToString(policy.PolicyName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          "global", // IAM is global
				Service:         "iam",
				Type:            "policy",
				Name:            aws.ToString(policy.PolicyName),
				ARN:             aws.ToString(policy.Arn),
				CreatedAt:       aws.ToTime(policy.CreateDate),
				UpdatedAt:       aws.ToTime(policy.UpdateDate),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseIAMPolicyTags(policy)

			// Parse configuration
			configJSON, _ := json.Marshal(policy)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverIAMAccessKeys discovers IAM access keys
func (p *AWSProvider) discoverIAMAccessKeys(ctx context.Context, config aws.Config) []core.Resource {
	client := iam.NewFromConfig(config)
	var resources []core.Resource

	// First get all users
	users, err := p.getAllUsers(ctx, client)
	if err != nil {
		logrus.Warnf("Failed to get users for access key discovery: %v", err)
		return resources
	}

	// For each user, get their access keys
	for _, user := range users {
		accessKeys, err := p.getUserAccessKeys(ctx, client, aws.ToString(user.UserName))
		if err != nil {
			logrus.Warnf("Failed to get access keys for user %s: %v", aws.ToString(user.UserName), err)
			continue
		}

		for _, accessKey := range accessKeys {
			resource := core.Resource{
				ID:              aws.ToString(accessKey.AccessKeyId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          "global", // IAM is global
				Service:         "iam",
				Type:            "access-key",
				Name:            fmt.Sprintf("%s Access Key", aws.ToString(user.UserName)),
				ARN:             fmt.Sprintf("arn:aws:iam::%s:user/%s", p.getAccountIDFromConfig(config), aws.ToString(user.UserName)),
				CreatedAt:       aws.ToTime(accessKey.CreateDate),
				UpdatedAt:       aws.ToTime(accessKey.CreateDate),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse configuration
			configJSON, _ := json.Marshal(accessKey)
			resource.Configuration = configJSON

			// Check for security issues
			resource.PublicAccess = p.isAccessKeyPublic(accessKey)
			resource.Encrypted = p.isAccessKeyEncrypted(accessKey)

			resources = append(resources, resource)
		}
	}

	return resources
}

// Helper methods for IAM resources
func (p *AWSProvider) getAllUsers(ctx context.Context, client *iam.Client) ([]iamTypes.User, error) {
	var users []iamTypes.User

	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		users = append(users, page.Users...)
	}

	return users, nil
}

func (p *AWSProvider) getUserAccessKeys(ctx context.Context, client *iam.Client, userName string) ([]iamTypes.AccessKeyMetadata, error) {
	var accessKeys []iamTypes.AccessKeyMetadata

	paginator := iam.NewListAccessKeysPaginator(client, &iam.ListAccessKeysInput{
		UserName: aws.String(userName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		accessKeys = append(accessKeys, page.AccessKeyMetadata...)
	}

	return accessKeys, nil
}

func (p *AWSProvider) parseIAMUserTags(user iamTypes.User) map[string]string {
	tags := make(map[string]string)
	for _, tag := range user.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) isUserPublic(user iamTypes.User) bool {
	// Check if user has public access (e.g., through policies)
	// This is a simplified check - in reality, you'd need to analyze policies
	return false
}

func (p *AWSProvider) isUserEncrypted(user iamTypes.User) bool {
	// Check if user has encryption requirements
	// This is a simplified check - in reality, you'd need to analyze policies
	return false
}

func (p *AWSProvider) parseIAMRoleTags(role iamTypes.Role) map[string]string {
	tags := make(map[string]string)
	for _, tag := range role.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) isRolePublic(role iamTypes.Role) bool {
	// Check if role has public access (e.g., through trust policy)
	// This is a simplified check - in reality, you'd need to analyze trust policies
	return false
}

func (p *AWSProvider) isRoleEncrypted(role iamTypes.Role) bool {
	// Check if role has encryption requirements
	// This is a simplified check - in reality, you'd need to analyze policies
	return false
}

func (p *AWSProvider) parseIAMGroupTags(group iamTypes.Group) map[string]string {
	tags := make(map[string]string)
	// IAM Groups don't have tags in the current API version
	return tags
}

func (p *AWSProvider) parseIAMPolicyTags(policy iamTypes.Policy) map[string]string {
	tags := make(map[string]string)
	for _, tag := range policy.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

func (p *AWSProvider) isAccessKeyPublic(accessKey iamTypes.AccessKeyMetadata) bool {
	// Check if access key has public access
	// This is a simplified check - in reality, you'd need to analyze policies
	return false
}

func (p *AWSProvider) isAccessKeyEncrypted(accessKey iamTypes.AccessKeyMetadata) bool {
	// Check if access key has encryption requirements
	// This is a simplified check - in reality, you'd need to analyze policies
	return false
}
