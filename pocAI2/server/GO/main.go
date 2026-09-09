package main

import (
	"context"
	controllers "golang-dh/controller"
	"golang-dh/middleware"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func setupRoutes(app *fiber.App) {
	auth := middleware.JWTAuth()

	// 1. Auth endpoints (public & protected) -> users table
	app.Post("/api/auth/register", controllers.Register)
	app.Post("/api/auth/login", controllers.Login)
	app.Get("/api/perusahaan", controllers.GetPerusahaan)

	api := app.Group("/api")
	api.Get("/auth/me", auth, controllers.GetMe)
	api.Get("/assets", controllers.GetAssets)
	api.Get("/assets/options", controllers.GetAssetOptions)
	api.Get("/assets/download", controllers.DownloadAssets)
	api.Post("/assets", controllers.CreateAssetManual)
	api.Patch("/assets/:id", controllers.UpdateAsset)
	api.Delete("/assets/:id", controllers.DeleteAsset)
	api.Post("/pdf", controllers.PdfHandler)
	api.Post("/csv", controllers.CsvHandler)
	api.Post("/excel", controllers.ExcelHandler)
	api.Post("/Tools", auth, controllers.CreateToolsHistory)
	api.Post("/schedule", controllers.CreateSchedule)
	api.Get("/schedule", controllers.GetSchedules)
	api.Delete("/schedule/:id", controllers.DeleteSchedule)
	app.Post("/schedule", controllers.CreateSchedule)
	app.Get("/schedule", controllers.GetSchedules)
	app.Post("/message", auth, controllers.SendMessage)
	app.Post("/message/audio", auth, controllers.SendAudioMessage)
	app.Post("/transcribe", controllers.TranscribeAudioHandler)
	app.Post("/tts", controllers.TTSHandler)
	app.Get("/tts", controllers.TTSHandler)
	api.Post("/tts", controllers.TTSHandler)
	api.Get("/tts", controllers.TTSHandler)
	app.Get("/messages/:id", auth, controllers.GetMessages)
	app.Get("/ws/:id", middleware.WebSocketMiddleware, websocket.New(controllers.SendMessages))
}

func main() {
	controllers.Connect()
	_ = controllers.EnsureScheduleTable(context.Background())

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())

	setupRoutes(app)
	app.Static("/uploads", "./uploads")
	log.Fatal(app.Listen("0.0.0.0:3000"))
}
