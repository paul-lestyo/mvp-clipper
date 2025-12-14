# Smart Crop Feature Setup

## Prerequisites

Before using the smart crop feature, you need to download and set up the following dependencies:

### 1. YuNet ONNX Model

Download the YuNet face detection model from OpenCV Zoo:

```bash
# Create models directory (already created)
# Download YuNet model
curl -L -o models/face_detection_yunet_2023mar.onnx https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx
```

Or download manually from:
https://github.com/opencv/opencv_zoo/tree/main/models/face_detection_yunet

### 2. ONNX Runtime Library

Download ONNX Runtime v1.22.0 for your platform:

#### Windows
1. Download from: https://github.com/microsoft/onnxruntime/releases/tag/v1.22.0
2. Extract `onnxruntime.dll` from the archive
3. Place it in the project root directory or add to PATH

Direct link:
```
https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-win-x64-1.22.0.zip
```

#### Linux
1. Download from: https://github.com/microsoft/onnxruntime/releases/tag/v1.22.0
2. Extract `libonnxruntime.so.1.22.0`
3. Place it in `/usr/local/lib` or update `LD_LIBRARY_PATH`

Direct link:
```
https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-linux-x64-1.22.0.tgz
```

### 3. Initialize YuNet on Startup

Before using the smart crop feature, initialize the YuNet model in your application startup code (e.g., `cmd/server/main.go`):

```go
import "mvp-clipper/internal/services/face"

func main() {
    // Initialize YuNet model
    err := face.InitYuNet("models/face_detection_yunet_2023mar.onnx")
    if err != nil {
        log.Fatalf("Failed to initialize YuNet: %v", err)
    }
    defer face.Cleanup()
    
    // ... rest of your application code
}
```

## Usage

### API Endpoint

Use the `/clip/generate` endpoint with the `smartCrop` parameter:

```json
{
  "url": "https://youtube.com/watch?v=VIDEO_ID",
  "start": "00:00:00",
  "end": "00:01:00",
  "smartCrop": true,
  "caption": false
}
```

### How It Works

1. **Frame Extraction**: Extracts frames at 2 fps from the video clip
2. **Face Detection**: Uses YuNet ONNX model to detect faces in each frame
3. **Mode Detection**: Determines "center" or "split" mode based on face positions
   - 0-1 faces → center mode
   - 2+ faces horizontally separated → split mode
   - 2+ faces in center → center mode
4. **Timeline Compression**: Filters timeline to keep only mode changes
5. **Dynamic Cropping**: Applies FFmpeg filters to crop video dynamically based on timeline

### Example Timeline

```
Input frames:  [0s:center, 0.5s:center, 1.0s:split, 1.5s:split, 2.0s:center]
Compressed:    [0s:center, 1.0s:split, 2.0s:center]
Result:        Video crops to center for 0-1s, split for 1-2s, center for 2s+
```

## Troubleshooting

### ONNX Runtime Library Not Found

If you get an error about `onnxruntime.dll` or `libonnxruntime.so` not found:

1. Verify the library is in the correct location
2. Update the path in `internal/services/face/yunet.go` line 33:
   ```go
   ort.SetSharedLibraryPath("path/to/onnxruntime.dll")  // Windows
   ort.SetSharedLibraryPath("path/to/libonnxruntime.so.1.22.0")  // Linux
   ```

### YuNet Model Not Found

Ensure the model file exists at `models/face_detection_yunet_2023mar.onnx` relative to your working directory.

### FFmpeg Errors

The dynamic cropping feature requires FFmpeg with filter support. Ensure FFmpeg is installed and accessible in PATH.
