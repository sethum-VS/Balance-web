package main

import (
	"fmt"
	"log"
	"os"

	"balance-web/internal/application"
	"balance-web/internal/infrastructure/auth"
	"balance-web/internal/infrastructure/turso"
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

	// 1. Initialize Firebase Auth
	firebaseAuth := auth.NewFirebaseAuth()

	// 2. Initialize Turso Store (auto-migrates schema)
	store := turso.NewStore()

	// 3. Create turso repository adapters that satisfy the domain interfaces
	activityRepo := turso.NewActivityRepoAdapter(store)
	sessionRepo := turso.NewSessionRepoAdapter(store)

	// 4. Initialize WebSocket Hub & Run it concurrently
	hub := websocket.NewHub()
	go hub.Run()

	// 5. Initialize TimerService with the adapter-wrapped repositories
	timerService := application.NewTimerService(sessionRepo, activityRepo)

	// Inject the user-scoped timer service logic into the Hub for welcome messages
	hub.GetGlobalBalance = func(userID string) int {
		return timerService.CalculateGlobalBalance(userID)
	}

	// 6. Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files from the web/static directory
	e.Static("/static", "web/static")

	// 7. Instantiate & Register HTTP routes (includes auth middleware for /api/*)
	httpH := httphandlers.NewHandlers(store, activityRepo, sessionRepo, timerService, hub, firebaseAuth)
	httpH.RegisterRoutes(e)

	// 8. Instantiate WebSocket handlers and register with auth middleware
	wsH := wshandlers.NewHandlers(hub)
	authMW := httphandlers.FirebaseAuthMiddleware(firebaseAuth)
	e.GET("/ws", wsH.ServeWS, authMW)

	// Start server
	address := fmt.Sprintf(":%s", port)
	log.Printf("Balance Web server starting on http://localhost%s\n", address)
	e.Logger.Fatal(e.Start(address))
}
