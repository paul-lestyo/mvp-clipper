import socket
import msgpack
import numpy as np
import cv2
import onnxruntime as ort
import time
import os
import sys
import logging

logging.basicConfig(level=logging.INFO, format='[%(levelname)s] %(message)s')
logger = logging.getLogger(__name__)

class YuNetService:
    def __init__(self, model_path, socket_path):
        logger.info(f"Initializing YuNet service with model: {model_path}")
        
        # Load ONNX model
        self.session = ort.InferenceSession(
            model_path,
            providers=['CPUExecutionProvider']
        )
        
        # Configure CPU optimization
        session_options = self.session.get_session_options()
        
        logger.info("YuNet model loaded successfully")
        
        # Setup Unix socket
        self.socket_path = socket_path
        if os.path.exists(socket_path):
            os.remove(socket_path)
        
        self.socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.socket.bind(socket_path)
        self.socket.listen(5)
        
        logger.info(f"Listening on Unix socket: {socket_path}")
        
        # Warm up model
        self._warmup()
    
    def _warmup(self):
        """Warm up model with dummy inference to avoid first-call overhead"""
        logger.info("Warming up model...")
        dummy_frame = np.zeros((480, 640, 3), dtype=np.uint8)
        self.infer(dummy_frame)
        logger.info("Model warm-up complete")
    
    def preprocess(self, frame, orig_width, orig_height):
        """Preprocess frame for YuNet: resize to 320x320, normalize, NCHW"""
        # Resize to 320x320
        resized = cv2.resize(frame, (320, 320))
        
        # Normalize to [0, 1]
        normalized = resized.astype(np.float32) / 255.0
        
        # Convert to NCHW format (batch, channels, height, width)
        nchw = np.transpose(normalized, (2, 0, 1))
        batched = np.expand_dims(nchw, axis=0)
        
        # Calculate scale factors for coordinate mapping
        scale_x = orig_width / 320.0
        scale_y = orig_height / 320.0
        
        return batched, scale_x, scale_y
    
    def infer(self, frame):
        """Run YuNet inference on frame"""
        start = time.perf_counter()
        
        orig_height, orig_width = frame.shape[:2]
        
        # Preprocess
        input_tensor, scale_x, scale_y = self.preprocess(frame, orig_width, orig_height)
        
        # Run inference
        outputs = self.session.run(None, {'input': input_tensor})
        
        # Parse YuNet output: [N, 15]
        # Each detection: [x, y, w, h, conf, x_eye1, y_eye1, x_eye2, y_eye2, x_nose, y_nose, x_mouth1, y_mouth1, x_mouth2, y_mouth2]
        detections = []
        
        if len(outputs) > 0 and len(outputs[0]) > 0:
            for det in outputs[0]:
                conf = float(det[4])
                
                # Filter by confidence threshold (0.6)
                if conf >= 0.6:
                    # Scale coordinates back to original image size
                    x = float(det[0]) * scale_x
                    y = float(det[1]) * scale_y
                    w = float(det[2]) * scale_x
                    h = float(det[3]) * scale_y
                    
                    # Scale landmarks
                    landmarks = []
                    for i in range(5, 15, 2):
                        lx = float(det[i]) * scale_x
                        ly = float(det[i+1]) * scale_y
                        landmarks.extend([lx, ly])
                    
                    detections.append({
                        'x': x,
                        'y': y,
                        'w': w,
                        'h': h,
                        'c': conf,
                        'l': landmarks  # 10 values: [x1,y1, x2,y2, x3,y3, x4,y4, x5,y5]
                    })
        
        elapsed = (time.perf_counter() - start) * 1000
        
        return {
            'detections': detections,
            'inference_ms': elapsed
        }
    
    def serve(self):
        """Main service loop"""
        logger.info("Service ready, waiting for connections...")
        
        while True:
            try:
                conn, _ = self.socket.accept()
                
                try:
                    # Read request (max 1920*1080*3 + 1KB header)
                    data = b''
                    while True:
                        chunk = conn.recv(65536)
                        if not chunk:
                            break
                        data += chunk
                        if len(data) >= 8:  # Check if we have enough for msgpack header
                            try:
                                # Try to unpack to see if we have complete message
                                unpacker = msgpack.Unpacker(raw=False)
                                unpacker.feed(data)
                                request = next(unpacker)
                                break
                            except StopIteration:
                                continue
                    
                    if not data:
                        continue
                    
                    # Unpack request
                    request = msgpack.unpackb(data, raw=False)
                    
                    # Reconstruct frame from raw bytes
                    height = request['h']
                    width = request['w']
                    frame_data = request['d']
                    
                    frame = np.frombuffer(
                        frame_data,
                        dtype=np.uint8
                    ).reshape(height, width, 3)
                    
                    # Run inference
                    response = self.infer(frame)
                    
                    # Pack and send response
                    response_data = msgpack.packb(response)
                    conn.sendall(response_data)
                    
                    logger.debug(f"Processed frame {width}x{height}, found {len(response['detections'])} faces in {response['inference_ms']:.1f}ms")
                    
                except Exception as e:
                    logger.error(f"Error processing request: {e}")
                finally:
                    conn.close()
                    
            except KeyboardInterrupt:
                logger.info("Shutting down...")
                break
            except Exception as e:
                logger.error(f"Connection error: {e}")
        
        self.socket.close()
        if os.path.exists(self.socket_path):
            os.remove(self.socket_path)

def main():
    model_path = os.getenv('MODEL_PATH', '/app/models/yunet_320x320.onnx')
    socket_path = os.getenv('SOCKET_PATH', '/tmp/yunet.sock')
    
    if not os.path.exists(model_path):
        logger.error(f"Model not found: {model_path}")
        sys.exit(1)
    
    service = YuNetService(model_path, socket_path)
    service.serve()

if __name__ == '__main__':
    main()
