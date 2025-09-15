package handlers

import (
	"context"
	"fmt"
	"net/http"
	"log"
	"github.com/Excellent58/urlShortener/database"
	"github.com/Excellent58/urlShortener/utils"
	"github.com/gin-gonic/gin"
)

type UrlForm struct {
	LongUrl string `form:"long_url"`
}


func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)	
}

func CreateShortUrl(c *gin.Context) {
	var form UrlForm
	if err := c.ShouldBind(&form); err != nil {
		c.HTML(http.StatusBadRequest, "index.html", gin.H{
			"Error": "Invalid input. Please enter your url.",
		})
		return
	}

	if form.LongUrl == "" {
		c.HTML(http.StatusBadRequest, "index.html", gin.H{
			"Error": "Enter long url to shorten",
		})
		return
	}

	shortCode := utils.CreateShortUrl()
	ctx := context.Background()
	err := database.InsertShortenerRow(ctx, database.Pool, form.LongUrl, shortCode)
	if err != nil {
		log.Printf("DB insert failed: %v", err)
		c.HTML(http.StatusInternalServerError, "index.html", gin.H{
			"Error": "Could not save your URL. Please try again.",
		})
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	host := c.Request.Host

	// shortUrl := fmt.Sprintf("http://localhost:8081/%s", shortCode)
	shortUrl := fmt.Sprintf("%s://%s/%s", scheme, host, shortCode)

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Message": "short url",
		"shortUrl": shortUrl,
	})
}

func RedirectUrl(c *gin.Context) {
	shortUrl := c.Param("short_url")
	ctx := context.Background()
	pool := database.Pool
	
	urlDetails, err := database.FetchUrlDetails(ctx, pool, shortUrl)
	if err != nil {
		fmt.Println("Error: ", err)
		c.String(http.StatusNotFound, "URl not found")
		return
	}

	if err := database.UpdateTimesFollowed(ctx, pool, shortUrl); err != nil {
		log.Printf("UpdateTimesFollowed failed for %s: %v", shortUrl, err)
	}

	log.Printf("Redirecting %s -> %s", shortUrl, urlDetails.LongUrl)
	c.Redirect(http.StatusFound, urlDetails.LongUrl)
}