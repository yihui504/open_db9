# Multi-stage build for optimized Docker image
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o db9 ./cmd/db9
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o db9-server ./cmd/server

# Final stage - minimal image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    postgresql-client \
    bash \
    curl \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 db9 && \
    adduser -D -u 1000 -G db9 db9

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/db9 .
COPY --from=builder /app/db9-server .

# Copy migrations
COPY migrations ./migrations

# Change ownership
RUN chown -R db9:db9 /app

# Switch to non-root user
USER db9

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["./db9-server"]
