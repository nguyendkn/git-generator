# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (required for go mod)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o git-generator cmd/git-generator/main.go

# Final stage
FROM alpine:latest

# Install git and ca-certificates
RUN apk --no-cache add git ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh gitgen

# Set working directory
WORKDIR /home/gitgen

# Copy binary from builder stage
COPY --from=builder /app/git-generator /usr/local/bin/git-generator

# Make binary executable
RUN chmod +x /usr/local/bin/git-generator

# Switch to non-root user
USER gitgen

# Set entrypoint
ENTRYPOINT ["git-generator"]

# Default command
CMD ["--help"]
