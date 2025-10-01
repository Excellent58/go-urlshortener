package handlers

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Excellent58/urlShortener/database"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockDatabase struct {
	mock.Mock
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

func (m *MockDatabase) ShortUrlExists(ctx context.Context, shortUrl string) (bool, error) {
	args := m.Called(ctx, shortUrl)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabase) CreateShortenerTable(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDatabase) Close() {
	m.Called()
}

type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) CreateShortUrl(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

// Test setup
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	tmpl := `{{if .Error}}{{.Error}}{{end}}{{if .shortUrl}}{{.shortUrl}}{{end}}`
	router.SetHTMLTemplate(template.Must(template.New("index.html").Parse(tmpl)))
	
	return router
}

func createFormData(values map[string]string) string {
	form := url.Values{}
	for key, value := range values {
		form.Set(key, value)
	}
	return form.Encode()
}

// Test: Create short URL successfully
func TestHandlerService_CreateShortUrl_Success(t *testing.T) {
	// Arrange
	mockDB := new(MockDatabase)
	mockGen := new(MockGenerator)
	
	deps := &Dependencies{
		DB:        mockDB,
		Generator: mockGen,
	}
	hs := NewHandlerService(deps)
	
	mockGen.On("CreateShortUrl", mock.Anything).Return("abc123", nil)
	mockDB.On("InsertShortenerRow", mock.Anything, "https://example.com", "abc123").Return(nil)
	
	router := setupTestRouter()
	router.POST("/", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{"long_url": "https://example.com"})
	req, _ := http.NewRequest("POST", "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	// Act
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "abc123")
	mockDB.AssertExpectations(t)
	mockGen.AssertExpectations(t)
}

// Test: Empty URL
func TestHandlerService_CreateShortUrl_EmptyURL(t *testing.T) {
	// Arrange
	deps := &Dependencies{}
	hs := NewHandlerService(deps)
	
	router := setupTestRouter()
	router.POST("/", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{"long_url": ""})
	req, _ := http.NewRequest("POST", "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	// Act
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Enter long url to shorten")
}

// Test: Generator error
func TestHandlerService_CreateShortUrl_GeneratorError(t *testing.T) {
	// Arrange
	mockDB := new(MockDatabase)
	mockGen := new(MockGenerator)
	
	deps := &Dependencies{
		DB:        mockDB,
		Generator: mockGen,
	}
	hs := NewHandlerService(deps)
	
	mockGen.On("CreateShortUrl", mock.Anything).Return("", errors.New("generation failed"))
	
	router := setupTestRouter()
	router.POST("/", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{"long_url": "https://example.com"})
	req, _ := http.NewRequest("POST", "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	// Act
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Could not generate short URL")
	mockGen.AssertExpectations(t)
}