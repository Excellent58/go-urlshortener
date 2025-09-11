package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"github.com/Excellent58/urlShortener/database"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
var length = 7

func generateRandomCode(charset string, length int) (string, error) {
	randomCode := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		randomCode[i] = charset[num.Int64()]
	}

	return string(randomCode), nil
}

func CreateShortUrl() string {
	shortUrl, err := generateRandomCode(charset, length)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	ctx := context.Background()
	pool := database.Pool
	shortUrlExists, _ := database.ShortUrlExists(ctx, shortUrl, pool)

	if shortUrlExists {
		generateRandomCode(charset, length)
	}
	
	return shortUrl
}