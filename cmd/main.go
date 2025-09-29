package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Excellent58/urlShortener/database"
	"github.com/Excellent58/urlShortener/handlers"
  "github.com/Excellent58/urlShortener/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading [.env file]: ", err)
  }

  // Initialize database
  ctx := context.Background()
  db, err := database.InitDB(ctx)
  if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
  defer db.Close()

  // create table
  if err := db.CreateShortenerTable(ctx); err != nil {
    log.Fatalf("Failed to create table: %v", err)
  }

  // Create utils generator with database dependency
	urlGenerator := utils.NewGenerator(db)

  // 5. Create handler dependencies
	deps := &handlers.Dependencies{
		DB:        db,
		Generator: urlGenerator, // Inject generator into handlers
	}

  router := gin.Default()
  router.LoadHTMLGlob("templates/*")
  router.Static("/static", "./static")

  handlerService := handlers.NewHandlerService(deps)
	router.GET("/", handlerService.Home)
	router.POST("/", handlerService.CreateShortUrl)
	router.GET("/:short_url", handlerService.RedirectUrl)
  
  router.Run() 
}