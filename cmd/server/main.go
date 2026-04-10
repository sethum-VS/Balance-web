package main

import (
	"fmt"
	"log"
	"os"

	"balance-web/internal/application"
	"balance-web/internal/infrastructure/memory"
	"balance-web/internal/infrastructure/websocket"
	httphandlers "balance-web/internal/presentation/http"
	wshandlers "balance-web/internal/presentation/ws"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// 1. Initialize In-Memory Store (auto-seeds mock data)
	store := memory.NewStore()

	// 2. Create repository adapters that satisfy the domain interfaces
	activityRepo := memory.NewActivityRepoAdapter(store)
	sessionRepo := memory.NewSessionRepoAdapter(store)

	// 3. Initialize WebSocket Hub & Run it concurrently
	hub := websocket.NewHub()
	go hub.Run()

	// 4. Initialize TimerService with the adapter-wrapped repositories
	timerService := application.NewTimerService(sessionRepo, activityRepo)

	// 5. Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files from the web/static directory
	e.Static("/static", "web/static")

	// 6. Instantiate & Register HTTP routes
	httpH := httphandlers.NewHandlers(store, timerService, hub)
	httpH.RegisterRoutes(e)

	// 7. Instantiate & Register WebSocket routes
	wsH := wshandlers.NewHandlers(hub)
	wsH.RegisterRoutes(e)

	// Start server
	address := fmt.Sprintf(":%s", port)
	log.Printf("Balance Web server starting on http://localhost%s\n", address)
	e.Logger.Fatal(e.Start(address))
}
