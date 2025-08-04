package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go-multi-chat-api/src/domain"
	domainErrors "go-multi-chat-api/src/domain/errors"
	domainUser "go-multi-chat-api/src/domain/user"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/security"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type mockUserService struct {
	getByEmailFn         func(string) (*domainUser.User, error)
	getByIDFn            func(int) (*domainUser.User, error)
	createFn             func(*domainUser.User) (*domainUser.User, error)
	callGetByEmailCalled bool
	callGetByIDCalled    bool
}

func (m *mockUserService) GetAll() (*[]domainUser.User, error) {
	return nil, nil
}
func (m *mockUserService) GetByID(id int) (*domainUser.User, error) {
	m.callGetByIDCalled = true
	return m.getByIDFn(id)
}
func (m *mockUserService) GetByEmail(email string) (*domainUser.User, error) {
	m.callGetByEmailCalled = true
	return m.getByEmailFn(email)
}
func (m *mockUserService) Create(newUser *domainUser.User) (*domainUser.User, error) {
	if m.createFn != nil {
		return m.createFn(newUser)
	}
	return nil, nil
}
func (m *mockUserService) Delete(id int) error {
	return nil
}
func (m *mockUserService) Update(id int, userMap map[string]interface{}) (*domainUser.User, error) {
	return nil, nil
}
func (m *mockUserService) SearchPaginated(filters domain.DataFilters) (*domainUser.SearchResultUser, error) {
	return nil, nil
}
func (m *mockUserService) SearchByProperty(property string, searchText string) (*[]string, error) {
	return nil, nil
}

type mockJWTService struct {
	generateTokenFn func(int, string) (*security.AppToken, error)
	verifyTokenFn   func(string, string) (jwt.MapClaims, error)
}

func (m *mockJWTService) GenerateJWTToken(userID int, tokenType string) (*security.AppToken, error) {
	return m.generateTokenFn(userID, tokenType)
}

func (m *mockJWTService) GetClaimsAndVerifyToken(tokenString string, tokenType string) (jwt.MapClaims, error) {
	return m.verifyTokenFn(tokenString, tokenType)
}

type mockLDAPService struct {
	authenticateFn func(string, string) (*domainUser.User, error)
	isEnabledFn    func() bool
}

func (m *mockLDAPService) Authenticate(username, password string) (*domainUser.User, error) {
	if m.authenticateFn != nil {
		return m.authenticateFn(username, password)
	}
	return nil, errors.New("LDAP authentication not implemented")
}

func (m *mockLDAPService) IsEnabled() bool {
	if m.isEnabledFn != nil {
		return m.isEnabledFn()
	}
	return false
}

type mockAzureADService struct {
	isEnabledFn           func() bool
	getAuthorizationURLFn func(string) string
	getTokenFromCodeFn    func(string) (*security.AzureADTokenResponse, error)
	getUserInfoFn         func(string) (*domainUser.User, error)
}

func (m *mockAzureADService) IsEnabled() bool {
	if m.isEnabledFn != nil {
		return m.isEnabledFn()
	}
	return false
}

func (m *mockAzureADService) GetAuthorizationURL(state string) string {
	if m.getAuthorizationURLFn != nil {
		return m.getAuthorizationURLFn(state)
	}
	return ""
}

func (m *mockAzureADService) GetTokenFromCode(code string) (*security.AzureADTokenResponse, error) {
	if m.getTokenFromCodeFn != nil {
		return m.getTokenFromCodeFn(code)
	}
	return nil, errors.New("GetTokenFromCode not implemented")
}

func (m *mockAzureADService) GetUserInfo(accessToken string) (*domainUser.User, error) {
	if m.getUserInfoFn != nil {
		return m.getUserInfoFn(accessToken)
	}
	return nil, errors.New("GetUserInfo not implemented")
}

func setupLogger(t *testing.T) *logger.Logger {
	loggerInstance, err := logger.NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return loggerInstance
}

func HashPasswordForTest(plain string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func TestCheckPasswordHash(t *testing.T) {
	password := "mySecretPass"
	hashed, err := HashPasswordForTest(password)
	if err != nil {
		t.Fatalf("failed to generate hash for test: %v", err)
	}

	if ok := checkPasswordHash(password, hashed); !ok {
		t.Errorf("checkPasswordHash() = false, want true")
	}

	if ok := checkPasswordHash("wrongPassword", hashed); ok {
		t.Errorf("checkPasswordHash() = true, want false")
	}
}

func TestAuthUseCase_Login(t *testing.T) {
	tests := []struct {
		name                   string
		mockGetByEmailFn       func(string) (*domainUser.User, error)
		mockGenerateTokenFn    func(int, string) (*security.AppToken, error)
		inputEmail             string
		inputPassword          string
		wantErr                bool
		wantErrType            domainErrors.ErrorType
		wantEmptySecurity      bool
		wantSuccessAccessToken bool
	}{
		{
			name: "Error fetching user from DB",
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				return nil, errors.New("db error")
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "test_token"}, nil
			},
			inputEmail:    "test@example.com",
			inputPassword: "123456",
			wantErr:       true,
		},
		{
			name: "User not found (ID=0)",
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				return &domainUser.User{ID: 0}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "test_token"}, nil
			},
			inputEmail:    "test@example.com",
			inputPassword: "123456",
			wantErr:       true,
			wantErrType:   domainErrors.NotAuthenticated,
		},
		{
			name: "Incorrect password",
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				hashed, _ := HashPasswordForTest("someOtherPass")
				return &domainUser.User{ID: 10, HashPassword: hashed}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "test_token"}, nil
			},
			inputEmail:        "test@example.com",
			inputPassword:     "wrong",
			wantErr:           true,
			wantErrType:       domainErrors.NotAuthenticated,
			wantEmptySecurity: true,
		},
		{
			name: "Access token generation fails",
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				hashed, _ := HashPasswordForTest("somePass")
				return &domainUser.User{ID: 10, HashPassword: hashed}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return nil, errors.New("token generation failed")
			},
			inputEmail:    "test@example.com",
			inputPassword: "somePass",
			wantErr:       true,
		},
		{
			name: "OK - everything correct",
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				hashed, _ := HashPasswordForTest("mySecretPass")
				return &domainUser.User{
					ID:           10,
					Email:        "test@example.com",
					HashPassword: hashed,
				}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{
					Token:          "test_token_" + tokenType,
					ExpirationTime: time.Now().Add(time.Hour),
				}, nil
			},
			inputEmail:             "test@example.com",
			inputPassword:          "mySecretPass",
			wantErr:                false,
			wantSuccessAccessToken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepoMock := &mockUserService{
				getByEmailFn: tt.mockGetByEmailFn,
			}

			jwtMock := &mockJWTService{
				generateTokenFn: tt.mockGenerateTokenFn,
			}

			logger := setupLogger(t)
			uc := NewAuthUseCase(userRepoMock, jwtMock, nil, nil, logger)

			user, authTokens, err := uc.Login(tt.inputEmail, tt.inputPassword)
			if (err != nil) != tt.wantErr {
				t.Fatalf("[%s] got err = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}

			if tt.wantErrType != "" && err != nil {
				appErr, ok := err.(*domainErrors.AppError)
				if !ok || appErr.Type != tt.wantErrType {
					t.Errorf("[%s] expected error type = %s, got = %v", tt.name, tt.wantErrType, err)
				}
			}

			if !tt.wantErr && tt.wantSuccessAccessToken {
				if authTokens.AccessToken == "" {
					t.Errorf("[%s] expected a non-empty AccessToken, got empty", tt.name)
				}
				if user == nil {
					t.Errorf("[%s] expected a non-nil user, got nil", tt.name)
				}
			} else if tt.wantErr && tt.wantEmptySecurity {
				if authTokens != nil && authTokens.AccessToken != "" {
					t.Errorf("[%s] expected empty AccessToken, but got a non-empty one", tt.name)
				}
			}
		})
	}
}

func TestAuthUseCase_InitiateAzureADAuth(t *testing.T) {
	tests := []struct {
		name                    string
		mockAzureADIsEnabledFn  func() bool
		mockAzureADGetAuthURLFn func(string) string
		wantErr                 bool
		wantErrType             domainErrors.ErrorType
		wantAuthURL             string
	}{
		{
			name: "Azure AD not enabled",
			mockAzureADIsEnabledFn: func() bool {
				return false
			},
			wantErr:     true,
			wantErrType: domainErrors.NotAuthenticated,
		},
		{
			name: "Azure AD enabled - success",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockAzureADGetAuthURLFn: func(state string) string {
				return "https://login.microsoftonline.com/auth?state=" + state
			},
			wantErr:     false,
			wantAuthURL: "https://login.microsoftonline.com/auth?state=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock services
			userRepoMock := &mockUserService{}
			jwtMock := &mockJWTService{}
			ldapMock := &mockLDAPService{}
			azureADMock := &mockAzureADService{
				isEnabledFn:           tt.mockAzureADIsEnabledFn,
				getAuthorizationURLFn: tt.mockAzureADGetAuthURLFn,
			}

			logger := setupLogger(t)
			uc := NewAuthUseCase(userRepoMock, jwtMock, ldapMock, azureADMock, logger)

			authURL, state, err := uc.InitiateAzureADAuth()
			if (err != nil) != tt.wantErr {
				t.Fatalf("[%s] got err = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}

			if tt.wantErrType != "" && err != nil {
				appErr, ok := err.(*domainErrors.AppError)
				if !ok || appErr.Type != tt.wantErrType {
					t.Errorf("[%s] expected error type = %s, got = %v", tt.name, tt.wantErrType, err)
				}
			}

			if !tt.wantErr {
				if authURL == "" {
					t.Errorf("[%s] expected a non-empty authURL, got empty", tt.name)
				}
				if state == "" {
					t.Errorf("[%s] expected a non-empty state, got empty", tt.name)
				}
				if tt.wantAuthURL != "" && !strings.HasPrefix(authURL, tt.wantAuthURL) {
					t.Errorf("[%s] expected authURL to start with %s, got %s", tt.name, tt.wantAuthURL, authURL)
				}
			}
		})
	}
}

func TestAuthUseCase_CompleteAzureADAuth(t *testing.T) {
	tests := []struct {
		name                   string
		mockAzureADIsEnabledFn func() bool
		mockGetTokenFromCodeFn func(string) (*security.AzureADTokenResponse, error)
		mockGetUserInfoFn      func(string) (*domainUser.User, error)
		mockGetByEmailFn       func(string) (*domainUser.User, error)
		mockCreateUserFn       func(*domainUser.User) (*domainUser.User, error)
		mockGenerateTokenFn    func(int, string) (*security.AppToken, error)
		inputCode              string
		inputState             string
		wantErr                bool
		wantErrType            domainErrors.ErrorType
		wantSuccess            bool
	}{
		{
			name: "Azure AD not enabled",
			mockAzureADIsEnabledFn: func() bool {
				return false
			},
			inputCode:   "test-code",
			inputState:  "test-state",
			wantErr:     true,
			wantErrType: domainErrors.NotAuthenticated,
		},
		{
			name: "GetTokenFromCode fails",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockGetTokenFromCodeFn: func(code string) (*security.AzureADTokenResponse, error) {
				return nil, errors.New("token exchange failed")
			},
			inputCode:  "test-code",
			inputState: "test-state",
			wantErr:    true,
		},
		{
			name: "GetUserInfo fails",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockGetTokenFromCodeFn: func(code string) (*security.AzureADTokenResponse, error) {
				return &security.AzureADTokenResponse{
					AccessToken: "test-access-token",
				}, nil
			},
			mockGetUserInfoFn: func(accessToken string) (*domainUser.User, error) {
				return nil, errors.New("failed to get user info")
			},
			inputCode:  "test-code",
			inputState: "test-state",
			wantErr:    true,
		},
		{
			name: "User not found - create user fails",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockGetTokenFromCodeFn: func(code string) (*security.AzureADTokenResponse, error) {
				return &security.AzureADTokenResponse{
					AccessToken: "test-access-token",
				}, nil
			},
			mockGetUserInfoFn: func(accessToken string) (*domainUser.User, error) {
				return &domainUser.User{
					Email:     "azure@example.com",
					FirstName: "Azure",
					LastName:  "User",
				}, nil
			},
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				return &domainUser.User{ID: 0}, nil // ID=0 means user not found
			},
			mockCreateUserFn: func(user *domainUser.User) (*domainUser.User, error) {
				return nil, errors.New("failed to create user")
			},
			inputCode:  "test-code",
			inputState: "test-state",
			wantErr:    true,
		},
		{
			name: "Token generation fails",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockGetTokenFromCodeFn: func(code string) (*security.AzureADTokenResponse, error) {
				return &security.AzureADTokenResponse{
					AccessToken: "test-access-token",
				}, nil
			},
			mockGetUserInfoFn: func(accessToken string) (*domainUser.User, error) {
				return &domainUser.User{
					Email:     "azure@example.com",
					FirstName: "Azure",
					LastName:  "User",
				}, nil
			},
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				return &domainUser.User{ID: 1, Email: "azure@example.com"}, nil // Existing user
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return nil, errors.New("token generation failed")
			},
			inputCode:  "test-code",
			inputState: "test-state",
			wantErr:    true,
		},
		{
			name: "Success - existing user",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockGetTokenFromCodeFn: func(code string) (*security.AzureADTokenResponse, error) {
				return &security.AzureADTokenResponse{
					AccessToken: "test-access-token",
				}, nil
			},
			mockGetUserInfoFn: func(accessToken string) (*domainUser.User, error) {
				return &domainUser.User{
					Email:     "azure@example.com",
					FirstName: "Azure",
					LastName:  "User",
				}, nil
			},
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				return &domainUser.User{ID: 1, Email: "azure@example.com"}, nil // Existing user
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{
					Token:          "test-token-" + tokenType,
					ExpirationTime: time.Now().Add(time.Hour),
				}, nil
			},
			inputCode:   "test-code",
			inputState:  "test-state",
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name: "Success - new user",
			mockAzureADIsEnabledFn: func() bool {
				return true
			},
			mockGetTokenFromCodeFn: func(code string) (*security.AzureADTokenResponse, error) {
				return &security.AzureADTokenResponse{
					AccessToken: "test-access-token",
				}, nil
			},
			mockGetUserInfoFn: func(accessToken string) (*domainUser.User, error) {
				return &domainUser.User{
					Email:     "azure@example.com",
					FirstName: "Azure",
					LastName:  "User",
				}, nil
			},
			mockGetByEmailFn: func(email string) (*domainUser.User, error) {
				return &domainUser.User{ID: 0}, nil // ID=0 means user not found
			},
			mockCreateUserFn: func(user *domainUser.User) (*domainUser.User, error) {
				return &domainUser.User{ID: 2, Email: "azure@example.com"}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{
					Token:          "test-token-" + tokenType,
					ExpirationTime: time.Now().Add(time.Hour),
				}, nil
			},
			inputCode:   "test-code",
			inputState:  "test-state",
			wantErr:     false,
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock services
			userRepoMock := &mockUserService{
				getByEmailFn: tt.mockGetByEmailFn,
			}
			if tt.mockCreateUserFn != nil {
				userRepoMock.createFn = tt.mockCreateUserFn
			}

			jwtMock := &mockJWTService{
				generateTokenFn: tt.mockGenerateTokenFn,
			}

			ldapMock := &mockLDAPService{}

			azureADMock := &mockAzureADService{
				isEnabledFn:        tt.mockAzureADIsEnabledFn,
				getTokenFromCodeFn: tt.mockGetTokenFromCodeFn,
				getUserInfoFn:      tt.mockGetUserInfoFn,
			}

			logger := setupLogger(t)
			uc := NewAuthUseCase(userRepoMock, jwtMock, ldapMock, azureADMock, logger)

			user, authTokens, err := uc.CompleteAzureADAuth(tt.inputCode, tt.inputState)
			if (err != nil) != tt.wantErr {
				t.Fatalf("[%s] got err = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}

			if tt.wantErrType != "" && err != nil {
				appErr, ok := err.(*domainErrors.AppError)
				if !ok || appErr.Type != tt.wantErrType {
					t.Errorf("[%s] expected error type = %s, got = %v", tt.name, tt.wantErrType, err)
				}
			}

			if !tt.wantErr && tt.wantSuccess {
				if authTokens.AccessToken == "" {
					t.Errorf("[%s] expected a non-empty AccessToken, got empty", tt.name)
				}
				if authTokens.RefreshToken == "" {
					t.Errorf("[%s] expected a non-empty RefreshToken, got empty", tt.name)
				}
				if user == nil {
					t.Errorf("[%s] expected a non-nil user, got nil", tt.name)
				}
			}
		})
	}
}

func TestAuthUseCase_AccessTokenByRefreshToken(t *testing.T) {
	tests := []struct {
		name                string
		mockVerifyTokenFn   func(string, string) (jwt.MapClaims, error)
		mockGetByIDFn       func(int) (*domainUser.User, error)
		mockGenerateTokenFn func(int, string) (*security.AppToken, error)
		inputRefreshToken   string
		wantErr             bool
		wantErrType         domainErrors.ErrorType
		wantSuccess         bool
	}{
		{
			name: "Invalid refresh token",
			mockVerifyTokenFn: func(token, tokenType string) (jwt.MapClaims, error) {
				return nil, errors.New("invalid token")
			},
			mockGetByIDFn: func(id int) (*domainUser.User, error) {
				return &domainUser.User{ID: 10}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "new_access_token"}, nil
			},
			inputRefreshToken: "invalid_token",
			wantErr:           true,
		},
		{
			name: "User not found after token verification",
			mockVerifyTokenFn: func(token, tokenType string) (jwt.MapClaims, error) {
				return jwt.MapClaims{"id": float64(999)}, nil
			},
			mockGetByIDFn: func(id int) (*domainUser.User, error) {
				return nil, errors.New("user not found")
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "new_access_token"}, nil
			},
			inputRefreshToken: "valid_token",
			wantErr:           true,
		},
		{
			name: "New access token generation fails",
			mockVerifyTokenFn: func(token, tokenType string) (jwt.MapClaims, error) {
				return jwt.MapClaims{"id": float64(10)}, nil
			},
			mockGetByIDFn: func(id int) (*domainUser.User, error) {
				return &domainUser.User{ID: 10}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return nil, errors.New("token generation failed")
			},
			inputRefreshToken: "valid_token",
			wantErr:           true,
		},
		{
			name: "OK - successful token refresh",
			mockVerifyTokenFn: func(token, tokenType string) (jwt.MapClaims, error) {
				return jwt.MapClaims{"id": float64(10), "exp": float64(time.Now().Add(time.Hour).Unix())}, nil
			},
			mockGetByIDFn: func(id int) (*domainUser.User, error) {
				return &domainUser.User{ID: 10, Email: "test@example.com"}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "new.token", TokenType: tokenType, ExpirationTime: time.Now().Add(time.Hour)}, nil
			},
			inputRefreshToken: "valid_refresh_token",
			wantErr:           false,
			wantSuccess:       true,
		},
		{
			name: "Refresh token generation fails",
			mockVerifyTokenFn: func(token string, tokenType string) (jwt.MapClaims, error) {
				return jwt.MapClaims{"id": float64(10), "type": "refresh"}, nil
			},
			mockGetByIDFn: func(id int) (*domainUser.User, error) {
				return &domainUser.User{ID: 10}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return nil, errors.New("token generation failed")
			},
			inputRefreshToken: "valid.refresh.token",
			wantErr:           true,
		},
		{
			name: "OK - everything correct",
			mockVerifyTokenFn: func(token string, tokenType string) (jwt.MapClaims, error) {
				return jwt.MapClaims{"id": float64(10), "type": "refresh", "exp": float64(time.Now().Add(time.Hour).Unix())}, nil
			},
			mockGetByIDFn: func(id int) (*domainUser.User, error) {
				return &domainUser.User{ID: 10}, nil
			},
			mockGenerateTokenFn: func(userID int, tokenType string) (*security.AppToken, error) {
				return &security.AppToken{Token: "new.token", TokenType: tokenType, ExpirationTime: time.Now().Add(time.Hour)}, nil
			},
			inputRefreshToken: "valid.refresh.token",
			wantErr:           false,
			wantSuccess:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepoMock := &mockUserService{
				getByIDFn: tt.mockGetByIDFn,
			}

			jwtMock := &mockJWTService{
				verifyTokenFn:   tt.mockVerifyTokenFn,
				generateTokenFn: tt.mockGenerateTokenFn,
			}

			logger := setupLogger(t)
			uc := NewAuthUseCase(userRepoMock, jwtMock, nil, nil, logger)

			user, authTokens, err := uc.AccessTokenByRefreshToken(tt.inputRefreshToken)
			if (err != nil) != tt.wantErr {
				t.Fatalf("[%s] got err = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}

			if tt.wantErrType != "" && err != nil {
				appErr, ok := err.(*domainErrors.AppError)
				if !ok || appErr.Type != tt.wantErrType {
					t.Errorf("[%s] expected error type = %s, got = %v", tt.name, tt.wantErrType, err)
				}
			}

			if !tt.wantErr && tt.wantSuccess {
				if authTokens.AccessToken == "" {
					t.Errorf("[%s] expected a non-empty AccessToken, got empty", tt.name)
				}
				if user == nil {
					t.Errorf("[%s] expected a non-nil user, got nil", tt.name)
				}
			}
		})
	}
}
