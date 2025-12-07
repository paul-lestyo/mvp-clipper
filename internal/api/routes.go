package api

import (
	"github.com/gofiber/fiber/v2"
	"mvp-clipper/internal/api/handlers"
	"mvp-clipper/internal/config"
)

func NewServer(cfg *config.Config) *fiber.App {
	app := fiber.New()

	handlers.RegisterHealthRoutes(app)
	handlers.RegisterClipRoutes(app)

	return app
}
