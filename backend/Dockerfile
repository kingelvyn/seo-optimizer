# Build stage
FROM golang:1.21-alpine AS builder

# Install build tools and debugging utilities
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Show environment and download dependencies with debug output
RUN pwd && \
    ls -la && \
    go version && \
    go env && \
    go mod download -x > /tmp/build.log 2>&1

# Copy source code
COPY . .

# Build with extensive debugging
RUN echo "=== Building Application ===" >> /tmp/build.log 2>&1 && \
    pwd >> /tmp/build.log 2>&1 && \
    ls -la >> /tmp/build.log 2>&1 && \
    go list -m all >> /tmp/build.log 2>&1 && \
    CGO_ENABLED=0 GOOS=linux go build -v -x -o main . >> /tmp/build.log 2>&1 || \
    (cat /tmp/build.log && exit 1)

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Create a directory for statistics
RUN mkdir -p /app/data

# Expose port 8081
EXPOSE 8081

# Set environment variables with defaults
ENV PORT=8081 \
    DEV_MODE=false \
    GIN_MODE=release

# Run the binary
CMD ["./main"] 