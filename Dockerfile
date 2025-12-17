# syntax=docker/dockerfile:1.4
# ============================================
# Builder Stage - Build Go binary
# ============================================
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install minimal build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    wget

# Download Pigo cascade file (facefinder)
RUN mkdir -p models && \
    wget -q -O models/facefinder \
    https://github.com/esimov/pigo/raw/master/cascade/facefinder

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build binary (pure Go, no CGO needed)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags='-w -s' \
    -o server ./cmd/server

# ============================================
# Final Stage - Alpine for minimal size
# ============================================
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip \
    font-liberation \
    fontconfig \
    curl \
    && pip3 install --no-cache-dir --break-system-packages yt-dlp \
    && fc-cache -f -v \
    && apk del py3-pip \
    && rm -rf /var/cache/apk/* /tmp/* /root/.cache

# Copy Pigo cascade from builder
COPY --from=builder /app/models /app/models

# Copy built binary
COPY --from=builder /app/server /app/server

# Create tmp directory
RUN mkdir -p /app/tmp

# Performance environment variables (Go only, no OpenCV)
ENV GOMEMLIMIT=1800MiB \
    GOGC=100

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run server
CMD ["./server"]

