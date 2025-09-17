#!/bin/bash

# CloudRecon Build Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build configuration
VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

# Output directory
OUTPUT_DIR="dist"
mkdir -p ${OUTPUT_DIR}

echo -e "${GREEN}Building CloudRecon v${VERSION}${NC}"
echo -e "${YELLOW}Build time: ${BUILD_TIME}${NC}"
echo -e "${YELLOW}Git commit: ${GIT_COMMIT}${NC}"

# Build for multiple platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "$platform"
    
    echo -e "${YELLOW}Building for ${os}/${arch}...${NC}"
    
    # Set output name
    output_name="cloudrecon"
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    # Build the binary
    GOOS=$os GOARCH=$arch go build \
        -ldflags "$LDFLAGS" \
        -o "${OUTPUT_DIR}/cloudrecon-${os}-${arch}/${output_name}" \
        ./cmd/cloudrecon
    
    # Create archive
    cd ${OUTPUT_DIR}
    if [ "$os" = "windows" ]; then
        zip -r "cloudrecon-${os}-${arch}.zip" "cloudrecon-${os}-${arch}/"
    else
        tar -czf "cloudrecon-${os}-${arch}.tar.gz" "cloudrecon-${os}-${arch}/"
    fi
    cd ..
    
    echo -e "${GREEN}✓ Built for ${os}/${arch}${NC}"
done

# Build Docker image
echo -e "${YELLOW}Building Docker image...${NC}"
docker build -t cloudrecon:${VERSION} .
docker tag cloudrecon:${VERSION} cloudrecon:latest

echo -e "${GREEN}✓ Docker image built${NC}"

# Create checksums
echo -e "${YELLOW}Creating checksums...${NC}"
cd ${OUTPUT_DIR}
sha256sum *.tar.gz *.zip > checksums.txt
cd ..

echo -e "${GREEN}✓ Checksums created${NC}"

# Show build summary
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${YELLOW}Output directory: ${OUTPUT_DIR}${NC}"
echo -e "${YELLOW}Files created:${NC}"
ls -la ${OUTPUT_DIR}/

echo -e "${GREEN}CloudRecon v${VERSION} is ready for distribution!${NC}"
