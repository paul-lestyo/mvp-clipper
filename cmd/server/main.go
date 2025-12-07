package main

import (
	"log"
	"mvp-clipper/internal/api"
	"mvp-clipper/internal/config"
)

func main() {
	cfg := config.Load()

	server := api.NewServer(cfg)

	log.Println("Server running on http://localhost:" + cfg.Port)
	err := server.Listen(":" + cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
