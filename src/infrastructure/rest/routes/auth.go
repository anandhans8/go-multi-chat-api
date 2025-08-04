package routes

import (
	authController "go-multi-chat-api/src/infrastructure/rest/controllers/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.RouterGroup, controller authController.IAuthController) {
	routerAuth := router.Group("/auth")
	{
		routerAuth.POST("/login", controller.Login)
		routerAuth.POST("/access-token", controller.GetAccessTokenByRefreshToken)
		routerAuth.POST("/azure-ad/init", controller.InitiateAzureADAuth)
		routerAuth.POST("/azure-ad/callback", controller.CompleteAzureADAuth)
	}
}
