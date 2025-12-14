# Docker Setup for ONNX Runtime

This guide explains how to run the MVP Clipper application with ONNX Runtime face detection in Docker.

## Overview

The Dockerfile has been configured to automatically:
1. Download ONNX Runtime v1.22.0 for Linux
2. Download the YuNet face detection model
3. Install all dependencies in a multi-stage build
4. Copy only necessary files to the final image

## Quick Start

### 1. Build the Docker Image

```bash
docker-compose build
```

This will:
- Download ONNX Runtime library (`libonnxruntime.so.1.22.0`)
- Download YuNet model (`face_detection_yunet_2023mar.onnx`)
- Build your Go application
- Set up all runtime dependencies

### 2. Run the Application

```bash
docker-compose up
```

The application will be available at `http://localhost:8080`

## What's Included

### Multi-Stage Build

The Dockerfile uses a multi-stage build to optimize image size:

**Builder Stage:**
- Downloads ONNX Runtime for Linux
- Downloads YuNet face detection model
- Builds the Go application

**Final Stage:**
- Copies only the compiled binary
- Copies ONNX Runtime library
- Copies YuNet model
- Installs runtime dependencies (ffmpeg, yt-dlp, fonts)

### Automatic OS Detection

The code automatically detects the operating system and uses the correct library:
- **Linux (Docker)**: `libonnxruntime.so`
- **Windows**: `onnxruntime.dll`

## Testing Smart Crop

Once the container is running, test the smart crop feature:

```bash
curl -X POST http://localhost:8080/clip/generate \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://youtube.com/watch?v=VIDEO_ID",
    "start": "00:00:00",
    "end": "00:01:00",
    "smartCrop": true
  }'
```

## Troubleshooting

### Check if ONNX Runtime is loaded

View container logs:
```bash
docker-compose logs app
```

You should see:
```
YuNet face detection initialized
```

If you see a warning instead, check the following:

### Verify files exist in container

```bash
docker-compose exec app ls -la /usr/local/lib/libonnxruntime*
docker-compose exec app ls -la /app/models/
```

Expected output:
```
/usr/local/lib/libonnxruntime.so -> libonnxruntime.so.1.22.0
/usr/local/lib/libonnxruntime.so.1.22.0

/app/models/face_detection_yunet_2023mar.onnx
```

### Manual verification

Enter the container:
```bash
docker-compose exec app bash
```

Test library loading:
```bash
ldconfig -p | grep onnxruntime
```

Should show:
```
libonnxruntime.so (libc6,x86-64) => /usr/local/lib/libonnxruntime.so
```

## Rebuilding

If you make changes to the Dockerfile or need to re-download dependencies:

```bash
# Rebuild without cache
docker-compose build --no-cache

# Restart containers
docker-compose down
docker-compose up
```

## Volume Mounts

The `docker-compose.yml` mounts `./tmp` to persist generated clips:
```yaml
volumes:
  - ./tmp:/app/tmp
```

Generated clips will be available in your local `tmp/clips/` directory.

## Notes

- ONNX Runtime v1.22.0 is used to match the Go bindings version
- The YuNet model is ~300KB and downloaded from OpenCV Zoo
- Total ONNX Runtime library size is ~10MB
- Build time: ~2-3 minutes (first build)
- Subsequent builds use Docker layer caching for faster rebuilds
