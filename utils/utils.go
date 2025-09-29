package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/Excellent58/urlShortener/database"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
var length = 7

type UrlGenerator interface {
	CreateShortUrl(ctx context.Context) (string, error)
}

// Generator implements UrlGenerator with database dependency
type Generator struct {
	db database.DatabaseInterface
}
// NewGenerator creates a new Generator with database dependency
func NewGenerator(db database.DatabaseInterface) *Generator {
	return &Generator{
		db: db,
	}
}

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

func (g *Generator) CreateShortUrl(ctx context.Context) (string, error) {
	shortUrl, err := generateRandomCode(charset, length)
	if err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}

	shortUrlExists, _ := g.db.ShortUrlExists(ctx, shortUrl)

	if shortUrlExists {
		generateRandomCode(charset, length)
	}
	
	return shortUrl, nil
}