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

// Dependencies holds all injected dependencies
type Dependencies struct {
	DB        database.DatabaseInterface
	Generator utils.UrlGenerator // Utils generator with database already injected
}

// HandlerService provides handler methods
type HandlerService struct {
	deps *Dependencies
}

// NewHandlerService creates a new handler service
func NewHandlerService(deps *Dependencies) *HandlerService {
	return &HandlerService{deps: deps}
}

func (hs *HandlerService) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}


func (hs *HandlerService) CreateShortUrl(c *gin.Context) {
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

	ctx := context.Background()
	shortCode, err := hs.deps.Generator.CreateShortUrl(ctx)
	if err != nil {
		log.Printf("Failed to create short code: %v", err)
		c.HTML(http.StatusInternalServerError, "index.html", gin.H{
			"Error": "Could not generate short URL. Please try again.",
		})
		return
	}

	err = hs.deps.DB.InsertShortenerRow(ctx, form.LongUrl, shortCode)

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

func (hs *HandlerService) RedirectUrl(c *gin.Context) {
	shortUrl := c.Param("short_url")
	ctx := context.Background()
	
	urlDetails, err := hs.deps.DB.FetchUrlDetails(ctx, shortUrl)
	if err != nil {
		log.Printf("URL not found: %s, error: %v", shortUrl, err)
		c.String(http.StatusNotFound, "URl not found")
		return
	}

	if err := hs.deps.DB.UpdateTimesFollowed(ctx, shortUrl); err != nil {
		log.Printf("UpdateTimesFollowed failed for %s: %v", shortUrl, err)
	}

	log.Printf("Redirecting %s -> %s", shortUrl, urlDetails.LongUrl)
	c.Redirect(http.StatusFound, urlDetails.LongUrl)
}