# Download YuNet model
Write-Output "Downloading YuNet ONNX model..."
$yunetUrl = "https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx"
$yunetPath = "models/face_detection_yunet_2023mar.onnx"

Invoke-WebRequest -Uri $yunetUrl -OutFile $yunetPath
Write-Output "YuNet model downloaded to $yunetPath"

# Download ONNX Runtime for Windows
Write-Output "Downloading ONNX Runtime..."
$onnxUrl = "https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-win-x64-1.22.0.zip"
$onnxZip = "onnxruntime.zip"

Invoke-WebRequest -Uri $onnxUrl -OutFile $onnxZip
Write-Output "Extracting ONNX Runtime..."

Expand-Archive -Path $onnxZip -DestinationPath "temp_onnx" -Force
Copy-Item "temp_onnx/onnxruntime-win-x64-1.22.0/lib/onnxruntime.dll" -Destination "." -Force

Remove-Item $onnxZip
Remove-Item "temp_onnx" -Recurse -Force

Write-Output "Setup complete!"
Write-Output "- YuNet model: $yunetPath"
Write-Output "- ONNX Runtime: onnxruntime.dll"
