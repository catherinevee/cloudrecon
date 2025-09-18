package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	elasticacheTypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2Types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snsTypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cloudrecon/cloudrecon/internal/core"
)

// DiscoverCloudFormationStacks discovers CloudFormation stacks
func (p *AWSProvider) DiscoverCloudFormationStacks(ctx context.Context, region string) ([]core.Resource, error) {
	client := cloudformation.NewFromConfig(p.config, func(o *cloudformation.Options) {
		o.Region = region
	})

	var resources []core.Resource
	paginator := cloudformation.NewListStacksPaginator(client, &cloudformation.ListStacksInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list CloudFormation stacks: %w", err)
		}

		for _, stack := range page.StackSummaries {
			if stack.StackStatus == types.StackStatusDeleteComplete {
				continue
			}

			// Get detailed stack information
			stackDetail, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
				StackName: stack.StackName,
			})
			if err != nil {
				continue // Skip stacks that can't be described
			}

			if len(stackDetail.Stacks) == 0 {
				continue
			}

			stack := stackDetail.Stacks[0]
			resource := core.Resource{
				ID:        aws.ToString(stack.StackId),
				Provider:  "aws",
				Region:    region,
				Service:   "cloudformation",
				Type:      "stack",
				Name:      aws.ToString(stack.StackName),
				CreatedAt: aws.ToTime(stack.CreationTime),
				UpdatedAt: time.Now(),
				Tags:      convertTags(stack.Tags),
				Configuration: marshalConfig(map[string]interface{}{
					"stack_status":           string(stack.StackStatus),
					"stack_status_reason":    aws.ToString(stack.StackStatusReason),
					"description":            aws.ToString(stack.Description),
					"capabilities":           stack.Capabilities,
					"parameters":             convertStackParameters(stack.Parameters),
					"outputs":                convertStackOutputs(stack.Outputs),
					"template_url":           "", // TemplateURL field not available in this version
					"parent_id":              aws.ToString(stack.ParentId),
					"root_id":                aws.ToString(stack.RootId),
					"role_arn":               aws.ToString(stack.RoleARN),
					"rollback_configuration": stack.RollbackConfiguration,
					"drift_information":      stack.DriftInformation,
				}),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "api",
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverECSServices discovers ECS services and clusters
func (p *AWSProvider) DiscoverECSServices(ctx context.Context, region string) ([]core.Resource, error) {
	client := ecs.NewFromConfig(p.config, func(o *ecs.Options) {
		o.Region = region
	})

	var resources []core.Resource

	// Discover clusters
	clusterPaginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
	for clusterPaginator.HasMorePages() {
		clusterPage, err := clusterPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list ECS clusters: %w", err)
		}

		for _, clusterArn := range clusterPage.ClusterArns {
			// Get cluster details
			clusterDetail, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
				Clusters: []string{clusterArn},
			})
			if err != nil {
				continue
			}

			if len(clusterDetail.Clusters) == 0 {
				continue
			}

			cluster := clusterDetail.Clusters[0]
			clusterResource := core.Resource{
				ID:        aws.ToString(cluster.ClusterArn),
				Provider:  "aws",
				Region:    region,
				Service:   "ecs",
				Type:      "cluster",
				Name:      aws.ToString(cluster.ClusterName),
				CreatedAt: time.Now(), // RegisteredAt field not available
				UpdatedAt: time.Now(),
				Tags:      convertECSTags(cluster.Tags),
				Configuration: marshalConfig(map[string]interface{}{
					"status":                               aws.ToString(cluster.Status),
					"running_tasks_count":                  cluster.RunningTasksCount,
					"pending_tasks_count":                  cluster.PendingTasksCount,
					"active_services_count":                cluster.ActiveServicesCount,
					"registered_container_instances_count": cluster.RegisteredContainerInstancesCount,
					"capacity_providers":                   cluster.CapacityProviders,
					"default_capacity_provider_strategy":   cluster.DefaultCapacityProviderStrategy,
					"statistics":                           cluster.Statistics,
				}),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "api",
			}

			resources = append(resources, clusterResource)

			// Discover services in this cluster
			servicePaginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
				Cluster: cluster.ClusterArn,
			})

			for servicePaginator.HasMorePages() {
				servicePage, err := servicePaginator.NextPage(ctx)
				if err != nil {
					continue
				}

				if len(servicePage.ServiceArns) == 0 {
					continue
				}

				// Get service details
				serviceDetail, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
					Cluster:  cluster.ClusterArn,
					Services: servicePage.ServiceArns,
				})
				if err != nil {
					continue
				}

				for _, service := range serviceDetail.Services {
					serviceResource := core.Resource{
						ID:        aws.ToString(service.ServiceArn),
						Provider:  "aws",
						Region:    region,
						Service:   "ecs",
						Type:      "service",
						Name:      aws.ToString(service.ServiceName),
						CreatedAt: aws.ToTime(service.CreatedAt),
						UpdatedAt: time.Now(),
						Tags:      convertECSTags(service.Tags),
						Configuration: marshalConfig(map[string]interface{}{
							"status":                            aws.ToString(service.Status),
							"desired_count":                     service.DesiredCount,
							"running_count":                     service.RunningCount,
							"pending_count":                     service.PendingCount,
							"launch_type":                       string(service.LaunchType),
							"platform_version":                  aws.ToString(service.PlatformVersion),
							"platform_family":                   aws.ToString(service.PlatformFamily),
							"task_definition":                   aws.ToString(service.TaskDefinition),
							"deployment_configuration":          service.DeploymentConfiguration,
							"deployments":                       service.Deployments,
							"role_arn":                          aws.ToString(service.RoleArn),
							"events":                            service.Events,
							"created_by":                        aws.ToString(service.CreatedBy),
							"enable_logging":                    false, // EnableLogging field not available
							"enable_execute_command":            service.EnableExecuteCommand,
							"placement_constraints":             service.PlacementConstraints,
							"placement_strategy":                service.PlacementStrategy,
							"network_configuration":             service.NetworkConfiguration,
							"health_check_grace_period_seconds": service.HealthCheckGracePeriodSeconds,
							"scheduling_strategy":               string(service.SchedulingStrategy),
							"deployment_controller":             service.DeploymentController,
							"tags":                              service.Tags,
						}),
						Dependencies:    []string{aws.ToString(cluster.ClusterArn)},
						DiscoveredAt:    time.Now(),
						DiscoveryMethod: "api",
					}

					resources = append(resources, serviceResource)
				}
			}
		}
	}

	return resources, nil
}

// DiscoverElastiCacheClusters discovers ElastiCache clusters
func (p *AWSProvider) DiscoverElastiCacheClusters(ctx context.Context, region string) ([]core.Resource, error) {
	client := elasticache.NewFromConfig(p.config, func(o *elasticache.Options) {
		o.Region = region
	})

	var resources []core.Resource

	// Discover replication groups
	replicationGroupPaginator := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})
	for replicationGroupPaginator.HasMorePages() {
		page, err := replicationGroupPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list ElastiCache replication groups: %w", err)
		}

		for _, rg := range page.ReplicationGroups {
			resource := core.Resource{
				ID:        aws.ToString(rg.ReplicationGroupId),
				Provider:  "aws",
				Region:    region,
				Service:   "elasticache",
				Type:      "replication-group",
				Name:      aws.ToString(rg.ReplicationGroupId),
				CreatedAt: time.Now(), // ElastiCache doesn't provide creation time
				UpdatedAt: time.Now(),
				Tags:      make(map[string]string), // Tags field not available
				Configuration: marshalConfig(map[string]interface{}{
					"status":                      aws.ToString(rg.Status),
					"description":                 aws.ToString(rg.Description),
					"replication_group_arn":       aws.ToString(rg.ARN),
					"node_type":                   aws.ToString(rg.CacheNodeType),
					"engine":                      aws.ToString(rg.Engine),
					"member_clusters":             rg.MemberClusters,
					"automatic_failover":          string(rg.AutomaticFailover),
					"multi_az":                    string(rg.MultiAZ),
					"configuration_endpoint":      rg.ConfigurationEndpoint,
					"snapshot_retention_limit":    aws.ToInt32(rg.SnapshotRetentionLimit),
					"snapshot_window":             aws.ToString(rg.SnapshotWindow),
					"cluster_enabled":             aws.ToBool(rg.ClusterEnabled),
					"cache_cluster_id":            "", // CacheClusterId field not available
					"snapshotting_cluster_id":     aws.ToString(rg.SnapshottingClusterId),
					"log_delivery_configurations": rg.LogDeliveryConfigurations,
				}),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "api",
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverLoadBalancers discovers Application and Network Load Balancers
func (p *AWSProvider) DiscoverLoadBalancers(ctx context.Context, region string) ([]core.Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(p.config, func(o *elasticloadbalancingv2.Options) {
		o.Region = region
	})

	var resources []core.Resource

	// Discover load balancers
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list load balancers: %w", err)
		}

		for _, lb := range page.LoadBalancers {
			// Get load balancer attributes
			attributes, err := client.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{
				LoadBalancerArn: lb.LoadBalancerArn,
			})
			if err != nil {
				continue
			}

			// Get tags
			tags, err := client.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{
				ResourceArns: []string{aws.ToString(lb.LoadBalancerArn)},
			})
			if err != nil {
				continue
			}

			var lbTags map[string]string
			if len(tags.TagDescriptions) > 0 {
				lbTags = convertELBv2Tags(tags.TagDescriptions[0].Tags)
			}

			resource := core.Resource{
				ID:        aws.ToString(lb.LoadBalancerArn),
				Provider:  "aws",
				Region:    region,
				Service:   "elbv2",
				Type:      strings.ToLower(string(lb.Type)),
				Name:      aws.ToString(lb.LoadBalancerName),
				CreatedAt: aws.ToTime(lb.CreatedTime),
				UpdatedAt: time.Now(),
				Tags:      lbTags,
				Configuration: marshalConfig(map[string]interface{}{
					"dns_name":                 aws.ToString(lb.DNSName),
					"canonical_hosted_zone_id": aws.ToString(lb.CanonicalHostedZoneId),
					"created_time":             lb.CreatedTime,
					"load_balancer_name":       aws.ToString(lb.LoadBalancerName),
					"scheme":                   string(lb.Scheme),
					"vpc_id":                   aws.ToString(lb.VpcId),
					"state":                    lb.State,
					"type":                     string(lb.Type),
					"availability_zones":       lb.AvailabilityZones,
					"security_groups":          lb.SecurityGroups,
					"ip_address_type":          string(lb.IpAddressType),
					"customer_owned_ipv4_pool": aws.ToString(lb.CustomerOwnedIpv4Pool),
					"attributes":               convertELBv2Attributes(attributes.Attributes),
				}),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "api",
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverRoute53Zones discovers Route 53 hosted zones
func (p *AWSProvider) DiscoverRoute53Zones(ctx context.Context, region string) ([]core.Resource, error) {
	client := route53.NewFromConfig(p.config, func(o *route53.Options) {
		o.Region = region
	})

	var resources []core.Resource

	paginator := route53.NewListHostedZonesPaginator(client, &route53.ListHostedZonesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Route53 hosted zones: %w", err)
		}

		for _, zone := range page.HostedZones {
			// Get zone tags
			tags, err := client.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
				ResourceId:   zone.Id,
				ResourceType: route53Types.TagResourceTypeHostedzone,
			})
			if err != nil {
				continue
			}

			resource := core.Resource{
				ID:        aws.ToString(zone.Id),
				Provider:  "aws",
				Region:    "global", // Route53 is global
				Service:   "route53",
				Type:      "hosted-zone",
				Name:      aws.ToString(zone.Name),
				CreatedAt: time.Now(), // Route53 doesn't provide creation time
				UpdatedAt: time.Now(),
				Tags:      convertRoute53Tags(tags.ResourceTagSet.Tags),
				Configuration: marshalConfig(map[string]interface{}{
					"caller_reference": aws.ToString(zone.CallerReference),
					"config":           zone.Config,
					"resource_count":   aws.ToInt64(zone.ResourceRecordSetCount),
					"linked_service":   zone.LinkedService,
				}),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "api",
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverSNSTopics discovers SNS topics
func (p *AWSProvider) DiscoverSNSTopics(ctx context.Context, region string) ([]core.Resource, error) {
	client := sns.NewFromConfig(p.config, func(o *sns.Options) {
		o.Region = region
	})

	var resources []core.Resource

	paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list SNS topics: %w", err)
		}

		for _, topic := range page.Topics {
			// Get topic attributes
			attributes, err := client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
				TopicArn: topic.TopicArn,
			})
			if err != nil {
				continue
			}

			// Get topic tags
			tags, err := client.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{
				ResourceArn: topic.TopicArn,
			})
			if err != nil {
				continue
			}

			resource := core.Resource{
				ID:        aws.ToString(topic.TopicArn),
				Provider:  "aws",
				Region:    region,
				Service:   "sns",
				Type:      "topic",
				Name:      aws.ToString(topic.TopicArn),
				CreatedAt: time.Now(), // SNS doesn't provide creation time
				UpdatedAt: time.Now(),
				Tags:      convertSNSTags(tags.Tags),
				Configuration: marshalConfig(map[string]interface{}{
					"topic_arn":  aws.ToString(topic.TopicArn),
					"attributes": attributes.Attributes,
				}),
				DiscoveredAt:    time.Now(),
				DiscoveryMethod: "api",
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverSQSQueues discovers SQS queues
func (p *AWSProvider) DiscoverSQSQueues(ctx context.Context, region string) ([]core.Resource, error) {
	client := sqs.NewFromConfig(p.config, func(o *sqs.Options) {
		o.Region = region
	})

	var resources []core.Resource

	// List queues
	result, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list SQS queues: %w", err)
	}

	for _, queueUrl := range result.QueueUrls {
		// Get queue attributes
		attributes, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(queueUrl),
			AttributeNames: []sqsTypes.QueueAttributeName{
				sqsTypes.QueueAttributeNameAll,
			},
		})
		if err != nil {
			continue
		}

		// Get queue tags
		tags, err := client.ListQueueTags(ctx, &sqs.ListQueueTagsInput{
			QueueUrl: aws.String(queueUrl),
		})
		if err != nil {
			continue
		}

		queueName := strings.Split(queueUrl, "/")[len(strings.Split(queueUrl, "/"))-1]

		resource := core.Resource{
			ID:        queueUrl,
			Provider:  "aws",
			Region:    region,
			Service:   "sqs",
			Type:      "queue",
			Name:      queueName,
			CreatedAt: time.Now(), // SQS doesn't provide creation time
			UpdatedAt: time.Now(),
			Tags:      convertSQSTags(tags.Tags),
			Configuration: marshalConfig(map[string]interface{}{
				"queue_url":  queueUrl,
				"attributes": attributes.Attributes,
			}),
			DiscoveredAt:    time.Now(),
			DiscoveryMethod: "api",
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// Helper functions for converting AWS types to our format
func convertStackParameters(params []types.Parameter) map[string]interface{} {
	result := make(map[string]interface{})
	for _, param := range params {
		result[aws.ToString(param.ParameterKey)] = map[string]interface{}{
			"value":              aws.ToString(param.ParameterValue),
			"use_previous_value": aws.ToBool(param.UsePreviousValue),
			"resolved_value":     aws.ToString(param.ResolvedValue),
		}
	}
	return result
}

func convertStackOutputs(outputs []types.Output) map[string]interface{} {
	result := make(map[string]interface{})
	for _, output := range outputs {
		result[aws.ToString(output.OutputKey)] = map[string]interface{}{
			"value":       aws.ToString(output.OutputValue),
			"description": aws.ToString(output.Description),
			"export_name": aws.ToString(output.ExportName),
		}
	}
	return result
}

func convertECSTags(tags []ecsTypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

//nolint:unused
func convertElastiCacheTags(tags []elasticacheTypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func convertELBv2Tags(tags []elbv2Types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func convertELBv2Attributes(attributes []elbv2Types.LoadBalancerAttribute) map[string]string {
	result := make(map[string]string)
	for _, attr := range attributes {
		result[aws.ToString(attr.Key)] = aws.ToString(attr.Value)
	}
	return result
}

func convertRoute53Tags(tags []route53Types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func convertSNSTags(tags []snsTypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func convertSQSTags(tags map[string]string) map[string]string {
	return tags
}

// convertTags converts AWS tags to our format
func convertTags(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

// marshalConfig converts a map to json.RawMessage
func marshalConfig(config map[string]interface{}) json.RawMessage {
	data, _ := json.Marshal(config)
	return json.RawMessage(data)
}
