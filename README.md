# CloudRecon

> **Multi-cloud infrastructure discovery and analysis in a single command**

[![CloudRecon CI/CD Pipeline](https://github.com/catherinevee/cloudrecon/workflows/CloudRecon%20CI%2FCD%20Pipeline/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![CI/CD Pipeline](https://github.com/catherinevee/cloudrecon/workflows/CI%2FCD%20Pipeline/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![Docker Build](https://github.com/catherinevee/cloudrecon/workflows/Docker%20Build/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![Trivy Security Scan](https://github.com/catherinevee/cloudrecon/workflows/Trivy%20Security%20Scan/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)]()

## Highlights

- **Zero Infrastructure**: Single binary, no databases, no agents, no servers required
- **Multi-Cloud Support**: Discover resources across AWS, Azure, and GCP simultaneously
- **Offline-First**: Local SQLite caching for instant queries and analysis
- **Intelligent Discovery**: Leverages native cloud tools (AWS Config, Azure Resource Graph, GCP Asset Inventory)
- **Comprehensive Analysis**: Dependency mapping, security analysis, and cost estimation
- **Interactive CLI**: Guided workflows with beautiful terminal UI
- **Performance Optimized**: Parallel processing and intelligent caching for large datasets
- **Rich Export**: Multiple formats including beautiful HTML reports and PDF-ready output

## Overview

CloudRecon is a powerful command-line tool that provides instant visibility into your multi-cloud infrastructure. Whether you're managing resources across AWS, Azure, and GCP, or need to understand dependencies, security posture, and costs, CloudRecon delivers comprehensive insights with zero infrastructure requirements.

### The Problem It Solves

Managing multi-cloud infrastructure is complex. Traditional solutions require:
- Multiple tools for different cloud providers
- Complex setup and configuration
- Ongoing maintenance and updates
- Limited cross-cloud visibility
- Expensive enterprise solutions

CloudRecon eliminates these barriers by providing a single, lightweight tool that discovers, analyzes, and reports on your entire cloud infrastructure.

### Why CloudRecon?

- **Unified View**: See all your cloud resources in one place
- **Instant Insights**: Get dependency maps, security findings, and cost analysis immediately
- **No Setup**: Download and run - no configuration required
- **Offline Capable**: Work with cached data when you need it
- **Developer Friendly**: Simple CLI with interactive modes and rich export options

### How It Works

CloudRecon uses a hybrid discovery approach:
1. **Native Tools First**: Leverages AWS Config, Azure Resource Graph, and GCP Asset Inventory when available
2. **API Fallback**: Falls back to direct API calls for comprehensive coverage
3. **Local Storage**: Caches everything in SQLite for instant offline access
4. **Smart Analysis**: Processes data with parallel algorithms for fast insights

## Usage

### Quick Start

```bash
# Discover all cloud resources
cloudrecon discover

# Run comprehensive analysis
cloudrecon analyze

# Interactive mode
cloudrecon interactive

# Query specific resources
cloudrecon query "SELECT * FROM resources WHERE provider='aws'"
```

### Interactive Analysis

```bash
cloudrecon interactive
```

Choose from:
- Quick Analysis (Dependencies + Security + Cost)
- Dependency Analysis Only
- Security Analysis Only
- Cost Analysis Only
- Custom Analysis
- Export Results
- View Cache Statistics

### Command Examples

```bash
# Discover resources from specific providers
cloudrecon discover --providers aws,azure

# Export to HTML report
cloudrecon export --format html --output report.html

# Security analysis with custom rules
cloudrecon security --rules strict

# Cost analysis with optimization tips
cloudrecon cost --optimize

# Query with natural language
cloudrecon ask "What are my most expensive resources?"
```

## Installation

### Pre-built Binaries

Download the latest release for your platform:

```bash
# Windows
curl -L https://github.com/cloudrecon/cloudrecon/releases/latest/download/cloudrecon-windows-amd64.exe -o cloudrecon.exe

# Linux
curl -L https://github.com/cloudrecon/cloudrecon/releases/latest/download/cloudrecon-linux-amd64 -o cloudrecon
chmod +x cloudrecon

# macOS
curl -L https://github.com/cloudrecon/cloudrecon/releases/latest/download/cloudrecon-darwin-amd64 -o cloudrecon
chmod +x cloudrecon
```

### From Source

```bash
git clone https://github.com/cloudrecon/cloudrecon.git
cd cloudrecon
go build -o cloudrecon ./cmd/cloudrecon
```

### Docker

```bash
# Pull the latest image
docker pull ghcr.io/cloudrecon/cloudrecon:latest

# Run CloudRecon
docker run --rm -v ~/.aws:/root/.aws -v ~/.azure:/root/.azure -v ~/.config/gcloud:/root/.config/gcloud ghcr.io/cloudrecon/cloudrecon:latest discover

# Or build locally
docker build -t cloudrecon .
docker run --rm -v ~/.aws:/root/.aws -v ~/.azure:/root/.azure -v ~/.config/gcloud:/root/.config/gcloud cloudrecon discover
```

### Requirements

- **Go 1.21+** (for building from source)
- **Cloud Credentials**: AWS, Azure, or GCP credentials configured
- **Platforms**: Windows, Linux, macOS (x64 and ARM64)

## Configuration

CloudRecon works out of the box, but you can customize behavior with a config file:

```yaml
# config/cloudrecon.yaml
providers:
  aws:
    regions: ["us-east-1", "us-west-2"]
    use_config: true
  azure:
    subscriptions: ["sub-123", "sub-456"]
  gcp:
    projects: ["project-1", "project-2"]

discovery:
  parallel_workers: 10
  cache_duration: "24h"
  rate_limit: 100

analysis:
  enable_dependencies: true
  enable_security: true
  enable_cost: true
  parallel_analysis: true
```

## Features

### Discovery
- **Multi-Cloud**: AWS, Azure, and GCP support
- **Service Coverage**: 50+ cloud services across all providers
- **Native Integration**: Uses cloud-native tools when superior
- **Parallel Processing**: Fast discovery with configurable concurrency
- **Incremental Updates**: Only discover what's changed

### Analysis
- **Dependency Mapping**: Understand resource relationships
- **Security Analysis**: Find misconfigurations and vulnerabilities
- **Cost Estimation**: Calculate and optimize cloud spending
- **Compliance Checking**: Validate against security frameworks
- **Performance Metrics**: Track analysis speed and efficiency

### Export & Reporting
- **Multiple Formats**: JSON, CSV, YAML, HTML, PDF
- **Beautiful Reports**: Rich HTML reports with charts and graphs
- **Custom Queries**: SQL and natural language support
- **Templates**: Pre-built report templates
- **Scheduled Exports**: Automated reporting capabilities

### Performance
- **Parallel Processing**: Multi-threaded analysis
- **Intelligent Caching**: Avoid redundant API calls
- **Memory Optimization**: Handle large datasets efficiently
- **Rate Limiting**: Respect cloud provider limits
- **Progress Tracking**: Real-time progress indicators

## Examples That Inspire Us

We've learned from the best READMEs in the open source community:

- [fatiando/pooch](https://github.com/fatiando/pooch) - Clean, friendly, and comprehensive
- [gruns/furl](https://github.com/gruns/furl) - Excellent examples and clear value proposition
- [giampaolo/psutil](https://github.com/giampaolo/psutil) - Great cross-platform support documentation
- [MonitorControl/MonitorControl](https://github.com/MonitorControl/MonitorControl) - Beautiful feature highlights

## Contributing

We welcome contributions! Here's how you can help:

### Report Issues
Found a bug or have a feature request? [Open an issue](https://github.com/cloudrecon/cloudrecon/issues) and let us know!

### Contribute Code
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Improve Documentation
- Fix typos and improve clarity
- Add examples and use cases
- Translate documentation to other languages
- Create video tutorials and guides

### Help with Testing
- Test on different platforms and cloud providers
- Report edge cases and compatibility issues
- Improve test coverage
- Performance testing and optimization

## Further Reading

- [CloudRecon Documentation](https://docs.cloudrecon.dev)
- [API Reference](https://docs.cloudrecon.dev/api)
- [Configuration Guide](https://docs.cloudrecon.dev/configuration)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- **AWS, Azure, and GCP** for their excellent SDKs and APIs
- **The Go Community** for amazing libraries and tools
- **Open Source Contributors** who make projects like this possible
- **Bane Sullivan** for the [excellent README guide](https://github.com/banesullivan/README) that inspired this documentation

---

**Made with love by the CloudRecon team**

*CloudRecon: Because your cloud infrastructure deserves better visibility.*