# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build

# Backend build stage
FROM golang:1.24.7-alpine AS backend-builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from frontend-builder to where embed.go expects it
COPY --from=frontend-builder /app/web/dist ./pkg/web/dist

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o owuiback cmd/owuiback/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS and age for encryption
RUN apk --no-cache add ca-certificates age

WORKDIR /root/

# Copy binary from builder
COPY --from=backend-builder /app/owuiback .

# Set entrypoint
ENTRYPOINT ["./owuiback"]
