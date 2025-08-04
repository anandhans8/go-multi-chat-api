package auth

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AccessTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// AzureADAuthRequest is used to initiate Azure AD authentication
type AzureADAuthRequest struct {
	RedirectURL string `json:"redirectUrl" binding:"required"`
}

// AzureADAuthResponse contains the URL to redirect the user to for Azure AD authentication
type AzureADAuthResponse struct {
	AuthURL string `json:"authUrl"`
	State   string `json:"state"`
}

// AzureADCallbackRequest is used to handle the callback from Azure AD
type AzureADCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

type UserData struct {
	UserName  string `json:"userName"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Status    bool   `json:"status"`
	ID        int    `json:"id"`
}

type SecurityData struct {
	JWTAccessToken            string    `json:"jwtAccessToken"`
	JWTRefreshToken           string    `json:"jwtRefreshToken"`
	ExpirationAccessDateTime  time.Time `json:"expirationAccessDateTime"`
	ExpirationRefreshDateTime time.Time `json:"expirationRefreshDateTime"`
}

type LoginResponse struct {
	Data     UserData     `json:"data"`
	Security SecurityData `json:"security"`
}
