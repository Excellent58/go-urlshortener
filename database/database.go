package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Url struct {
	ID	int
	ShortUrl string
	LongUrl string
	TimesFollowed int
	CreatedAt time.Time
}

var Pool *pgxpool.Pool

func InitDB(ctx context.Context) {
	dsn := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	
	Pool = pool
	log.Println("[DATABASE] CONNECTED!!")
}

func CreateShortenerTable(ctx context.Context, poll *pgxpool.Pool) error {
	_, err := poll.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS shortener (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			long_url TEXT,
			short_url TEXT UNIQUE,
			times_followed BIGINT NOT NULL DEFAULT 0 CHECK (times_followed >= 0) 
		);
		`)

	return err
}

func InsertShortenerRow(ctx context.Context, pool *pgxpool.Pool, longUrl, shortUrl string) error {
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO shortener (long_url, short_url) VALUES($1, $2)
	`, longUrl, shortUrl)

	if err != nil {
		fmt.Println("error: ", err)
		return fmt.Errorf("failed to insert row into shortener: %w", err)
	}

	return nil
}

func FetchUrlDetails(ctx context.Context, pool *pgxpool.Pool, ShortUrl string) (*Url, error) {
	var u Url
	err := pool.QueryRow(ctx, `
		SELECT id, short_url, long_url, times_followed, created_at
		FROM shortener
		WHERE short_url = $1
	`, ShortUrl).Scan(
		&u.ID,
		&u.ShortUrl,
		&u.LongUrl,
		&u.TimesFollowed,
		&u.CreatedAt,
	)

	if err != nil {
		return nil, err
	} 

	return &u, nil
}

func UpdateTimesFollowed(ctx context.Context, pool *pgxpool.Pool, short_url string) error {
	_, err := pool.Exec(ctx, `
		UPDATE shortener
		SET times_followed = times_followed + 1
		WHERE short_url = $1
	`, short_url)

	return err
}

func ShortUrlExists(ctx context.Context, shortUrl string, pool *pgxpool.Pool) (bool, error) {
	if pool == nil {
		fmt.Println("Pool empty")
		return false, fmt.Errorf("database pool is nil")
	}

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM shortener WHERE short_url = $1
		)
	`, shortUrl).Scan(&exists)

	if err != nil {
		fmt.Println("query failed: ", err)
		return false, fmt.Errorf("query failed: %w", err)
	}
	return exists, nil
}