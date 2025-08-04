package auth

import (
	"net/http"

	useCaseAuth "go-multi-chat-api/src/application/usecases/auth"
	domainErrors "go-multi-chat-api/src/domain/errors"
	logger "go-multi-chat-api/src/infrastructure/logger"
	"go-multi-chat-api/src/infrastructure/rest/controllers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IAuthController interface {
	Login(ctx *gin.Context)
	GetAccessTokenByRefreshToken(ctx *gin.Context)
	InitiateAzureADAuth(ctx *gin.Context)
	CompleteAzureADAuth(ctx *gin.Context)
}

type AuthController struct {
	authUseCase useCaseAuth.IAuthUseCase
	Logger      *logger.Logger
}

func NewAuthController(authUsecase useCaseAuth.IAuthUseCase, loggerInstance *logger.Logger) IAuthController {
	return &AuthController{
		authUseCase: authUsecase,
		Logger:      loggerInstance,
	}
}

func (c *AuthController) Login(ctx *gin.Context) {
	c.Logger.Info("User login request")
	var request LoginRequest
	if err := controllers.BindJSON(ctx, &request); err != nil {
		c.Logger.Error("Error binding JSON for login", zap.Error(err))
		appError := domainErrors.NewAppError(err, domainErrors.ValidationError)
		_ = ctx.Error(appError)
		return
	}

	domainUser, authTokens, err := c.authUseCase.Login(request.Email, request.Password)
	if err != nil {
		c.Logger.Error("Login failed", zap.Error(err), zap.String("email", request.Email))
		_ = ctx.Error(err)
		return
	}

	response := LoginResponse{
		Data: UserData{
			UserName:  domainUser.UserName,
			Email:     domainUser.Email,
			FirstName: domainUser.FirstName,
			LastName:  domainUser.LastName,
			Status:    domainUser.Status,
			ID:        domainUser.ID,
		},
		Security: SecurityData{
			JWTAccessToken:            authTokens.AccessToken,
			JWTRefreshToken:           authTokens.RefreshToken,
			ExpirationAccessDateTime:  authTokens.ExpirationAccessDateTime,
			ExpirationRefreshDateTime: authTokens.ExpirationRefreshDateTime,
		},
	}

	c.Logger.Info("Login successful", zap.String("email", request.Email), zap.Int("userID", domainUser.ID))
	ctx.JSON(http.StatusOK, response)
}

func (c *AuthController) GetAccessTokenByRefreshToken(ctx *gin.Context) {
	c.Logger.Info("Token refresh request")
	var request AccessTokenRequest
	if err := controllers.BindJSON(ctx, &request); err != nil {
		c.Logger.Error("Error binding JSON for token refresh", zap.Error(err))
		appError := domainErrors.NewAppError(err, domainErrors.ValidationError)
		_ = ctx.Error(appError)
		return
	}

	domainUser, authTokens, err := c.authUseCase.AccessTokenByRefreshToken(request.RefreshToken)
	if err != nil {
		c.Logger.Error("Token refresh failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response := LoginResponse{
		Data: UserData{
			UserName:  domainUser.UserName,
			Email:     domainUser.Email,
			FirstName: domainUser.FirstName,
			LastName:  domainUser.LastName,
			Status:    domainUser.Status,
			ID:        domainUser.ID,
		},
		Security: SecurityData{
			JWTAccessToken:            authTokens.AccessToken,
			JWTRefreshToken:           authTokens.RefreshToken,
			ExpirationAccessDateTime:  authTokens.ExpirationAccessDateTime,
			ExpirationRefreshDateTime: authTokens.ExpirationRefreshDateTime,
		},
	}

	c.Logger.Info("Token refresh successful", zap.Int("userID", domainUser.ID))
	ctx.JSON(http.StatusOK, response)
}

// InitiateAzureADAuth initiates the Azure AD authentication process
func (c *AuthController) InitiateAzureADAuth(ctx *gin.Context) {
	c.Logger.Info("Azure AD auth initiation request")
	var request AzureADAuthRequest
	if err := controllers.BindJSON(ctx, &request); err != nil {
		c.Logger.Error("Error binding JSON for Azure AD auth initiation", zap.Error(err))
		appError := domainErrors.NewAppError(err, domainErrors.ValidationError)
		_ = ctx.Error(appError)
		return
	}

	authURL, state, err := c.authUseCase.InitiateAzureADAuth()
	if err != nil {
		c.Logger.Error("Azure AD auth initiation failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response := AzureADAuthResponse{
		AuthURL: authURL,
		State:   state,
	}

	c.Logger.Info("Azure AD auth initiation successful", zap.String("state", state))
	ctx.JSON(http.StatusOK, response)
}

// CompleteAzureADAuth completes the Azure AD authentication process
func (c *AuthController) CompleteAzureADAuth(ctx *gin.Context) {
	c.Logger.Info("Azure AD auth completion request")
	var request AzureADCallbackRequest
	if err := controllers.BindJSON(ctx, &request); err != nil {
		c.Logger.Error("Error binding JSON for Azure AD auth completion", zap.Error(err))
		appError := domainErrors.NewAppError(err, domainErrors.ValidationError)
		_ = ctx.Error(appError)
		return
	}

	domainUser, authTokens, err := c.authUseCase.CompleteAzureADAuth(request.Code, request.State)
	if err != nil {
		c.Logger.Error("Azure AD auth completion failed", zap.Error(err))
		_ = ctx.Error(err)
		return
	}

	response := LoginResponse{
		Data: UserData{
			UserName:  domainUser.UserName,
			Email:     domainUser.Email,
			FirstName: domainUser.FirstName,
			LastName:  domainUser.LastName,
			Status:    domainUser.Status,
			ID:        domainUser.ID,
		},
		Security: SecurityData{
			JWTAccessToken:            authTokens.AccessToken,
			JWTRefreshToken:           authTokens.RefreshToken,
			ExpirationAccessDateTime:  authTokens.ExpirationAccessDateTime,
			ExpirationRefreshDateTime: authTokens.ExpirationRefreshDateTime,
		},
	}

	c.Logger.Info("Azure AD auth completion successful", zap.Int("userID", domainUser.ID))
	ctx.JSON(http.StatusOK, response)
}
