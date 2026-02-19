# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /capfox ./cmd/capfox

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S capfox && adduser -S capfox -G capfox

# Create directories
RUN mkdir -p /app/data /app/configs && \
    chown -R capfox:capfox /app

# Copy binary from builder
COPY --from=builder /capfox /app/capfox

# Copy example config
COPY --chown=capfox:capfox configs/capfox.example.yaml /app/configs/capfox.yaml

# Switch to non-root user
USER capfox

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["/app/capfox"]
CMD ["start", "--config", "/app/configs/capfox.yaml"]
