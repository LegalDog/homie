# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o homie .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create data directory
RUN mkdir -p /app/data

# Copy binary from builder
COPY --from=builder /build/homie .

# Expose port
EXPOSE 8080

# Volume for persistent data
VOLUME ["/app/data"]

# Run
CMD ["./homie"]
