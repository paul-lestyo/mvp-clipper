package main

import (
	"log"
	"mvp-clipper/internal/api"
	"mvp-clipper/internal/config"
	"mvp-clipper/internal/services/face"
)

func main() {
	cfg := config.Load()

	// Initialize Pigo face detection model (pure Go, no CGO required)
	err := face.InitPigo("models/facefinder")
	if err != nil {
		log.Printf("Warning: Failed to initialize Pigo (smart crop disabled): %v", err)
	} else {
		defer face.Cleanup()
		log.Println("Pigo face detection initialized")
	}

	server := api.NewServer(cfg)

	log.Println("Server running on http://localhost:" + cfg.Port)
	err = server.Listen(":" + cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
