# CloudRecon Build Script for Windows

param(
    [string]$Version = "1.0.0"
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"

# Build configuration
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$GitCommit = try { (git rev-parse --short HEAD) } catch { "unknown" }
$LDFlags = "-X main.version=$Version -X main.buildTime=$BuildTime -X main.gitCommit=$GitCommit"

# Output directory
$OutputDir = "dist"
if (Test-Path $OutputDir) {
    Remove-Item $OutputDir -Recurse -Force
}
New-Item -ItemType Directory -Path $OutputDir | Out-Null

Write-Host "Building CloudRecon v$Version" -ForegroundColor $Green
Write-Host "Build time: $BuildTime" -ForegroundColor $Yellow
Write-Host "Git commit: $GitCommit" -ForegroundColor $Yellow

# Build for multiple platforms
$Platforms = @(
    @{OS="linux"; Arch="amd64"},
    @{OS="linux"; Arch="arm64"},
    @{OS="windows"; Arch="amd64"},
    @{OS="windows"; Arch="arm64"},
    @{OS="darwin"; Arch="amd64"},
    @{OS="darwin"; Arch="arm64"}
)

foreach ($platform in $Platforms) {
    $os = $platform.OS
    $arch = $platform.Arch
    
    Write-Host "Building for $os/$arch..." -ForegroundColor $Yellow
    
    # Set output name
    $outputName = "cloudrecon"
    if ($os -eq "windows") {
        $outputName = "$outputName.exe"
    }
    
    # Create platform directory
    $platformDir = "$OutputDir/cloudrecon-$os-$arch"
    New-Item -ItemType Directory -Path $platformDir | Out-Null
    
    # Build the binary
    $env:GOOS = $os
    $env:GOARCH = $arch
    go build -ldflags $LDFlags -o "$platformDir/$outputName" ./cmd/cloudrecon
    
    # Create archive
    if ($os -eq "windows") {
        Compress-Archive -Path $platformDir -DestinationPath "$OutputDir/cloudrecon-$os-$arch.zip"
    } else {
        # Use tar if available, otherwise create zip
        try {
            tar -czf "$OutputDir/cloudrecon-$os-$arch.tar.gz" -C $OutputDir "cloudrecon-$os-$arch"
        } catch {
            Compress-Archive -Path $platformDir -DestinationPath "$OutputDir/cloudrecon-$os-$arch.zip"
        }
    }
    
    Write-Host "✓ Built for $os/$arch" -ForegroundColor $Green
}

# Build Docker image
Write-Host "Building Docker image..." -ForegroundColor $Yellow
docker build -t "cloudrecon:$Version" .
docker tag "cloudrecon:$Version" "cloudrecon:latest"

Write-Host "✓ Docker image built" -ForegroundColor $Green

# Create checksums
Write-Host "Creating checksums..." -ForegroundColor $Yellow
$checksumFile = "$OutputDir/checksums.txt"
Get-ChildItem $OutputDir -Filter "*.tar.gz" | ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash
    "$hash  $($_.Name)" | Add-Content $checksumFile
}
Get-ChildItem $OutputDir -Filter "*.zip" | ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash
    "$hash  $($_.Name)" | Add-Content $checksumFile
}

Write-Host "✓ Checksums created" -ForegroundColor $Green

# Show build summary
Write-Host "Build completed successfully!" -ForegroundColor $Green
Write-Host "Output directory: $OutputDir" -ForegroundColor $Yellow
Write-Host "Files created:" -ForegroundColor $Yellow
Get-ChildItem $OutputDir | Format-Table Name, Length, LastWriteTime

Write-Host "CloudRecon v$Version is ready for distribution!" -ForegroundColor $Green
