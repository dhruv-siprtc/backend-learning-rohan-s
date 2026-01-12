# Dockerfile (single file for both API and Consumer)
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy .env file (optional, can use docker-compose env)
COPY --from=builder /app/.env .

# Make binary executable
RUN chmod +x ./main

# Expose port (only used by API)
EXPOSE 8080

# Default command (can be overridden in docker-compose)
CMD ["./main", "-mode", "api"]