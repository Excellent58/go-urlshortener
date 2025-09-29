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

//DatabaseInterface defines the contract for database operations
type DatabaseInterface interface {
	CreateShortenerTable(ctx context.Context) error
	InsertShortenerRow(ctx context.Context, longUrl, shortUrl string) error
	FetchUrlDetails(ctx context.Context, shortUrl string) (*Url, error)
	UpdateTimesFollowed(ctx context.Context, shortUrl string) error
	ShortUrlExists(ctx context.Context, shortUrl string) (bool, error)
	Close()
}

//Database implements DatabaseInterface
type Database struct {
	pool *pgxpool.Pool
}

// Global variable for backward compatibility
var Pool *pgxpool.Pool

// NewDatabase creates a new Database instance
func NewDatabase(pool *pgxpool.Pool) *Database {
	return &Database{
		pool: pool,
	}
}

// InitDB initializes the database connection and returns a Database instance
func InitDB(ctx context.Context) (*Database, error) {
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
	return  NewDatabase(pool), nil
}

// Dtabase method implementations
func (db *Database) CreateShortenerTable(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `
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

func (db *Database) InsertShortenerRow(ctx context.Context, longUrl, shortUrl string) error {
	if db.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	_, err := db.pool.Exec(ctx, `
		INSERT INTO shortener (long_url, short_url) VALUES($1, $2)
	`, longUrl, shortUrl)

	if err != nil {
		return fmt.Errorf("failed to insert row into shortener: %w", err)
	}

	return nil
}

func (db *Database) FetchUrlDetails(ctx context.Context, ShortUrl string) (*Url, error) {
	var u Url
	err := db.pool.QueryRow(ctx, `
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

func (db *Database) UpdateTimesFollowed(ctx context.Context, short_url string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE shortener
		SET times_followed = times_followed + 1
		WHERE short_url = $1
	`, short_url)

	return err
}

func (db *Database) ShortUrlExists(ctx context.Context, shortUrl string) (bool, error) {
	if db.pool == nil {
		return false, fmt.Errorf("database pool is nil")
	}

	var exists bool
	err := db.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM shortener WHERE short_url = $1
		)
	`, shortUrl).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}
	return exists, nil
}