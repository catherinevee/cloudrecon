# Multi-stage Dockerfile for CloudRecon
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o cloudrecon \
    ./cmd/cloudrecon

# Final stage - minimal image
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S cloudrecon && \
    adduser -u 1001 -S cloudrecon -G cloudrecon

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/cloudrecon /app/cloudrecon

# Copy configuration files
COPY --from=builder /app/config/ /app/config/

# Change ownership to non-root user
RUN chown -R cloudrecon:cloudrecon /app

# Switch to non-root user
USER cloudrecon

# Expose port (if needed for future web interface)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /app/cloudrecon --version || exit 1

# Default command
ENTRYPOINT ["/app/cloudrecon"]
CMD ["--help"]