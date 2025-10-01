package utils

import (
	"context"
	"testing"

	"github.com/Excellent58/urlShortener/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabase mocks the database interface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) ShortUrlExists(ctx context.Context, shortUrl string) (bool, error) {
	args := m.Called(ctx, shortUrl)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabase) InsertShortenerRow(ctx context.Context, longUrl, shortUrl string) error {
	args := m.Called(ctx, longUrl, shortUrl)
	return args.Error(0)
}

func (m *MockDatabase) FetchUrlDetails(ctx context.Context, shortUrl string) (*database.Url, error) {
	args := m.Called(ctx, shortUrl)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.Url), args.Error(1)
}

func (m *MockDatabase) UpdateTimesFollowed(ctx context.Context, shortUrl string) error {
	args := m.Called(ctx, shortUrl)
	return args.Error(0)
}

func (m *MockDatabase) CreateShortenerTable(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDatabase) Close() {
	m.Called()
}

// Test: Successfully create a short URL
func TestGenerator_CreateShortUrl_Success(t *testing.T) {
	// Arrange
	mockDB := new(MockDatabase)
	generator := NewGenerator(mockDB)
	ctx := context.Background()

	// Mock: URL doesn't exist (first attempt succeeds)
	mockDB.On("ShortUrlExists", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

	// Act
	shortUrl, err := generator.CreateShortUrl(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, shortUrl)
	assert.Equal(t, 7, len(shortUrl)) // Default length is 7
	mockDB.AssertExpectations(t)
}
