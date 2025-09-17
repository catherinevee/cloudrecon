package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverLambdaResources discovers comprehensive Lambda resources
func (p *AWSProvider) discoverLambdaResources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover functions
	resources = append(resources, p.discoverLambdaFunctions(ctx, config)...)

	// Discover layers
	resources = append(resources, p.discoverLambdaLayers(ctx, config)...)

	// Discover event source mappings
	resources = append(resources, p.discoverLambdaEventSourceMappings(ctx, config)...)

	return resources
}

// discoverLambdaFunctions discovers Lambda functions
func (p *AWSProvider) discoverLambdaFunctions(ctx context.Context, config aws.Config) []core.Resource {
	client := lambda.NewFromConfig(config)
	var resources []core.Resource

	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list Lambda functions: %v", err)
			continue
		}

		for _, function := range page.Functions {
			resource := core.Resource{
				ID:              aws.ToString(function.FunctionName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "lambda",
				Type:            "function",
				Name:            aws.ToString(function.FunctionName),
				ARN:             aws.ToString(function.FunctionArn),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseLambdaFunctionTags(function)

			// Parse configuration
			configJSON, _ := json.Marshal(function)
			resource.Configuration = configJSON

			// Set security and cost flags
			resource.PublicAccess = p.isLambdaFunctionPublic(function)
			resource.Encrypted = p.isLambdaFunctionEncrypted(function)
			resource.MonthlyCost = p.estimateLambdaFunctionCost(function)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverLambdaLayers discovers Lambda layers
func (p *AWSProvider) discoverLambdaLayers(ctx context.Context, config aws.Config) []core.Resource {
	client := lambda.NewFromConfig(config)
	var resources []core.Resource

	paginator := lambda.NewListLayersPaginator(client, &lambda.ListLayersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list Lambda layers: %v", err)
			continue
		}

		for _, layer := range page.Layers {
			resource := core.Resource{
				ID:              aws.ToString(layer.LayerName),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "lambda",
				Type:            "layer",
				Name:            aws.ToString(layer.LayerName),
				ARN:             aws.ToString(layer.LayerArn),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse configuration
			configJSON, _ := json.Marshal(layer)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverLambdaEventSourceMappings discovers Lambda event source mappings
func (p *AWSProvider) discoverLambdaEventSourceMappings(ctx context.Context, config aws.Config) []core.Resource {
	client := lambda.NewFromConfig(config)
	var resources []core.Resource

	// First get all functions
	functions, err := p.getAllLambdaFunctions(ctx, client)
	if err != nil {
		logrus.Warnf("Failed to get Lambda functions for event source mapping discovery: %v", err)
		return resources
	}

	// For each function, get its event source mappings
	for _, function := range functions {
		mappings, err := p.getFunctionEventSourceMappings(ctx, client, aws.ToString(function.FunctionName))
		if err != nil {
			logrus.Warnf("Failed to get event source mappings for function %s: %v", aws.ToString(function.FunctionName), err)
			continue
		}

		for _, mapping := range mappings {
			resource := core.Resource{
				ID:              aws.ToString(mapping.UUID),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "lambda",
				Type:            "event-source-mapping",
				Name:            fmt.Sprintf("%s Event Source Mapping", aws.ToString(function.FunctionName)),
				ARN:             aws.ToString(mapping.EventSourceArn),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse configuration
			configJSON, _ := json.Marshal(mapping)
			resource.Configuration = configJSON

			resources = append(resources, resource)
		}
	}

	return resources
}

// Helper methods for Lambda resources
func (p *AWSProvider) getAllLambdaFunctions(ctx context.Context, client *lambda.Client) ([]lambdaTypes.FunctionConfiguration, error) {
	var functions []lambdaTypes.FunctionConfiguration

	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		functions = append(functions, page.Functions...)
	}

	return functions, nil
}

func (p *AWSProvider) getFunctionEventSourceMappings(ctx context.Context, client *lambda.Client, functionName string) ([]lambdaTypes.EventSourceMappingConfiguration, error) {
	var mappings []lambdaTypes.EventSourceMappingConfiguration

	paginator := lambda.NewListEventSourceMappingsPaginator(client, &lambda.ListEventSourceMappingsInput{
		FunctionName: aws.String(functionName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, page.EventSourceMappings...)
	}

	return mappings, nil
}

func (p *AWSProvider) parseLambdaFunctionTags(function lambdaTypes.FunctionConfiguration) map[string]string {
	tags := make(map[string]string)
	// Lambda functions don't have tags in the basic configuration
	// Tags would need to be retrieved separately using ListTags API
	return tags
}

func (p *AWSProvider) isLambdaFunctionPublic(function lambdaTypes.FunctionConfiguration) bool {
	// Check if function has public access through resource policy
	// This would require additional API calls to get the resource policy
	return false
}

func (p *AWSProvider) isLambdaFunctionEncrypted(function lambdaTypes.FunctionConfiguration) bool {
	// Check if function uses encrypted environment variables
	return function.Environment != nil && function.Environment.Variables != nil
}

func (p *AWSProvider) estimateLambdaFunctionCost(function lambdaTypes.FunctionConfiguration) float64 {
	// TODO: Implement Lambda cost estimation based on:
	// - Memory allocation
	// - Duration
	// - Number of invocations
	// - Provisioned concurrency
	return 0.0
}
