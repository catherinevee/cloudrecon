package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudrecon/cloudrecon/internal/core"
	"gopkg.in/yaml.v2"
)

type Exporter struct{}

// NewExporter creates a new exporter
func NewExporter() *Exporter {
	return &Exporter{}
}

// Export exports resources to various formats
func (e *Exporter) Export(resources []core.Resource, format, outputPath string) error {
	switch strings.ToLower(format) {
	case "json":
		return e.exportJSON(resources, outputPath)
	case "csv":
		return e.exportCSV(resources, outputPath)
	case "yaml":
		return e.exportYAML(resources, outputPath)
	case "terraform":
		return e.exportTerraform(resources, outputPath)
	case "grafana":
		return e.exportGrafana(resources, outputPath)
	case "datadog":
		return e.exportDatadog(resources, outputPath)
	case "splunk":
		return e.exportSplunk(resources, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports resources to JSON format
func (e *Exporter) exportJSON(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(resources)
}

// exportCSV exports resources to CSV format
func (e *Exporter) exportCSV(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "Provider", "AccountID", "Region", "Service", "Type", "Name", "ARN",
		"CreatedAt", "UpdatedAt", "PublicAccess", "Encrypted", "MonthlyCost",
		"DiscoveredAt", "DiscoveryMethod",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, resource := range resources {
		record := []string{
			resource.ID,
			resource.Provider,
			resource.AccountID,
			resource.Region,
			resource.Service,
			resource.Type,
			resource.Name,
			resource.ARN,
			resource.CreatedAt.Format(time.RFC3339),
			resource.UpdatedAt.Format(time.RFC3339),
			fmt.Sprintf("%t", resource.PublicAccess),
			fmt.Sprintf("%t", resource.Encrypted),
			fmt.Sprintf("%.2f", resource.MonthlyCost),
			resource.DiscoveredAt.Format(time.RFC3339),
			resource.DiscoveryMethod,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// exportYAML exports resources to YAML format
func (e *Exporter) exportYAML(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	data, err := yaml.Marshal(resources)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	_, err = file.Write(data)
	return err
}

// exportTerraform exports resources to Terraform state format
func (e *Exporter) exportTerraform(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Create Terraform state structure
	state := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.0.0",
		"serial":            1,
		"lineage":           "cloudrecon-export",
		"outputs":           map[string]interface{}{},
		"resources":         []map[string]interface{}{},
	}

	// Group resources by provider
	providerResources := make(map[string][]core.Resource)
	for _, resource := range resources {
		providerResources[resource.Provider] = append(providerResources[resource.Provider], resource)
	}

	// Convert resources to Terraform state format
	for provider, providerResources := range providerResources {
		resourceType := fmt.Sprintf("%s_resource", provider)
		instances := []map[string]interface{}{}

		for _, resource := range providerResources {
			instance := map[string]interface{}{
				"schema_version": 0,
				"attributes": map[string]interface{}{
					"id":   resource.ID,
					"name": resource.Name,
					"type": resource.Type,
					"tags": resource.Tags,
				},
			}
			instances = append(instances, instance)
		}

		resource := map[string]interface{}{
			"mode":      "managed",
			"type":      resourceType,
			"name":      "cloudrecon_resources",
			"provider":  fmt.Sprintf("provider[\"registry.terraform.io/hashicorp/%s\"]", provider),
			"instances": instances,
		}

		state["resources"] = append(state["resources"].([]map[string]interface{}), resource)
	}

	// Marshal to JSON (Terraform state is JSON)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Terraform state: %w", err)
	}

	_, err = file.Write(data)
	return err
}

// exportGrafana exports resources to Grafana dashboard format
func (e *Exporter) exportGrafana(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Create Grafana dashboard
	dashboard := map[string]interface{}{
		"dashboard": map[string]interface{}{
			"id":       nil,
			"title":    "CloudRecon Resources",
			"tags":     []string{"cloudrecon", "cloud", "resources"},
			"timezone": "browser",
			"panels": []map[string]interface{}{
				{
					"id":    1,
					"title": "Resources by Provider",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "sum by (provider) (cloudrecon_resources_total)",
							"legendFormat": "{{provider}}",
						},
					},
				},
				{
					"id":    2,
					"title": "Resources by Service",
					"type":  "piechart",
					"targets": []map[string]interface{}{
						{
							"expr":         "sum by (service) (cloudrecon_resources_total)",
							"legendFormat": "{{service}}",
						},
					},
				},
				{
					"id":    3,
					"title": "Monthly Cost by Provider",
					"type":  "bargauge",
					"targets": []map[string]interface{}{
						{
							"expr":         "sum by (provider) (cloudrecon_monthly_cost)",
							"legendFormat": "{{provider}}",
						},
					},
				},
			},
			"time": map[string]interface{}{
				"from": "now-1h",
				"to":   "now",
			},
			"refresh": "5m",
		},
	}

	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Grafana dashboard: %w", err)
	}

	_, err = file.Write(data)
	return err
}

// exportDatadog exports resources to Datadog format
func (e *Exporter) exportDatadog(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Create Datadog dashboard
	dashboard := map[string]interface{}{
		"title":       "CloudRecon Resources",
		"description": "Cloud resource inventory dashboard",
		"widgets": []map[string]interface{}{
			{
				"definition": map[string]interface{}{
					"type": "timeseries",
					"requests": []map[string]interface{}{
						{
							"q":            "sum:cloudrecon.resources.count{*} by {provider}",
							"display_type": "line",
						},
					},
					"title": "Resources by Provider",
				},
			},
			{
				"definition": map[string]interface{}{
					"type": "toplist",
					"requests": []map[string]interface{}{
						{
							"q": "top(sum:cloudrecon.monthly_cost{*} by {provider}, 10, 'value', 'desc')",
						},
					},
					"title": "Top 10 Providers by Cost",
				},
			},
		},
	}

	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Datadog dashboard: %w", err)
	}

	_, err = file.Write(data)
	return err
}

// exportSplunk exports resources to Splunk format
func (e *Exporter) exportSplunk(resources []core.Resource, outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Export as JSONL (JSON Lines) format for Splunk
	for _, resource := range resources {
		event := map[string]interface{}{
			"time":       resource.DiscoveredAt.Unix(),
			"source":     "cloudrecon",
			"sourcetype": "cloudrecon:resources",
			"index":      "cloudrecon",
			"event":      resource,
		}

		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal Splunk event: %w", err)
		}

		if _, err := file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write Splunk event: %w", err)
		}
	}

	return nil
}
