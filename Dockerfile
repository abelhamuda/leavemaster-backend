# 🏗️ Stage 1: Build the Go binary
FROM golang:1.25.1 AS builder

# Set working directory
WORKDIR /app

# Copy dependency files first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the binary
RUN go build -o main .

# 🧩 Stage 2: Create lightweight production image
FROM debian:bookworm-slim

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Expose the app port
EXPOSE 8080

# Run the app
CMD ["./main"]
