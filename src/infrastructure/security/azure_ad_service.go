package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	domainErrors "go-multi-chat-api/src/domain/errors"
	domainUser "go-multi-chat-api/src/domain/user"
	logger "go-multi-chat-api/src/infrastructure/logger"

	"go.uber.org/zap"
)

// AzureADConfig holds the configuration for Azure AD connection
type AzureADConfig struct {
	TenantID          string
	ClientID          string
	ClientSecret      string
	RedirectURI       string
	Scopes            []string
	GraphAPIEndpoint  string
	AuthorizeEndpoint string
	TokenEndpoint     string
	Enabled           bool
}

// IAzureADService defines the interface for Azure AD operations
type IAzureADService interface {
	IsEnabled() bool
	GetAuthorizationURL(state string) string
	GetTokenFromCode(code string) (*AzureADTokenResponse, error)
	GetUserInfo(accessToken string) (*domainUser.User, error)
}

// AzureADService implements the IAzureADService interface
type AzureADService struct {
	Config AzureADConfig
	Logger *logger.Logger
	Client *http.Client
}

// AzureADTokenResponse represents the response from Azure AD token endpoint
type AzureADTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

// AzureADUserInfo represents the user information from Azure AD Graph API
type AzureADUserInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	UserPrincipalName string `json:"userPrincipalName"`
	Mail              string `json:"mail"`
}

// NewAzureADService creates a new Azure AD service
func NewAzureADService(config AzureADConfig, loggerInstance *logger.Logger) IAzureADService {
	// Set default endpoints if not provided
	if config.AuthorizeEndpoint == "" {
		config.AuthorizeEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", config.TenantID)
	}
	if config.TokenEndpoint == "" {
		config.TokenEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", config.TenantID)
	}
	if config.GraphAPIEndpoint == "" {
		config.GraphAPIEndpoint = "https://graph.microsoft.com/v1.0/me"
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email", "User.Read"}
	}

	return &AzureADService{
		Config: config,
		Logger: loggerInstance,
		Client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// IsEnabled returns whether Azure AD authentication is enabled
func (s *AzureADService) IsEnabled() bool {
	return s.Config.Enabled
}

// GetAuthorizationURL generates the authorization URL for Azure AD OAuth flow
func (s *AzureADService) GetAuthorizationURL(state string) string {
	params := url.Values{}
	params.Add("client_id", s.Config.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", s.Config.RedirectURI)
	params.Add("response_mode", "query")
	params.Add("scope", strings.Join(s.Config.Scopes, " "))
	params.Add("state", state)

	return fmt.Sprintf("%s?%s", s.Config.AuthorizeEndpoint, params.Encode())
}

// GetTokenFromCode exchanges an authorization code for an access token
func (s *AzureADService) GetTokenFromCode(code string) (*AzureADTokenResponse, error) {
	if !s.Config.Enabled {
		return nil, domainErrors.NewAppError(errors.New("Azure AD authentication is not enabled"), domainErrors.NotAuthenticated)
	}

	s.Logger.Info("Exchanging authorization code for token")

	data := url.Values{}
	data.Set("client_id", s.Config.ClientID)
	data.Set("client_secret", s.Config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.Config.RedirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", s.Config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		s.Logger.Error("Error creating token request", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		s.Logger.Error("Error sending token request", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Error("Error reading token response", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}

	if resp.StatusCode != http.StatusOK {
		s.Logger.Error("Token request failed", zap.Int("status", resp.StatusCode), zap.String("response", string(body)))
		return nil, domainErrors.NewAppError(fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body)), domainErrors.NotAuthenticated)
	}

	var tokenResponse AzureADTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		s.Logger.Error("Error parsing token response", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}

	return &tokenResponse, nil
}

// GetUserInfo retrieves user information from Azure AD Graph API
func (s *AzureADService) GetUserInfo(accessToken string) (*domainUser.User, error) {
	if !s.Config.Enabled {
		return nil, domainErrors.NewAppError(errors.New("Azure AD authentication is not enabled"), domainErrors.NotAuthenticated)
	}

	s.Logger.Info("Getting user info from Azure AD")

	req, err := http.NewRequest("GET", s.Config.GraphAPIEndpoint, nil)
	if err != nil {
		s.Logger.Error("Error creating user info request", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := s.Client.Do(req)
	if err != nil {
		s.Logger.Error("Error sending user info request", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Error("Error reading user info response", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}

	if resp.StatusCode != http.StatusOK {
		s.Logger.Error("User info request failed", zap.Int("status", resp.StatusCode), zap.String("response", string(body)))
		return nil, domainErrors.NewAppError(fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body)), domainErrors.NotAuthenticated)
	}

	var userInfo AzureADUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		s.Logger.Error("Error parsing user info response", zap.Error(err))
		return nil, domainErrors.NewAppError(err, domainErrors.NotAuthenticated)
	}

	// Create a domain user from the Azure AD user info
	email := userInfo.Mail
	if email == "" {
		email = userInfo.UserPrincipalName
	}

	user := &domainUser.User{
		UserName:  userInfo.UserPrincipalName,
		Email:     email,
		FirstName: userInfo.GivenName,
		LastName:  userInfo.Surname,
		Status:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.Logger.Info("Azure AD authentication successful", zap.String("email", user.Email))
	return user, nil
}
