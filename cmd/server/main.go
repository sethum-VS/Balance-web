package main

import (
	"fmt"
	"log"
	"os"

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

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files from the web/static directory
	e.Static("/static", "web/static")

	// Register HTTP routes (health check, index page)
	httpH := httphandlers.NewHandlers()
	httpH.RegisterRoutes(e)

	// Register WebSocket routes (placeholder)
	wsH := wshandlers.NewHandlers()
	wsH.RegisterRoutes(e)

	// Start server
	address := fmt.Sprintf(":%s", port)
	log.Printf("Balance Web server starting on http://localhost%s\n", address)
	e.Logger.Fatal(e.Start(address))
}
