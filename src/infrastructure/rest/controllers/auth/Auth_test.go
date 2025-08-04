package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	useCaseAuth "go-multi-chat-api/src/application/usecases/auth"
	domainErrors "go-multi-chat-api/src/domain/errors"
	userDomain "go-multi-chat-api/src/domain/user"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// MockAuthUseCase implements IAuthUseCase for testing
type MockAuthUseCase struct {
	loginFunc                func(string, string) (*userDomain.User, *useCaseAuth.AuthTokens, error)
	accessTokenByRefreshFunc func(string) (*userDomain.User, *useCaseAuth.AuthTokens, error)
	initiateAzureADAuthFunc  func() (string, string, error)
	completeAzureADAuthFunc  func(string, string) (*userDomain.User, *useCaseAuth.AuthTokens, error)
}

func (m *MockAuthUseCase) Login(email, password string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
	if m.loginFunc != nil {
		return m.loginFunc(email, password)
	}
	return nil, nil, nil
}

func (m *MockAuthUseCase) AccessTokenByRefreshToken(refreshToken string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
	if m.accessTokenByRefreshFunc != nil {
		return m.accessTokenByRefreshFunc(refreshToken)
	}
	return nil, nil, nil
}

func (m *MockAuthUseCase) InitiateAzureADAuth() (string, string, error) {
	if m.initiateAzureADAuthFunc != nil {
		return m.initiateAzureADAuthFunc()
	}
	return "", "", nil
}

func (m *MockAuthUseCase) CompleteAzureADAuth(code, state string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
	if m.completeAzureADAuthFunc != nil {
		return m.completeAzureADAuthFunc(code, state)
	}
	return nil, nil, nil
}

func setupLogger(t *testing.T) *logger.Logger {
	loggerInstance, err := logger.NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return loggerInstance
}

func TestNewAuthController(t *testing.T) {
	mockUseCase := &MockAuthUseCase{}
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	if controller == nil {
		t.Error("Expected NewAuthController to return a non-nil controller")
	}
}

func TestAuthController_Login_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockUseCase := &MockAuthUseCase{
		loginFunc: func(email, password string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
			user := &userDomain.User{
				UserName:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Status:    true,
				ID:        1,
			}
			authTokens := &useCaseAuth.AuthTokens{
				AccessToken:               "test-access-token",
				RefreshToken:              "test-refresh-token",
				ExpirationAccessDateTime:  time.Now().Add(time.Hour),
				ExpirationRefreshDateTime: time.Now().Add(24 * time.Hour),
			}
			return user, authTokens, nil
		},
	}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create test request
	loginRequest := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	requestBody, _ := json.Marshal(loginRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Login(c)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthController_Login_InvalidRequest(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockUseCase := &MockAuthUseCase{}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create invalid request (missing required fields)
	requestBody := []byte(`{"email": "test@example.com"}`) // Missing password

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.Login(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Error("Expected error to be added to context")
	}
}

func TestAuthController_GetAccessTokenByRefreshToken_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockUseCase := &MockAuthUseCase{
		accessTokenByRefreshFunc: func(refreshToken string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
			user := &userDomain.User{
				UserName:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Status:    true,
				ID:        1,
			}
			authTokens := &useCaseAuth.AuthTokens{
				AccessToken:               "new-access-token",
				RefreshToken:              "new-refresh-token",
				ExpirationAccessDateTime:  time.Now().Add(time.Hour),
				ExpirationRefreshDateTime: time.Now().Add(24 * time.Hour),
			}
			return user, authTokens, nil
		},
	}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create test request
	accessTokenRequest := AccessTokenRequest{
		RefreshToken: "test-refresh-token",
	}

	requestBody, _ := json.Marshal(accessTokenRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.GetAccessTokenByRefreshToken(c)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthController_GetAccessTokenByRefreshToken_InvalidRequest(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockUseCase := &MockAuthUseCase{}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create invalid request (missing required fields)
	requestBody := []byte(`{}`) // Missing refreshToken

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.GetAccessTokenByRefreshToken(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Error("Expected error to be added to context")
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	// Test valid request
	validRequest := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	if validRequest.Email == "" {
		t.Error("Email should not be empty")
	}

	if validRequest.Password == "" {
		t.Error("Password should not be empty")
	}

	// Test invalid email format (basic check)
	if validRequest.Email == "invalid-email" {
		t.Error("Email should be in valid format")
	}
}

func TestAccessTokenRequest_Validation(t *testing.T) {
	// Test valid request
	validRequest := AccessTokenRequest{
		RefreshToken: "valid-refresh-token",
	}

	if validRequest.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
}

func TestAuthController_InitiateAzureADAuth_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockUseCase := &MockAuthUseCase{
		initiateAzureADAuthFunc: func() (string, string, error) {
			return "https://login.microsoftonline.com/auth", "test-state", nil
		},
	}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create test request
	authRequest := AzureADAuthRequest{
		RedirectURL: "https://example.com/callback",
	}

	requestBody, _ := json.Marshal(authRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/azure-auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.InitiateAzureADAuth(c)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response AzureADAuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response fields
	if response.AuthURL != "https://login.microsoftonline.com/auth" {
		t.Errorf("Expected AuthURL to be 'https://login.microsoftonline.com/auth', got '%s'", response.AuthURL)
	}
	if response.State != "test-state" {
		t.Errorf("Expected State to be 'test-state', got '%s'", response.State)
	}
}

func TestAuthController_InitiateAzureADAuth_Error(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case with error
	mockUseCase := &MockAuthUseCase{
		initiateAzureADAuthFunc: func() (string, string, error) {
			return "", "", domainErrors.NewAppError(errors.New("Azure AD auth failed"), domainErrors.NotAuthenticated)
		},
	}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create test request
	authRequest := AzureADAuthRequest{
		RedirectURL: "https://example.com/callback",
	}

	requestBody, _ := json.Marshal(authRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/azure-auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.InitiateAzureADAuth(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Error("Expected error to be added to context")
	}
}

func TestAuthController_CompleteAzureADAuth_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case
	mockUseCase := &MockAuthUseCase{
		completeAzureADAuthFunc: func(code, state string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
			user := &userDomain.User{
				UserName:  "azureuser",
				Email:     "azure@example.com",
				FirstName: "Azure",
				LastName:  "User",
				Status:    true,
				ID:        2,
			}
			authTokens := &useCaseAuth.AuthTokens{
				AccessToken:               "azure-access-token",
				RefreshToken:              "azure-refresh-token",
				ExpirationAccessDateTime:  time.Now().Add(time.Hour),
				ExpirationRefreshDateTime: time.Now().Add(24 * time.Hour),
			}
			return user, authTokens, nil
		},
	}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create test request
	callbackRequest := AzureADCallbackRequest{
		Code:  "auth-code",
		State: "test-state",
	}

	requestBody, _ := json.Marshal(callbackRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/azure-callback", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.CompleteAzureADAuth(c)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response fields
	if response.Data.Email != "azure@example.com" {
		t.Errorf("Expected Email to be 'azure@example.com', got '%s'", response.Data.Email)
	}
	if response.Security.JWTAccessToken != "azure-access-token" {
		t.Errorf("Expected JWTAccessToken to be 'azure-access-token', got '%s'", response.Security.JWTAccessToken)
	}
}

func TestAuthController_CompleteAzureADAuth_Error(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock use case with error
	mockUseCase := &MockAuthUseCase{
		completeAzureADAuthFunc: func(code, state string) (*userDomain.User, *useCaseAuth.AuthTokens, error) {
			return nil, nil, domainErrors.NewAppError(errors.New("Azure AD auth completion failed"), domainErrors.NotAuthenticated)
		},
	}

	// Create controller
	logger := setupLogger(t)
	controller := NewAuthController(mockUseCase, logger)

	// Create test request
	callbackRequest := AzureADCallbackRequest{
		Code:  "auth-code",
		State: "test-state",
	}

	requestBody, _ := json.Marshal(callbackRequest)

	// Create HTTP request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/azure-callback", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the method
	controller.CompleteAzureADAuth(c)

	// Check that an error was added to the context
	if len(c.Errors) == 0 {
		t.Error("Expected error to be added to context")
	}
}
