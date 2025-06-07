# Multi-stage build for multistage_pipeline_fanout

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/build/multistage_pipeline_fanout .

# Create a non-root user to run the application
RUN adduser -D -h /app appuser
USER appuser

# Command to run the application
ENTRYPOINT ["./multistage_pipeline_fanout"]