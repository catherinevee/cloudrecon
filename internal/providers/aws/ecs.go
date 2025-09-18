package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
	"github.com/sirupsen/logrus"
)

// discoverECSResources discovers comprehensive ECS resources
//nolint:unused
func (p *AWSProvider) discoverECSResources(ctx context.Context, config aws.Config) []core.Resource {
	var resources []core.Resource

	// Discover clusters
	resources = append(resources, p.discoverECSClusters(ctx, config)...)

	// Discover services
	resources = append(resources, p.discoverECSServices(ctx, config)...)

	// Discover task definitions
	resources = append(resources, p.discoverECSTaskDefinitions(ctx, config)...)

	// Discover tasks
	resources = append(resources, p.discoverECSTasks(ctx, config)...)

	return resources
}

// discoverECSClusters discovers ECS clusters
//nolint:unused
func (p *AWSProvider) discoverECSClusters(ctx context.Context, config aws.Config) []core.Resource {
	client := ecs.NewFromConfig(config)
	var resources []core.Resource

	paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list ECS clusters: %v", err)
			continue
		}

		for _, clusterArn := range page.ClusterArns {
			// Get detailed cluster information
			cluster, err := p.getECSClusterDetails(ctx, client, clusterArn)
			if err != nil {
				logrus.Warnf("Failed to get ECS cluster details for %s: %v", clusterArn, err)
				continue
			}

			resource := core.Resource{
				ID:              aws.ToString(cluster.ClusterArn),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ecs",
				Type:            "cluster",
				Name:            aws.ToString(cluster.ClusterName),
				ARN:             aws.ToString(cluster.ClusterArn),
				CreatedAt:       time.Now(), // ECS clusters don't have creation time
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseECSClusterTags(cluster)

			// Parse configuration
			configJSON, _ := json.Marshal(cluster)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags["status"] = aws.ToString(cluster.Status)
			resource.Tags["active_services_count"] = fmt.Sprintf("%d", cluster.ActiveServicesCount)
			resource.Tags["running_tasks_count"] = fmt.Sprintf("%d", cluster.RunningTasksCount)
			resource.Tags["pending_tasks_count"] = fmt.Sprintf("%d", cluster.PendingTasksCount)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverECSServices discovers ECS services
//nolint:unused
func (p *AWSProvider) discoverECSServices(ctx context.Context, config aws.Config) []core.Resource {
	client := ecs.NewFromConfig(config)
	var resources []core.Resource

	// First get all clusters
	clusters, err := p.getAllECSClusters(ctx, client)
	if err != nil {
		logrus.Warnf("Failed to get ECS clusters for service discovery: %v", err)
		return resources
	}

	// For each cluster, get its services
	for _, cluster := range clusters {
		services, err := p.getClusterServices(ctx, client, aws.ToString(cluster.ClusterArn))
		if err != nil {
			logrus.Warnf("Failed to get services for cluster %s: %v", aws.ToString(cluster.ClusterArn), err)
			continue
		}

		for _, service := range services {
			resource := core.Resource{
				ID:              aws.ToString(service.ServiceArn),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ecs",
				Type:            "service",
				Name:            aws.ToString(service.ServiceName),
				ARN:             aws.ToString(service.ServiceArn),
				CreatedAt:       aws.ToTime(service.CreatedAt),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseECSServiceTags(service)

			// Parse configuration
			configJSON, _ := json.Marshal(service)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags["status"] = aws.ToString(service.Status)
			resource.Tags["desired_count"] = fmt.Sprintf("%d", service.DesiredCount)
			resource.Tags["running_count"] = fmt.Sprintf("%d", service.RunningCount)
			resource.Tags["pending_count"] = fmt.Sprintf("%d", service.PendingCount)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverECSTaskDefinitions discovers ECS task definitions
//nolint:unused
func (p *AWSProvider) discoverECSTaskDefinitions(ctx context.Context, config aws.Config) []core.Resource {
	client := ecs.NewFromConfig(config)
	var resources []core.Resource

	paginator := ecs.NewListTaskDefinitionsPaginator(client, &ecs.ListTaskDefinitionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Warnf("Failed to list ECS task definitions: %v", err)
			continue
		}

		for _, taskDefArn := range page.TaskDefinitionArns {
			// Get detailed task definition information
			taskDef, err := p.getECSTaskDefinitionDetails(ctx, client, taskDefArn)
			if err != nil {
				logrus.Warnf("Failed to get ECS task definition details for %s: %v", taskDefArn, err)
				continue
			}

			resource := core.Resource{
				ID:              aws.ToString(taskDef.TaskDefinitionArn),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ecs",
				Type:            "task-definition",
				Name:            aws.ToString(taskDef.Family),
				ARN:             aws.ToString(taskDef.TaskDefinitionArn),
				CreatedAt:       aws.ToTime(taskDef.RegisteredAt),
				UpdatedAt:       aws.ToTime(taskDef.RegisteredAt),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseECSTaskDefinitionTags(taskDef)

			// Parse configuration
			configJSON, _ := json.Marshal(taskDef)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags["status"] = string(taskDef.Status)
			resource.Tags["revision"] = fmt.Sprintf("%d", taskDef.Revision)
			resource.Tags["cpu"] = aws.ToString(taskDef.Cpu)
			resource.Tags["memory"] = aws.ToString(taskDef.Memory)

			resources = append(resources, resource)
		}
	}

	return resources
}

// discoverECSTasks discovers ECS tasks
//nolint:unused
func (p *AWSProvider) discoverECSTasks(ctx context.Context, config aws.Config) []core.Resource {
	client := ecs.NewFromConfig(config)
	var resources []core.Resource

	// First get all clusters
	clusters, err := p.getAllECSClusters(ctx, client)
	if err != nil {
		logrus.Warnf("Failed to get ECS clusters for task discovery: %v", err)
		return resources
	}

	// For each cluster, get its tasks
	for _, cluster := range clusters {
		tasks, err := p.getClusterTasks(ctx, client, aws.ToString(cluster.ClusterArn))
		if err != nil {
			logrus.Warnf("Failed to get tasks for cluster %s: %v", aws.ToString(cluster.ClusterArn), err)
			continue
		}

		for _, task := range tasks {
			resource := core.Resource{
				ID:              aws.ToString(task.TaskArn),
				Provider:        "aws",
				AccountID:       p.getAccountIDFromConfig(config),
				Region:          config.Region,
				Service:         "ecs",
				Type:            "task",
				Name:            aws.ToString(task.TaskArn),
				ARN:             aws.ToString(task.TaskArn),
				CreatedAt:       aws.ToTime(task.CreatedAt),
				UpdatedAt:       time.Now(),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "direct_api",
			}

			// Parse tags
			resource.Tags = p.parseECSTaskTags(task)

			// Parse configuration
			configJSON, _ := json.Marshal(task)
			resource.Configuration = configJSON

			// Set status flags
			resource.Tags["last_status"] = aws.ToString(task.LastStatus)
			resource.Tags["desired_status"] = aws.ToString(task.DesiredStatus)
			resource.Tags["health_status"] = string(task.HealthStatus)

			resources = append(resources, resource)
		}
	}

	return resources
}

// Helper methods for ECS resources
//nolint:unused
func (p *AWSProvider) getECSClusterDetails(ctx context.Context, client *ecs.Client, clusterArn string) (*ecsTypes.Cluster, error) {
	result, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterArn},
	})
	if err != nil {
		return nil, err
	}
	if len(result.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", clusterArn)
	}
	return &result.Clusters[0], nil
}

//nolint:unused
func (p *AWSProvider) getAllECSClusters(ctx context.Context, client *ecs.Client) ([]ecsTypes.Cluster, error) {
	var clusters []ecsTypes.Cluster

	paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		// Get detailed cluster information
		clusterDetails, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: page.ClusterArns,
		})
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, clusterDetails.Clusters...)
	}

	return clusters, nil
}

//nolint:unused
func (p *AWSProvider) getClusterServices(ctx context.Context, client *ecs.Client, clusterArn string) ([]ecsTypes.Service, error) {
	var services []ecsTypes.Service

	paginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
		Cluster: aws.String(clusterArn),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		// Get detailed service information
		serviceDetails, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterArn),
			Services: page.ServiceArns,
		})
		if err != nil {
			return nil, err
		}
		services = append(services, serviceDetails.Services...)
	}

	return services, nil
}

//nolint:unused
func (p *AWSProvider) getECSTaskDefinitionDetails(ctx context.Context, client *ecs.Client, taskDefArn string) (*ecsTypes.TaskDefinition, error) {
	result, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefArn),
	})
	if err != nil {
		return nil, err
	}
	return result.TaskDefinition, nil
}

//nolint:unused
func (p *AWSProvider) getClusterTasks(ctx context.Context, client *ecs.Client, clusterArn string) ([]ecsTypes.Task, error) {
	var tasks []ecsTypes.Task

	paginator := ecs.NewListTasksPaginator(client, &ecs.ListTasksInput{
		Cluster: aws.String(clusterArn),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		// Get detailed task information
		taskDetails, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: aws.String(clusterArn),
			Tasks:   page.TaskArns,
		})
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, taskDetails.Tasks...)
	}

	return tasks, nil
}

//nolint:unused
func (p *AWSProvider) parseECSClusterTags(cluster *ecsTypes.Cluster) map[string]string {
	tags := make(map[string]string)
	for _, tag := range cluster.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

//nolint:unused
func (p *AWSProvider) parseECSServiceTags(service ecsTypes.Service) map[string]string {
	tags := make(map[string]string)
	for _, tag := range service.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}

//nolint:unused
func (p *AWSProvider) parseECSTaskDefinitionTags(taskDef *ecsTypes.TaskDefinition) map[string]string {
	tags := make(map[string]string)
	// Task definitions don't have tags in the current API version
	return tags
}

//nolint:unused
func (p *AWSProvider) parseECSTaskTags(task ecsTypes.Task) map[string]string {
	tags := make(map[string]string)
	for _, tag := range task.Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return tags
}
