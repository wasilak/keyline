# Build stage
FROM golang:1.26-alpine AS builder

# Build arguments
ARG VERSION=dev

# Install build dependencies
RUN apk add --no-cache git make zip

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o keyline ./cmd/keyline

# Test the binary
RUN ./keyline --version || true

# Runtime stage
FROM alpine:3.23

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 keyline && \
    adduser -D -u 1000 -G keyline keyline

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/keyline /app/keyline

# Copy example configuration
COPY --from=builder /build/config/config.example.yaml /app/config.example.yaml

# Change ownership
RUN chown -R keyline:keyline /app

# Switch to non-root user
USER keyline

# Expose port
EXPOSE 9000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9000/_health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/keyline"]

# Default command (can be overridden)
CMD ["--config", "/etc/keyline/config.yaml"]
