package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	domainErrors "go-multi-chat-api/src/domain/errors"
	domainUser "go-multi-chat-api/src/domain/user"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/repository/mysql/user"
	"go-multi-chat-api/src/infrastructure/security"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type IAuthUseCase interface {
	Login(email, password string) (*domainUser.User, *AuthTokens, error)
	AccessTokenByRefreshToken(refreshToken string) (*domainUser.User, *AuthTokens, error)
	InitiateAzureADAuth() (string, string, error)
	CompleteAzureADAuth(code, state string) (*domainUser.User, *AuthTokens, error)
}

type AuthUseCase struct {
	UserRepository user.UserRepositoryInterface
	JWTService     security.IJWTService
	LDAPService    security.ILDAPService
	AzureADService security.IAzureADService
	Logger         *logger.Logger
}

func NewAuthUseCase(
	userRepository user.UserRepositoryInterface,
	jwtService security.IJWTService,
	ldapService security.ILDAPService,
	azureADService security.IAzureADService,
	loggerInstance *logger.Logger,
) IAuthUseCase {
	return &AuthUseCase{
		UserRepository: userRepository,
		JWTService:     jwtService,
		LDAPService:    ldapService,
		AzureADService: azureADService,
		Logger:         loggerInstance,
	}
}

type AuthTokens struct {
	AccessToken               string
	RefreshToken              string
	ExpirationAccessDateTime  time.Time
	ExpirationRefreshDateTime time.Time
}

func (s *AuthUseCase) Login(email, password string) (*domainUser.User, *AuthTokens, error) {
	s.Logger.Info("User login attempt", zap.String("email", email))

	var user *domainUser.User
	var err error

	// Try LDAP authentication first if enabled
	if s.LDAPService != nil && s.LDAPService.IsEnabled() {
		s.Logger.Info("Attempting LDAP authentication", zap.String("email", email))
		// For LDAP, we use the email as the username
		username := email
		// If email contains @, extract the username part
		if idx := strings.Index(email, "@"); idx > 0 {
			username = email[:idx]
		}

		ldapUser, ldapErr := s.LDAPService.Authenticate(username, password)
		if ldapErr == nil && ldapUser != nil {
			s.Logger.Info("LDAP authentication successful", zap.String("email", email))

			// Check if user exists in local database
			dbUser, dbErr := s.UserRepository.GetByEmail(ldapUser.Email)
			if dbErr != nil || dbUser.ID == 0 {
				// User doesn't exist in local database, create a new user
				s.Logger.Info("Creating new user from LDAP", zap.String("email", ldapUser.Email))

				// Set a random password hash for the local user (they'll continue using LDAP auth)
				randomHash, _ := bcrypt.GenerateFromPassword([]byte(time.Now().String()), bcrypt.DefaultCost)
				ldapUser.HashPassword = string(randomHash)

				// Create user in local database
				dbUser, dbErr = s.UserRepository.Create(ldapUser)
				if dbErr != nil {
					s.Logger.Error("Error creating user from LDAP", zap.Error(dbErr))
					return nil, nil, dbErr
				}
				user = dbUser
			} else {
				// User exists in local database, use that user
				user = dbUser
			}
		} else {
			// LDAP authentication failed, log and fall back to local authentication
			s.Logger.Info("LDAP authentication failed, falling back to local authentication", zap.Error(ldapErr))
		}
	}

	// If LDAP authentication failed or not enabled, try local authentication
	if user == nil {
		dbUser, dbErr := s.UserRepository.GetByEmail(email)
		if dbErr != nil {
			s.Logger.Error("Error getting user for login", zap.Error(dbErr), zap.String("email", email))
			return nil, nil, dbErr
		}
		if dbUser.ID == 0 {
			s.Logger.Warn("Login failed: user not found", zap.String("email", email))
			return nil, nil, domainErrors.NewAppError(errors.New("email or password does not match"), domainErrors.NotAuthenticated)
		}

		isAuthenticated := checkPasswordHash(password, dbUser.HashPassword)
		if !isAuthenticated {
			s.Logger.Warn("Login failed: invalid password", zap.String("email", email))
			return nil, nil, domainErrors.NewAppError(errors.New("email or password does not match"), domainErrors.NotAuthenticated)
		}

		user = dbUser
	}

	// Generate tokens for authenticated user
	accessTokenClaims, err := s.JWTService.GenerateJWTToken(user.ID, "access", user.Role)
	if err != nil {
		s.Logger.Error("Error generating access token", zap.Error(err), zap.Int("userID", user.ID))
		return nil, nil, err
	}
	refreshTokenClaims, err := s.JWTService.GenerateJWTToken(user.ID, "refresh", user.Role)
	if err != nil {
		s.Logger.Error("Error generating refresh token", zap.Error(err), zap.Int("userID", user.ID))
		return nil, nil, err
	}

	authTokens := &AuthTokens{
		AccessToken:               accessTokenClaims.Token,
		RefreshToken:              refreshTokenClaims.Token,
		ExpirationAccessDateTime:  accessTokenClaims.ExpirationTime,
		ExpirationRefreshDateTime: refreshTokenClaims.ExpirationTime,
	}

	s.Logger.Info("User login successful", zap.String("email", email), zap.Int("userID", user.ID))
	return user, authTokens, nil
}

func (s *AuthUseCase) AccessTokenByRefreshToken(refreshToken string) (*domainUser.User, *AuthTokens, error) {
	s.Logger.Info("Refreshing access token")
	claimsMap, err := s.JWTService.GetClaimsAndVerifyToken(refreshToken, "refresh")
	if err != nil {
		s.Logger.Error("Error verifying refresh token", zap.Error(err))
		return nil, nil, err
	}
	userID := int(claimsMap["id"].(float64))
	user, err := s.UserRepository.GetByID(userID)
	if err != nil {
		s.Logger.Error("Error getting user for token refresh", zap.Error(err), zap.Int("userID", userID))
		return nil, nil, err
	}

	accessTokenClaims, err := s.JWTService.GenerateJWTToken(user.ID, "access", user.Role)
	if err != nil {
		s.Logger.Error("Error generating new access token", zap.Error(err), zap.Int("userID", user.ID))
		return nil, nil, err
	}

	var expTime = int64(claimsMap["exp"].(float64))

	authTokens := &AuthTokens{
		AccessToken:               accessTokenClaims.Token,
		ExpirationAccessDateTime:  accessTokenClaims.ExpirationTime,
		RefreshToken:              refreshToken,
		ExpirationRefreshDateTime: time.Unix(expTime, 0),
	}

	s.Logger.Info("Access token refreshed successfully", zap.Int("userID", user.ID))
	return user, authTokens, nil
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// InitiateAzureADAuth starts the Azure AD authentication process
func (s *AuthUseCase) InitiateAzureADAuth() (string, string, error) {
	if !s.AzureADService.IsEnabled() {
		return "", "", domainErrors.NewAppError(errors.New("Azure AD authentication is not enabled"), domainErrors.NotAuthenticated)
	}

	// Generate a random state parameter to prevent CSRF
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		s.Logger.Error("Error generating random state", zap.Error(err))
		return "", "", domainErrors.NewAppError(err, domainErrors.UnknownError)
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Get the authorization URL
	authURL := s.AzureADService.GetAuthorizationURL(state)

	s.Logger.Info("Initiated Azure AD authentication", zap.String("state", state))
	return authURL, state, nil
}

// CompleteAzureADAuth completes the Azure AD authentication process
func (s *AuthUseCase) CompleteAzureADAuth(code, state string) (*domainUser.User, *AuthTokens, error) {
	if !s.AzureADService.IsEnabled() {
		return nil, nil, domainErrors.NewAppError(errors.New("Azure AD authentication is not enabled"), domainErrors.NotAuthenticated)
	}

	s.Logger.Info("Completing Azure AD authentication", zap.String("state", state))

	// Exchange the authorization code for tokens
	tokenResponse, err := s.AzureADService.GetTokenFromCode(code)
	if err != nil {
		s.Logger.Error("Error getting token from code", zap.Error(err))
		return nil, nil, err
	}

	// Get user information from the access token
	azureUser, err := s.AzureADService.GetUserInfo(tokenResponse.AccessToken)
	if err != nil {
		s.Logger.Error("Error getting user info", zap.Error(err))
		return nil, nil, err
	}

	// Check if user exists in local database
	dbUser, dbErr := s.UserRepository.GetByEmail(azureUser.Email)
	if dbErr != nil || dbUser.ID == 0 {
		// User doesn't exist in local database, create a new user
		s.Logger.Info("Creating new user from Azure AD", zap.String("email", azureUser.Email))

		// Set a random password hash for the local user (they'll continue using Azure AD auth)
		randomHash, _ := bcrypt.GenerateFromPassword([]byte(time.Now().String()), bcrypt.DefaultCost)
		azureUser.HashPassword = string(randomHash)

		// Create user in local database
		dbUser, dbErr = s.UserRepository.Create(azureUser)
		if dbErr != nil {
			s.Logger.Error("Error creating user from Azure AD", zap.Error(dbErr))
			return nil, nil, dbErr
		}
	}

	// Generate tokens for authenticated user
	accessTokenClaims, err := s.JWTService.GenerateJWTToken(dbUser.ID, "access", dbUser.Role)
	if err != nil {
		s.Logger.Error("Error generating access token", zap.Error(err), zap.Int("userID", dbUser.ID))
		return nil, nil, err
	}
	refreshTokenClaims, err := s.JWTService.GenerateJWTToken(dbUser.ID, "refresh", dbUser.Role)
	if err != nil {
		s.Logger.Error("Error generating refresh token", zap.Error(err), zap.Int("userID", dbUser.ID))
		return nil, nil, err
	}

	authTokens := &AuthTokens{
		AccessToken:               accessTokenClaims.Token,
		RefreshToken:              refreshTokenClaims.Token,
		ExpirationAccessDateTime:  accessTokenClaims.ExpirationTime,
		ExpirationRefreshDateTime: refreshTokenClaims.ExpirationTime,
	}

	s.Logger.Info("Azure AD authentication successful", zap.String("email", dbUser.Email), zap.Int("userID", dbUser.ID))
	return dbUser, authTokens, nil
}
