# Build stage
FROM golang:1.24-alpine AS builder

# Install build tools
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies with verbose output
RUN go mod download -x

# Copy source code
COPY . .

# Build the application with verbose output
RUN CGO_ENABLED=0 GOOS=linux go build -v -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port 8081
EXPOSE 8081

# Set environment variable for the port
ENV PORT=8081

# Run the binary
CMD ["./main"] 