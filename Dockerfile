# ============================================
# Builder Stage - Build Go binary and download dependencies
# ============================================
FROM golang:1.22-bullseye AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    wget \
    tar \
    && rm -rf /var/lib/apt/lists/*

# Download ONNX Runtime for Linux (glibc)
RUN wget https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-linux-x64-1.22.0.tgz \
    && tar -xzf onnxruntime-linux-x64-1.22.0.tgz \
    && mv onnxruntime-linux-x64-1.22.0/lib/libonnxruntime.so.1.22.0 /usr/local/lib/ \
    && rm -rf onnxruntime-linux-x64-1.22.0.tgz onnxruntime-linux-x64-1.22.0

# Download YuNet face detection model
RUN mkdir -p models \
    && wget -O models/face_detection_yunet_2023mar.onnx \
    https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary with CGO enabled (required for ONNX Runtime)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags='-w -s' \
    -o server ./cmd/server

# ============================================
# Final Stage - Debian Slim for glibc compatibility with ONNX Runtime
# ============================================
FROM debian:12-slim

WORKDIR /app

# Install runtime dependencies only
RUN apt-get update && apt-get install -y \
    # Video processing
    ffmpeg \
    # Python for yt-dlp
    python3 \
    python3-pip \
    # Font support
    fonts-liberation \
    fontconfig \
    # Required for ONNX Runtime
    libgomp1 \
    && pip3 install --no-cache-dir --break-system-packages yt-dlp \
    && fc-cache -f -v \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy ONNX Runtime library from builder
COPY --from=builder /usr/local/lib/libonnxruntime.so.1.22.0 /usr/local/lib/
RUN ln -s /usr/local/lib/libonnxruntime.so.1.22.0 /usr/local/lib/libonnxruntime.so \
    && ldconfig /usr/local/lib || true

# Copy YuNet model from builder
COPY --from=builder /app/models /app/models

# Copy built binary from builder
COPY --from=builder /app/server /app/server

# Create tmp directory for temporary files
RUN mkdir -p /app/tmp

# Set environment variables for library paths
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# Expose port (adjust if needed)
EXPOSE 8080

# Run the server
CMD ["./server"]

