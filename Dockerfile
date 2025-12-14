# Dockerfile for reverse-proxy application

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o reverse-proxy ./cmd/app

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/reverse-proxy .

# Copy configuration files (excluding sample directory)
COPY config/ config/

# Copy certificates (if needed)
COPY certs/ certs/

# Expose ports
EXPOSE 80 443

# Command to run the application
CMD ["./reverse-proxy"]