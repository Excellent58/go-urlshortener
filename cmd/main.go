package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Excellent58/urlShortener/database"
	"github.com/Excellent58/urlShortener/handlers"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading [.env file]: ", err)
  }

  ctx := context.Background()
  database.InitDB(ctx)
  defer database.Pool.Close()

  pool := database.Pool
  // create table
  if err := database.CreateShortenerTable(ctx, pool); err != nil {
    fmt.Println("Tables failed to be created")
    log.Println(err)
  }

  router := gin.Default()
  router.LoadHTMLGlob("templates/*")
  router.Static("/static", "./static")

  router.GET("/", handlers.Home)
  router.POST("/", handlers.CreateShortUrl)
  router.GET("/:short_url", handlers.RedirectUrl)
  
  router.Run() 
}