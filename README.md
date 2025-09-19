# CloudRecon

[![CloudRecon CI/CD Pipeline](https://github.com/catherinevee/cloudrecon/workflows/CloudRecon%20CI%2FCD%20Pipeline/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![CI/CD Pipeline](https://github.com/catherinevee/cloudrecon/workflows/CI%2FCD%20Pipeline/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![Docker Build](https://github.com/catherinevee/cloudrecon/workflows/Docker%20Build/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![Trivy Security Scan](https://github.com/catherinevee/cloudrecon/workflows/Trivy%20Security%20Scan/badge.svg)](https://github.com/catherinevee/cloudrecon/actions)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)]()

Multi-cloud infrastructure discovery and analysis in a single command.

## Overview

CloudRecon provides instant visibility into your multi-cloud infrastructure across AWS, Azure, and GCP. It discovers resources, analyzes dependencies, identifies security issues, and estimates costs with zero infrastructure requirements.

## Quick Start

### Prerequisites

* Go 1.21+
* Cloud credentials configured (AWS, Azure, or GCP)
* 2 minutes

### Install & Run (3 Commands)

```bash
# 1. Download binary
curl -L https://github.com/cloudrecon/cloudrecon/releases/latest/download/cloudrecon-linux-amd64 -o cloudrecon
chmod +x cloudrecon

# 2. Discover resources
./cloudrecon discover

# 3. Run analysis
./cloudrecon analyze
```

## Features

### Available Now

* **Multi-Cloud Discovery** - AWS, Azure, and GCP resource discovery
* **Zero Infrastructure** - Single binary, no databases, no agents required
* **Offline-First** - Local SQLite caching for instant queries
* **Interactive CLI** - Guided workflows with beautiful terminal UI
* **Comprehensive Analysis** - Dependencies, security, and cost analysis
* **Rich Export** - Multiple formats including HTML and PDF reports
* **Performance Optimized** - Parallel processing and intelligent caching

## Usage Examples

### Basic CLI Usage

```bash
# Discover all cloud resources
./cloudrecon discover

# Run comprehensive analysis
./cloudrecon analyze

# Interactive mode
./cloudrecon interactive

# Query specific resources
./cloudrecon query "SELECT * FROM resources WHERE provider='aws'"
```

### Interactive Mode

```bash
./cloudrecon interactive
```

Choose from guided workflows:
* **Quick Analysis** - Dependencies + Security + Cost
* **Dependency Analysis** - Resource relationships
* **Security Analysis** - Vulnerabilities and misconfigurations
* **Cost Analysis** - Spending optimization
* **Custom Analysis** - Configure your own analysis
* **Export Results** - Generate reports
* **View Cache Statistics** - Performance metrics

## Installation

### Pre-built Binaries

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

### Docker

```bash
# Pull the latest image
docker pull ghcr.io/cloudrecon/cloudrecon:latest

# Run CloudRecon
docker run --rm -v ~/.aws:/root/.aws -v ~/.azure:/root/.azure -v ~/.config/gcloud:/root/.config/gcloud ghcr.io/cloudrecon/cloudrecon:latest discover
```

### From Source

```bash
git clone https://github.com/cloudrecon/cloudrecon.git
cd cloudrecon
go build -o cloudrecon ./cmd/cloudrecon
```

## Contributing

Contributions are welcome in the following areas:

* Cloud provider integrations
* Analysis rule implementations
* Test coverage improvements
* Documentation updates
* Performance optimizations

See CONTRIBUTING.md for guidelines.

## License

MIT License - see LICENSE file for details.

## Support

* Issues: [GitHub Issues](https://github.com/cloudrecon/cloudrecon/issues)
* Discussions: [GitHub Discussions](https://github.com/cloudrecon/cloudrecon/discussions)
* Security: [Security Advisory](https://github.com/cloudrecon/cloudrecon/security/advisories)