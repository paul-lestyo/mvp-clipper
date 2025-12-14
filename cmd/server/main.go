package main

import (
	"log"
	"mvp-clipper/internal/api"
	"mvp-clipper/internal/config"
	"mvp-clipper/internal/services/face"
)

func main() {
	cfg := config.Load()

	// Initialize YuNet face detection model
	// Note: Run setup_smart_crop.ps1 first to download model and ONNX Runtime
	err := face.InitYuNet("models/face_detection_yunet_2023mar.onnx")
	if err != nil {
		log.Printf("Warning: Failed to initialize YuNet (smart crop disabled): %v", err)
	} else {
		defer face.Cleanup()
		log.Println("YuNet face detection initialized")
	}

	server := api.NewServer(cfg)

	log.Println("Server running on http://localhost:" + cfg.Port)
	err = server.Listen(":" + cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
