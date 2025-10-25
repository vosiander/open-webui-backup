# Build stage
FROM golang:1.24.7-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o owuiback cmd/owuiback/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS and age for encryption
RUN apk --no-cache add ca-certificates age

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/owuiback .

# Set entrypoint
ENTRYPOINT ["./owuiback"]
