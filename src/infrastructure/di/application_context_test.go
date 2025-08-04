package di

import (
	"os"
	"testing"

	"go-multi-chat-api/src/domain"
	domainUser "go-multi-chat-api/src/domain/user"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/security"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories and services
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetAll() (*[]domainUser.User, error) {
	args := m.Called()
	return args.Get(0).(*[]domainUser.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(id int) (*domainUser.User, error) {
	args := m.Called(id)
	return args.Get(0).(*domainUser.User), args.Error(1)
}

func (m *MockUserRepository) Create(user *domainUser.User) (*domainUser.User, error) {
	args := m.Called(user)
	return args.Get(0).(*domainUser.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*domainUser.User, error) {
	args := m.Called(email)
	return args.Get(0).(*domainUser.User), args.Error(1)
}

func (m *MockUserRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Update(id int, userMap map[string]interface{}) (*domainUser.User, error) {
	args := m.Called(id, userMap)
	return args.Get(0).(*domainUser.User), args.Error(1)
}

func (m *MockUserRepository) SearchPaginated(filters domain.DataFilters) (*domainUser.SearchResultUser, error) {
	args := m.Called(filters)
	return args.Get(0).(*domainUser.SearchResultUser), args.Error(1)
}

func (m *MockUserRepository) SearchByProperty(property string, searchText string) (*[]string, error) {
	args := m.Called(property, searchText)
	return args.Get(0).(*[]string), args.Error(1)
}

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateJWTToken(userID int, tokenType string) (*security.AppToken, error) {
	args := m.Called(userID, tokenType)
	return args.Get(0).(*security.AppToken), args.Error(1)
}

func (m *MockJWTService) GetClaimsAndVerifyToken(tokenString string, tokenType string) (jwt.MapClaims, error) {
	args := m.Called(tokenString, tokenType)
	return args.Get(0).(jwt.MapClaims), args.Error(1)
}

type MockLDAPService struct {
	mock.Mock
}

func (m *MockLDAPService) Authenticate(username, password string) (*domainUser.User, error) {
	args := m.Called(username, password)
	return args.Get(0).(*domainUser.User), args.Error(1)
}

func (m *MockLDAPService) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

type MockAzureADService struct {
	mock.Mock
}

func (m *MockAzureADService) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAzureADService) GetAuthorizationURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *MockAzureADService) GetTokenFromCode(code string) (*security.AzureADTokenResponse, error) {
	args := m.Called(code)
	return args.Get(0).(*security.AzureADTokenResponse), args.Error(1)
}

func (m *MockAzureADService) GetUserInfo(accessToken string) (*domainUser.User, error) {
	args := m.Called(accessToken)
	return args.Get(0).(*domainUser.User), args.Error(1)
}

func setupLogger(t *testing.T) *logger.Logger {
	loggerInstance, err := logger.NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return loggerInstance
}

func TestNewTestApplicationContext(t *testing.T) {
	mockUserRepo := &MockUserRepository{}
	mockJWTService := &MockJWTService{}
	mockLDAPService := &MockLDAPService{}
	mockAzureADService := &MockAzureADService{}
	logger := setupLogger(t)

	appContext := NewTestApplicationContext(mockUserRepo, mockJWTService, mockLDAPService, mockAzureADService, logger)

	assert.NotNil(t, appContext)
	assert.Equal(t, mockUserRepo, appContext.UserRepository)
	assert.Equal(t, mockJWTService, appContext.JWTService)
	assert.Equal(t, mockLDAPService, appContext.LDAPService)
	assert.Equal(t, mockAzureADService, appContext.AzureADService)

	// Test that controllers are created
	assert.NotNil(t, appContext.AuthController)
	assert.NotNil(t, appContext.UserController)

	// Test that use cases are created
	assert.NotNil(t, appContext.AuthUseCase)
	assert.NotNil(t, appContext.UserUseCase)
}

func TestSetupDependencies(t *testing.T) {
	// This test will fail in CI/CD without a real database connection
	// We'll test the error path by setting invalid environment variables
	originalPort := os.Getenv("DB_PORT")
	os.Setenv("DB_PORT", "99999") // Invalid port to cause connection failure
	defer os.Setenv("DB_PORT", originalPort)

	logger := setupLogger(t)
	appContext, err := SetupDependencies(logger)

	assert.Error(t, err)
	assert.Nil(t, appContext)
}

func TestApplicationContextStructure(t *testing.T) {
	mockUserRepo := &MockUserRepository{}
	mockJWTService := &MockJWTService{}
	mockLDAPService := &MockLDAPService{}
	mockAzureADService := &MockAzureADService{}
	logger := setupLogger(t)

	appContext := NewTestApplicationContext(mockUserRepo, mockJWTService, mockLDAPService, mockAzureADService, logger)

	// Test that all fields are properly set
	assert.NotNil(t, appContext.AuthController)
	assert.NotNil(t, appContext.UserController)
	assert.NotNil(t, appContext.JWTService)
	assert.NotNil(t, appContext.LDAPService)
	assert.NotNil(t, appContext.AzureADService)
	assert.NotNil(t, appContext.UserRepository)
	assert.NotNil(t, appContext.AuthUseCase)
	assert.NotNil(t, appContext.UserUseCase)

	// Test that DB is nil in test context (as expected)
	assert.Nil(t, appContext.DB)
}
