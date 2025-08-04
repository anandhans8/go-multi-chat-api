package routes

import (
	"go-multi-chat-api/src/infrastructure/rest/controllers/signal"
	"go-multi-chat-api/src/infrastructure/rest/middlewares"

	"github.com/gin-gonic/gin"
)

func SignalRoutes(router *gin.RouterGroup, controller signal.ISignalController) {
	signalRoute := router.Group("/signal")
	signalRoute.Use(middlewares.AuthJWTMiddleware())
	{
		signalRoute.POST("/register/:number", controller.RegisterNumber)
		signalRoute.POST("/register/:number/verify/:token", controller.VerifyRegisteredNumber)
		signalRoute.GET("/qrcode", controller.GetQrCodeLink)
		signalRoute.POST("/send", controller.Send)
	}
}
