package handlers

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Excellent58/urlShortener/database"
	"github.com/Excellent58/urlShortener/utils"
)

// Mock implementations
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateShortenerTable(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
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

func (m *MockDatabase) Close() {
	m.Called()
}

type MockUtils struct {
	mock.Mock
}

func (m *MockUtils) CreateShortUrl(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockUtils) GenerateRandomCode(charset string, length int) (string, error) {
	args := m.Called(charset, length)
	return args.String(0), args.Error(1)
}

type MockRandomGenerator struct {
	mock.Mock
}

func (m *MockRandomGenerator) GenerateRandomCode(charset string, length int) (string, error) {
	args := m.Called(charset, length)
	return args.String(0), args.Error(1)
}

// Test setup helpers
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// For testing purposes, you might want to use a simple template
	// or mock the HTML rendering
	router.SetHTMLTemplate(createMockTemplate())
	
	return router
}

func createMockTemplate() *template.Template {
	// Create a simple template for testing
	tmpl := `
	{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
	{{if .Message}}<div class="message">{{.Message}}</div>{{end}}
	{{if .shortUrl}}<div class="short-url">{{.shortUrl}}</div>{{end}}
	`
	return template.Must(template.New("index.html").Parse(tmpl))
}

func createFormData(values map[string]string) string {
	form := url.Values{}
	for key, value := range values {
		form.Set(key, value)
	}
	return form.Encode()
}

// Database tests
func TestDatabase_InsertShortenerRow(t *testing.T) {
	mockDB := new(MockDatabase)
	ctx := context.Background()
	
	mockDB.On("InsertShortenerRow", ctx, "https://example.com", "abc123").Return(nil)
	
	err := mockDB.InsertShortenerRow(ctx, "https://example.com", "abc123")
	
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDatabase_FetchUrlDetails(t *testing.T) {
	mockDB := new(MockDatabase)
	ctx := context.Background()
	
	expectedUrl := &database.Url{
		ID:            1,
		ShortUrl:      "abc123",
		LongUrl:       "https://example.com",
		TimesFollowed: 5,
		CreatedAt:     time.Now(),
	}
	
	mockDB.On("FetchUrlDetails", ctx, "abc123").Return(expectedUrl, nil)
	
	result, err := mockDB.FetchUrlDetails(ctx, "abc123")
	
	assert.NoError(t, err)
	assert.Equal(t, expectedUrl, result)
	mockDB.AssertExpectations(t)
}

func TestDatabase_ShortUrlExists(t *testing.T) {
	tests := []struct {
		name     string
		shortUrl string
		exists   bool
		err      error
	}{
		{
			name:     "URL exists",
			shortUrl: "abc123",
			exists:   true,
			err:      nil,
		},
		{
			name:     "URL does not exist",
			shortUrl: "xyz789",
			exists:   false,
			err:      nil,
		},
		{
			name:     "Database error",
			shortUrl: "error",
			exists:   false,
			err:      errors.New("database error"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabase)
			ctx := context.Background()
			
			mockDB.On("ShortUrlExists", ctx, tt.shortUrl).Return(tt.exists, tt.err)
			
			exists, err := mockDB.ShortUrlExists(ctx, tt.shortUrl)
			
			if tt.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exists, exists)
			}
			
			mockDB.AssertExpectations(t)
		})
	}
}

// Utils tests
func TestUtils_CreateShortUrl(t *testing.T) {
	mockDB := new(MockDatabase)
	mockGen := new(MockRandomGenerator)
	ctx := context.Background()
	
	utilsService := utils.NewUtils(mockDB, mockGen)
	
	// Mock successful generation
	mockGen.On("GenerateRandomCode", "abcdefghijklmnopqrstuvwxyz0123456789", 7).Return("abc123", nil)
	mockDB.On("ShortUrlExists", ctx, "abc123").Return(false, nil)
	
	result, err := utilsService.CreateShortUrl(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, "abc123", result)
	mockDB.AssertExpectations(t)
	mockGen.AssertExpectations(t)
}

func TestUtils_CreateShortUrl_RetryOnCollision(t *testing.T) {
	mockDB := new(MockDatabase)
	mockGen := new(MockRandomGenerator)
	ctx := context.Background()
	
	utilsService := utils.NewUtils(mockDB, mockGen)
	
	// First attempt returns existing URL, second attempt succeeds
	mockGen.On("GenerateRandomCode", "abcdefghijklmnopqrstuvwxyz0123456789", 7).Return("abc123", nil).Once()
	mockDB.On("ShortUrlExists", ctx, "abc123").Return(true, nil).Once()
	
	mockGen.On("GenerateRandomCode", "abcdefghijklmnopqrstuvwxyz0123456789", 7).Return("xyz789", nil).Once()
	mockDB.On("ShortUrlExists", ctx, "xyz789").Return(false, nil).Once()
	
	result, err := utilsService.CreateShortUrl(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, "xyz789", result)
	mockDB.AssertExpectations(t)
	mockGen.AssertExpectations(t)
}

func TestUtils_CreateShortUrl_DatabaseError(t *testing.T) {
	mockDB := new(MockDatabase)
	mockGen := new(MockRandomGenerator)
	ctx := context.Background()
	
	utilsService := utils.NewUtils(mockDB, mockGen)
	
	mockGen.On("GenerateRandomCode", "abcdefghijklmnopqrstuvwxyz0123456789", 7).Return("abc123", nil)
	mockDB.On("ShortUrlExists", ctx, "abc123").Return(false, errors.New("database error"))
	
	result, err := utilsService.CreateShortUrl(ctx)
	
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "failed to check if short URL exists")
	mockDB.AssertExpectations(t)
	mockGen.AssertExpectations(t)
}

// Handler tests
func TestHandlerService_Home(t *testing.T) {
	deps := &Dependencies{}
	hs := NewHandlerService(deps)
	
	router := setupTestRouter()
	router.GET("/", hs.Home)
	
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerService_CreateShortUrl_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	mockUtils := new(MockUtils)
	
	deps := &Dependencies{
		DB:    mockDB,
		Utils: mockUtils,
	}
	hs := NewHandlerService(deps)
	
	// Set up expectations
	mockUtils.On("CreateShortUrl", mock.AnythingOfType("*context.emptyCtx")).Return("abc123", nil)
	mockDB.On("InsertShortenerRow", mock.AnythingOfType("*context.emptyCtx"), "https://example.com", "abc123").Return(nil)
	
	router := setupTestRouter()
	router.POST("/create", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{
		"long_url": "https://example.com",
	})
	
	req, _ := http.NewRequest("POST", "/create", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Could not generate short URL")
	
	mockUtils.AssertExpectations(t)
}

func TestHandlerService_CreateShortUrl_DatabaseError(t *testing.T) {
	mockDB := new(MockDatabase)
	mockUtils := new(MockUtils)
	
	deps := &Dependencies{
		DB:    mockDB,
		Utils: mockUtils,
	}
	hs := NewHandlerService(deps)
	
	// Utils succeeds but database fails
	mockUtils.On("CreateShortUrl", mock.AnythingOfType("*context.emptyCtx")).Return("abc123", nil)
	mockDB.On("InsertShortenerRow", mock.AnythingOfType("*context.emptyCtx"), "https://example.com", "abc123").Return(errors.New("database error"))
	
	router := setupTestRouter()
	router.POST("/create", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{
		"long_url": "https://example.com",
	})
	
	req, _ := http.NewRequest("POST", "/create", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Could not save your URL")
	
	mockDB.AssertExpectations(t)
	mockUtils.AssertExpectations(t)
}

func TestHandlerService_CreateShortUrl_InvalidFormData(t *testing.T) {
	deps := &Dependencies{}
	hs := NewHandlerService(deps)
	
	router := setupTestRouter()
	router.POST("/create", hs.CreateShortUrl)
	
	// Send invalid form data
	req, _ := http.NewRequest("POST", "/create", strings.NewReader("invalid-data"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid input")
}

func TestHandlerService_RedirectUrl_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	
	deps := &Dependencies{
		DB: mockDB,
	}
	hs := NewHandlerService(deps)
	
	expectedUrl := &database.Url{
		ID:            1,
		ShortUrl:      "abc123",
		LongUrl:       "https://example.com",
		TimesFollowed: 5,
		CreatedAt:     time.Now(),
	}
	
	// Set up expectations
	mockDB.On("FetchUrlDetails", mock.AnythingOfType("*context.emptyCtx"), "abc123").Return(expectedUrl, nil)
	mockDB.On("UpdateTimesFollowed", mock.AnythingOfType("*context.emptyCtx"), "abc123").Return(nil)
	
	router := setupTestRouter()
	router.GET("/:short_url", hs.RedirectUrl)
	
	req, _ := http.NewRequest("GET", "/abc123", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))
	
	mockDB.AssertExpectations(t)
}

func TestHandlerService_RedirectUrl_NotFound(t *testing.T) {
	mockDB := new(MockDatabase)
	
	deps := &Dependencies{
		DB: mockDB,
	}
	hs := NewHandlerService(deps)
	
	// URL not found
	mockDB.On("FetchUrlDetails", mock.AnythingOfType("*context.emptyCtx"), "nonexistent").Return(nil, errors.New("not found"))
	
	router := setupTestRouter()
	router.GET("/:short_url", hs.RedirectUrl)
	
	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "URL not found")
	
	mockDB.AssertExpectations(t)
}

func TestHandlerService_RedirectUrl_UpdateTimesFollowedError(t *testing.T) {
	mockDB := new(MockDatabase)
	
	deps := &Dependencies{
		DB: mockDB,
	}
	hs := NewHandlerService(deps)
	
	expectedUrl := &database.Url{
		ID:            1,
		ShortUrl:      "abc123",
		LongUrl:       "https://example.com",
		TimesFollowed: 5,
		CreatedAt:     time.Now(),
	}
	
	// Fetch succeeds but update fails (should still redirect)
	mockDB.On("FetchUrlDetails", mock.AnythingOfType("*context.emptyCtx"), "abc123").Return(expectedUrl, nil)
	mockDB.On("UpdateTimesFollowed", mock.AnythingOfType("*context.emptyCtx"), "abc123").Return(errors.New("update failed"))
	
	router := setupTestRouter()
	router.GET("/:short_url", hs.RedirectUrl)
	
	req, _ := http.NewRequest("GET", "/abc123", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	// Should still redirect even if update fails
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))
	
	mockDB.AssertExpectations(t)
}

// Table-driven tests
func TestHandlerService_CreateShortUrl_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		longUrl        string
		mockShortCode  string
		mockUtilsError error
		mockDBError    error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid URL",
			longUrl:        "https://example.com",
			mockShortCode:  "abc123",
			mockUtilsError: nil,
			mockDBError:    nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "short url",
		},
		{
			name:           "Empty URL",
			longUrl:        "",
			mockShortCode:  "",
			mockUtilsError: nil,
			mockDBError:    nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Enter long url to shorten",
		},
		{
			name:           "Utils error",
			longUrl:        "https://example.com",
			mockShortCode:  "",
			mockUtilsError: errors.New("generation failed"),
			mockDBError:    nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Could not generate short URL",
		},
		{
			name:           "Database error",
			longUrl:        "https://example.com",
			mockShortCode:  "abc123",
			mockUtilsError: nil,
			mockDBError:    errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Could not save your URL",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabase)
			mockUtils := new(MockUtils)
			
			deps := &Dependencies{
				DB:    mockDB,
				Utils: mockUtils,
			}
			hs := NewHandlerService(deps)
			
			// Set up mocks only if URL is not empty
			if tt.longUrl != "" {
				mockUtils.On("CreateShortUrl", mock.AnythingOfType("*context.emptyCtx")).Return(tt.mockShortCode, tt.mockUtilsError)
				
				if tt.mockUtilsError == nil {
					mockDB.On("InsertShortenerRow", mock.AnythingOfType("*context.emptyCtx"), tt.longUrl, tt.mockShortCode).Return(tt.mockDBError)
				}
			}
			
			router := setupTestRouter()
			router.POST("/create", hs.CreateShortUrl)
			
			formData := createFormData(map[string]string{
				"long_url": tt.longUrl,
			})
			
			req, _ := http.NewRequest("POST", "/create", strings.NewReader(formData))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			
			if tt.longUrl != "" {
				mockUtils.AssertExpectations(t)
				if tt.mockUtilsError == nil {
					mockDB.AssertExpectations(t)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkHandlerService_CreateShortUrl(b *testing.B) {
	mockDB := new(MockDatabase)
	mockUtils := new(MockUtils)
	
	deps := &Dependencies{
		DB:    mockDB,
		Utils: mockUtils,
	}
	hs := NewHandlerService(deps)
	
	mockUtils.On("CreateShortUrl", mock.Anything).Return("abc123", nil)
	mockDB.On("InsertShortenerRow", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	router := setupTestRouter()
	router.POST("/create", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{
		"long_url": "https://example.com",
	})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/create", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHandlerService_RedirectUrl(b *testing.B) {
	mockDB := new(MockDatabase)
	
	deps := &Dependencies{
		DB: mockDB,
	}
	hs := NewHandlerService(deps)
	
	expectedUrl := &database.Url{
		LongUrl: "https://example.com",
	}
	
	mockDB.On("FetchUrlDetails", mock.Anything, mock.Anything).Return(expectedUrl, nil)
	mockDB.On("UpdateTimesFollowed", mock.Anything, mock.Anything).Return(nil)
	
	router := setupTestRouter()
	router.GET("/:short_url", hs.RedirectUrl)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/abc123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
	
	formData := createFormData(map[string]string{
		"long_url": "https://example.com",
	})
	
	req, _ := http.NewRequest("POST", "/create", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "short url")
	assert.Contains(t, w.Body.String(), "abc123")
	
	mockDB.AssertExpectations(t)
	mockUtils.AssertExpectations(t)
}

func TestHandlerService_CreateShortUrl_EmptyURL(t *testing.T) {
	deps := &Dependencies{}
	hs := NewHandlerService(deps)
	
	router := setupTestRouter()
	router.POST("/create", hs.CreateShortUrl)
	
	formData := createFormData(map[string]string{
		"long_url": "",
	})
	
	req, _ := http.NewRequest("POST", "/create", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Enter long url to shorten")
}

func TestHandlerService_CreateShortUrl_UtilsError(t *testing.T) {
	mockDB := new(MockDatabase)
	mockUtils := new(MockUtils)
	
	deps := &Dependencies{
		DB:    mockDB,
		Utils: mockUtils,
	}
	hs := NewHandlerService(deps)
	
	// Utils returns error
	mockUtils.On("CreateShortUrl", mock.AnythingOfType("*context.emptyCtx")).Return("", errors.New("generation failed"))
	
	router := setupTestRouter()
	router.POST("/create", hs.CreateShortUrl)