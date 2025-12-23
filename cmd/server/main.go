package main

import (
	"log"
	"mvp-clipper/internal/api"
	"mvp-clipper/internal/config"
	"mvp-clipper/internal/services/face"
)

func main() {
	cfg := config.Load()

	// Initialize YuNet client (connects to Python service via Unix socket)
	socketPath := "/tmp/yunet.sock"
	err := face.InitYuNet(socketPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize YuNet (smart crop disabled): %v", err)
	} else {
		defer face.Cleanup()
		log.Println("YuNet client initialized successfully")
	}

	server := api.NewServer(cfg)

	log.Println("Server running on http://localhost:" + cfg.Port)
	err = server.Listen(":" + cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
