package main

import (
	"context"
	"log"
	"strconv"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	controllers "golang-dh/controller"
	"golang-dh/internal/modules/asset"
	assetRepo "golang-dh/internal/modules/asset/repository"
	assetSvc "golang-dh/internal/modules/asset/service"
	authMod "golang-dh/internal/modules/auth"
	authDto "golang-dh/internal/modules/auth/dto"
	authRepo "golang-dh/internal/modules/auth/repository"
	authSvc "golang-dh/internal/modules/auth/service"
	"golang-dh/internal/modules/chat"
	chatRepo "golang-dh/internal/modules/chat/repository"
	chatSvc "golang-dh/internal/modules/chat/service"
	"golang-dh/internal/modules/company"
	companyRepo "golang-dh/internal/modules/company/repository"
	companySvc "golang-dh/internal/modules/company/service"
	"golang-dh/internal/modules/schedule"
	scheduleDto "golang-dh/internal/modules/schedule/dto"
	scheduleRepo "golang-dh/internal/modules/schedule/repository"
	scheduleSvc "golang-dh/internal/modules/schedule/service"
	"golang-dh/internal/modules/tools"
	toolsRepo "golang-dh/internal/modules/tools/repository"
	toolsSvc "golang-dh/internal/modules/tools/service"
	"golang-dh/middleware"
	"golang-dh/pkg/database"
)

func setupRoutes(app *fiber.App, dbService database.PostgreSQLServicer) {
	auth := middleware.HeaderAuth()
	api := app.Group("/api")

	// 1. Repositories
	compRepo := companyRepo.NewCompanyRepository(dbService)
	schedRepo := scheduleRepo.NewScheduleRepository(dbService)
	toolRepo := toolsRepo.NewToolsRepository(dbService)
	asstRepo := assetRepo.NewAssetRepository(dbService)
	usrRepo := authRepo.NewUserRepository(dbService)
	chtRepo := chatRepo.NewChatRepository(dbService)

	// 2. Services
	compSvc := companySvc.NewCompanyService(compRepo)
	schedSvc := scheduleSvc.NewScheduleService(schedRepo)
	toolSvc := toolsSvc.NewToolsService(toolRepo)
	asstSvc := assetSvc.NewAssetService(asstRepo)
	atSvc := authSvc.NewAuthService(usrRepo)
	chtSvc := chatSvc.NewChatService(chtRepo)

	// 3. Module Routes
	company.SetupRoutes(api, compSvc)
	app.Get("/api/perusahaan", func(c *fiber.Ctx) error {
		comps, err := compSvc.GetAllCompanies(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"companies": comps})
	})

	authMod.SetupRoutes(api, auth, atSvc)
	app.Post("/api/auth/register", func(c *fiber.Ctx) error {
		var req authDto.RegisterRequest
		_ = c.BodyParser(&req)
		res, err := atSvc.Register(c.Context(), req)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(fiber.Map{"token": res.Token, "user": res.User})
	})
	app.Post("/api/auth/login", func(c *fiber.Ctx) error {
		var req authDto.LoginRequest
		_ = c.BodyParser(&req)
		res, err := atSvc.Login(c.Context(), req)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": err.Error()})
		}
		c.Set("X-User-ID", res.User.ID)
		c.Set("X-User-Role", res.User.Role)
		c.Set("X-User-Name", res.User.Username)
		c.Set("X-Company", res.User.Company)
		if res.User.IdPerusahaan > 0 {
			c.Set("X-Company-ID", strconv.Itoa(res.User.IdPerusahaan))
		}
		token := res.Token
		if token == "" {
			token = "sess_" + res.User.ID
		}
		return c.JSON(fiber.Map{"message": "login successful", "token": token, "user": res.User})
	})

	asset.SetupRoutes(api, auth, asstSvc)
	schedule.SetupRoutes(api, schedSvc)
	app.Post("/schedule", func(c *fiber.Ctx) error {
		var req scheduleDto.CreateScheduleRequest
		_ = c.BodyParser(&req)
		id, err := schedSvc.CreateSchedule(c.Context(), req)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(fiber.Map{"id_schedule": id})
	})
	app.Get("/schedule", func(c *fiber.Ctx) error {
		schedules, err := schedSvc.GetSchedules(c.Context(), c.Query("category"))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"schedules": schedules})
	})

	tools.SetupRoutes(api, auth, toolSvc)
	chat.SetupRoutes(app, auth, chtSvc)

	// 4. Specialized Media & Stream Handlers
	api.Post("/pdf", controllers.PdfHandler)
	api.Post("/csv", controllers.CsvHandler)
	api.Post("/excel", controllers.ExcelHandler)
	app.Post("/message", auth, controllers.SendMessage)
	app.Post("/message/audio", auth, controllers.SendAudioMessage)
	app.Post("/transcribe", controllers.TranscribeAudioHandler)
	app.Post("/tts", controllers.TTSHandler)
	app.Get("/tts", controllers.TTSHandler)
	api.Post("/tts", controllers.TTSHandler)
	api.Get("/tts", controllers.TTSHandler)
	app.Get("/ws/:id", middleware.WebSocketMiddleware, websocket.New(controllers.SendMessages))
}

func main() {
	// Initialize PostgreSQL pool
	pool, err := database.InitPool()
	if err != nil {
		log.Fatalf("[main] Database connection failed: %v", err)
	}
	defer pool.Close()

	dbService := database.NewPostgreSQLService(pool)
	// Retain global pool for legacy handlers during gradual migration
	controllers.DB = pool
	_ = controllers.EnsureScheduleTable(context.Background())

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())

	setupRoutes(app, dbService)
	app.Static("/uploads", "./uploads")
	log.Fatal(app.Listen("0.0.0.0:3000"))
}

