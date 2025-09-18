#!/bin/bash

# CloudRecon Installation Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR=${INSTALL_DIR:-"/usr/local/bin"}
CONFIG_DIR=${CONFIG_DIR:-"$HOME/.cloudrecon"}
VERSION=${VERSION:-"latest"}

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}CloudRecon Installation Script${NC}"
echo -e "${YELLOW}OS: $OS${NC}"
echo -e "${YELLOW}Architecture: $ARCH${NC}"
echo -e "${YELLOW}Install directory: $INSTALL_DIR${NC}"
echo -e "${YELLOW}Config directory: $CONFIG_DIR${NC}"

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${YELLOW}Running as root - installing system-wide${NC}"
else
    echo -e "${YELLOW}Running as user - installing to user directory${NC}"
    INSTALL_DIR="$HOME/.local/bin"
fi

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download and install binary
echo -e "${YELLOW}Downloading CloudRecon...${NC}"

if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/cloudrecon/cloudrecon/releases/latest/download/cloudrecon-${OS}-${ARCH}.tar.gz"
else
    DOWNLOAD_URL="https://github.com/cloudrecon/cloudrecon/releases/download/v${VERSION}/cloudrecon-${OS}-${ARCH}.tar.gz"
fi

# Download binary
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if command -v curl >/dev/null 2>&1; then
    curl -L -o "cloudrecon.tar.gz" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -O "cloudrecon.tar.gz" "$DOWNLOAD_URL"
else
    echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
fi

# Extract and install
echo -e "${YELLOW}Installing CloudRecon...${NC}"
tar -xzf "cloudrecon.tar.gz"
cp "cloudrecon-${OS}-${ARCH}/cloudrecon" "$INSTALL_DIR/cloudrecon"
chmod +x "$INSTALL_DIR/cloudrecon"

# Clean up
cd /
rm -rf "$TEMP_DIR"

# Create config directory
mkdir -p "$CONFIG_DIR"

# Create default config if it doesn't exist
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo -e "${YELLOW}Creating default configuration...${NC}"
    cat > "$CONFIG_DIR/config.yaml" << 'EOF'
# CloudRecon Configuration File

# Storage configuration
storage:
  database_path: "cloudrecon.db"
  cache_size: 1000

# AWS configuration
aws:
  regions:
    - "us-east-1"
    - "us-west-2"
    - "eu-west-1"
    - "ap-southeast-1"
  max_retries: 3
  timeout: "30s"

# Azure configuration
azure:
  subscriptions: []  # Empty means discover all accessible subscriptions
  max_retries: 3
  timeout: "30s"

# GCP configuration
gcp:
  projects: []  # Empty means discover all accessible projects
  max_retries: 3
  timeout: "30s"

# Discovery configuration
discovery:
  default_mode: "standard"  # quick, standard, deep
  use_native_tools: true
  max_parallel: 10
  timeout: "30m"

# Query configuration
query:
  cache_ttl: "5m"
  max_results: 10000

# Export configuration
export:
  default_format: "json"
  output_directory: "./exports"

# Logging configuration
logging:
  level: "info"  # debug, info, warn, error
  format: "text"  # text, json
  output: "stdout"  # stdout, stderr, file
EOF
fi

# Add to PATH if not already there
if [ "$EUID" -eq 0 ]; then
    # System-wide installation
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> /etc/profile
    fi
else
    # User installation
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$HOME/.bashrc"
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$HOME/.zshrc" 2>/dev/null || true
    fi
fi

# Verify installation
echo -e "${YELLOW}Verifying installation...${NC}"
if "$INSTALL_DIR/cloudrecon" --version >/dev/null 2>&1; then
    echo -e "${GREEN}✓ CloudRecon installed successfully!${NC}"
    echo -e "${GREEN}Version: $("$INSTALL_DIR/cloudrecon" --version)${NC}"
else
    echo -e "${RED}✗ Installation verification failed${NC}"
    exit 1
fi

# Show next steps
echo -e "${BLUE}Next steps:${NC}"
echo -e "${YELLOW}1. Configure your cloud credentials:${NC}"
echo -e "   AWS: aws configure"
echo -e "   Azure: az login"
echo -e "   GCP: gcloud auth login"
echo -e ""
echo -e "${YELLOW}2. Run your first discovery:${NC}"
echo -e "   cloudrecon discover"
echo -e ""
echo -e "${YELLOW}3. Query your resources:${NC}"
echo -e "   cloudrecon query \"SELECT * FROM resources LIMIT 10\""
echo -e ""
echo -e "${YELLOW}4. Export your data:${NC}"
echo -e "   cloudrecon export --format json --output resources.json"
echo -e ""
echo -e "${GREEN}Happy cloud discovering!${NC}"
