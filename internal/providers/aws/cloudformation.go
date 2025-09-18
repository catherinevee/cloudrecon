package aws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverCloudFormationResources discovers comprehensive CloudFormation resources
//nolint:unused
func (p *AWSProvider) discoverCloudFormationResources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover stacks
	resources = append(resources, p.discoverCloudFormationStacks(ctx, config)...)

	// Discover stack sets
	resources = append(resources, p.discoverCloudFormationStackSets(ctx, config)...)

	// Discover change sets
	resources = append(resources, p.discoverCloudFormationChangeSets(ctx, config)...)

	return resources
}

// discoverCloudFormationStacks discovers CloudFormation stacks
//nolint:unused
func (p *AWSProvider) discoverCloudFormationStacks(ctx context.Context, config aws.Config) []core.Resource {
	client := cloudformation.NewFromConfig(config)
	var resources []core.Resource

	paginator := cloudformation.NewListStacksPaginator(client, &cloudformation.ListStacksInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list CloudFormation stacks: %v", err)
			continue
		}

		for _, stack := range page.StackSummaries {
			// Skip deleted stacks unless in deep mode
			if stack.StackStatus == cfTypes.StackStatusDeleteComplete {
				continue
			}

			resource := core.Resource{
				ID:              aws.ToString(stack.StackId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "cloudformation",
				Type:            "stack",
				Name:            aws.ToString(stack.StackName),
				ARN:             aws.ToString(stack.StackId),
				CreatedAt:       aws.ToTime(stack.CreationTime),
				UpdatedAt:       aws.ToTime(stack.LastUpdatedTime),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseCloudFormationStackTags(stack)

			// Parse configuration
			configJSON, _ := json.Marshal(stack)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags["stack_status"] = string(stack.StackStatus)
			resource.Tags["parent_stack_id"] = aws.ToString(stack.ParentId)
			resource.Tags["root_stack_id"] = aws.ToString(stack.RootId)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverCloudFormationStackSets discovers CloudFormation stack sets
//nolint:unused
func (p *AWSProvider) discoverCloudFormationStackSets(ctx context.Context, config aws.Config) []core.Resource {
	client := cloudformation.NewFromConfig(config)
	var resources []core.Resource

	paginator := cloudformation.NewListStackSetsPaginator(client, &cloudformation.ListStackSetsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list CloudFormation stack sets: %v", err)
			continue
		}

		for _, stackSet := range page.Summaries {
			resource := core.Resource{
				ID:              aws.ToString(stackSet.StackSetId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "cloudformation",
				Type:            "stack-set",
				Name:            aws.ToString(stackSet.StackSetName),
				ARN:             aws.ToString(stackSet.StackSetId),
				CreatedAt:       time.Now(), // StackSet summaries don't have creation time
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse configuration
			configJSON, _ := json.Marshal(stackSet)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags = map[string]string{
				"stack_set_status": string(stackSet.Status),
				"description":      aws.ToString(stackSet.Description),
			}

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverCloudFormationChangeSets discovers CloudFormation change sets
//nolint:unused
func (p *AWSProvider) discoverCloudFormationChangeSets(ctx context.Context, config aws.Config) []core.Resource {
	client := cloudformation.NewFromConfig(config)
	var resources []core.Resource

	// First get all stacks
	stacks, err := p.getAllCloudFormationStacks(ctx, client)
	if err != nil {
		logrus.Warnf("Failed to get CloudFormation stacks for change set discovery: %v", err)
		return resources
	}

	// For each stack, get its change sets
	for _, stack := range stacks {
		changeSets, err := p.getStackChangeSets(ctx, client, aws.ToString(stack.StackName))
		if err != nil {
			logrus.Warnf("Failed to get change sets for stack %s: %v", aws.ToString(stack.StackName), err)
			continue
		}

		for _, changeSet := range changeSets {
			resource := core.Resource{
				ID:              aws.ToString(changeSet.ChangeSetId),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "cloudformation",
				Type:            "change-set",
				Name:            aws.ToString(changeSet.ChangeSetName),
				ARN:             aws.ToString(changeSet.ChangeSetId),
				CreatedAt:       aws.ToTime(changeSet.CreationTime),
				UpdatedAt:       aws.ToTime(changeSet.CreationTime),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse configuration
			configJSON, _ := json.Marshal(changeSet)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags = map[string]string{
				"change_set_status": string(changeSet.Status),
				"stack_name":        aws.ToString(changeSet.StackName),
				"description":       aws.ToString(changeSet.Description),
			}

			resources = append(resources, resource)
		}
	}

	return resources
}

// Helper methods for CloudFormation resources
//nolint:unused
func (p *AWSProvider) getAllCloudFormationStacks(ctx context.Context, client *cloudformation.Client) ([]cfTypes.StackSummary, error) {
	var stacks []cfTypes.StackSummary

	paginator := cloudformation.NewListStacksPaginator(client, &cloudformation.ListStacksInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, page.StackSummaries...)
	}

	return stacks, nil
}

//nolint:unused
func (p *AWSProvider) getStackChangeSets(ctx context.Context, client *cloudformation.Client, stackName string) ([]cfTypes.ChangeSetSummary, error) {
	var changeSets []cfTypes.ChangeSetSummary

	paginator := cloudformation.NewListChangeSetsPaginator(client, &cloudformation.ListChangeSetsInput{
		StackName: aws.String(stackName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		changeSets = append(changeSets, page.Summaries...)
	}

	return changeSets, nil
}

//nolint:unused
func (p *AWSProvider) parseCloudFormationStackTags(stack cfTypes.StackSummary) map[string]string {
	tags := make(map[string]string)
	// Stack summaries don't include tags, would need separate API call
	return tags
}
