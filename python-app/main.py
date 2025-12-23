import cv2
import mediapipe as mp
import json
import os
from flask import Flask, request, jsonify

app = Flask(__name__)

# Konfigurasi MediaPipe
mp_face_detection = mp.solutions.face_detection
mp_face_mesh = mp.solutions.face_mesh

def analyze_video(video_path):
    # Gunakan Short Range untuk selfie/podcast dekat
    face_detection = mp_face_detection.FaceDetection(model_selection=1, min_detection_confidence=0.5)
    
    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        return []

    width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
    height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
    
    timestamps = []
    
    # --- STABILIZATION VARIABLES ---
    last_centers = None   # For split/group
    
    # Config
    ALPHA = 0.1          # Smoothing factor
    DEAD_ZONE = 0.05     # 5% of screen width. If movement is less than this, don't move.
    SPLIT_THRESHOLD = 0.25 # If faces are > 25% width apart, split.

    frame_count = 0
    while cap.isOpened():
        success, image = cap.read()
        if not success:
            break

        if frame_count % 5 != 0:
            frame_count += 1
            continue

        image.flags.writeable = False
        results = face_detection.process(cv2.cvtColor(image, cv2.COLOR_BGR2RGB))

        current_centers = []
        mode = "center"

        if results.detections:
            # Collect all valid faces
            faces = []
            for detection in results.detections:
                bboxC = detection.location_data.relative_bounding_box
                w_px = bboxC.width * width
                h_px = bboxC.height * height
                area = w_px * h_px
                
                # Filter small faces (background noise)
                if area < (width * height * 0.01): # < 1% of screen
                    continue
                
                center_x = (bboxC.xmin + bboxC.width / 2) * width
                faces.append({
                    "center": center_x,
                    "area": area,
                    "bbox": bboxC
                })
            
            # Sort by area (largest first)
            faces.sort(key=lambda x: x["area"], reverse=True)
            
            # Logic: Single vs Multi
            if len(faces) >= 2:
                primary = faces[0]
                secondary = faces[1]
                
                # Check distance
                dist = abs(primary["center"] - secondary["center"])
                dist_ratio = dist / width
                
                if dist_ratio > SPLIT_THRESHOLD:
                    mode = "split"
                    # Order left to right
                    if primary["center"] < secondary["center"]:
                        current_centers = [primary["center"], secondary["center"]]
                    else:
                        current_centers = [secondary["center"], primary["center"]]
                else:
                    # Group mode (close together) - use average center
                    avg_center = (primary["center"] + secondary["center"]) / 2
                    current_centers = [avg_center]
            elif len(faces) == 1:
                current_centers = [faces[0]["center"]]
            # else: no faces
        
        # --- STABILIZATION LOGIC ---
        final_centers = []
        
        # Initialize if needed
        if last_centers is None and current_centers:
            last_centers = current_centers
        elif current_centers:
            # If mode changed or number of people changed, reset hard (snap)
            if len(current_centers) != len(last_centers):
                # Only use last centers if we are still in same mode approx? 
                # Simpler: Snap if topology changes.
                last_centers = current_centers
            else:
                # Apply smoothing with dead zone per center
                new_smoothed = []
                for i, curr in enumerate(current_centers):
                    last = last_centers[i]
                    diff = abs(curr - last)
                    
                    # Dead Zone Check
                    if diff < (width * DEAD_ZONE):
                        # Within dead zone: keep last position (stable)
                        new_smoothed.append(last)
                    else:
                        # Outside dead zone: Smooth move
                        # But we smooth towards the current.
                        val = (ALPHA * curr) + ((1 - ALPHA) * last)
                        new_smoothed.append(val)
                last_centers = new_smoothed
        
        # Handle lost tracking
        if not current_centers:
            if last_centers:
                 # Keep holding last known position
                 pass 
            else:
                # Default to center
                last_centers = [width / 2]
        
        # Final formatting
        final_centers = [int(c) for c in last_centers]

        timestamps.append({
            "frame": frame_count,
            "mode": mode,
            "centers": final_centers
        })
        
        frame_count += 1

    cap.release()
    return timestamps

@app.route('/process', methods=['POST'])
def process_video():
    data = request.json
    filename = data.get('filename')
    if not filename:
        return jsonify({"error": "No filename provided"}), 400
    
    # Assuming video is in the mapped shared volume
    storage_path = os.environ.get('VIDEO_STORAGE_PATH', '/app/shared')
    video_path = os.path.join(storage_path, filename)
    
    print(f"Processing video: {video_path}")
    
    if not os.path.exists(video_path):
        return jsonify({"error": f"File not found: {video_path}"}), 404

    try:
        results = analyze_video(video_path)
        return jsonify(results)
    except Exception as e:
        print(f"Error: {e}")
        return jsonify({"error": str(e)}), 500

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=5000)
