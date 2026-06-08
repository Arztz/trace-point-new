# Build stage
FROM golang:1.23 AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./

# Enable auto toolchain download
ENV GOTOOLCHAIN=auto
ENV GOFLAGS=-mod=mod

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./cmd/server/

# Runtime stage - use Debian for glibc compatibility
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/.env .

# Create data directory
RUN mkdir -p /app/data

EXPOSE 8088

CMD ["./server"]