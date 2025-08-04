package routes

import (
	"go-multi-chat-api/src/infrastructure/rest/controllers/send"
	"go-multi-chat-api/src/infrastructure/rest/middlewares"

	"github.com/gin-gonic/gin"
)

func SendRoutes(router *gin.RouterGroup, controller send.ISendController) {
	signalRoute := router.Group("/send")
	signalRoute.Use(middlewares.AuthJWTMiddleware())
	{
		signalRoute.POST("/message", controller.Message)
		signalRoute.GET("/message/:id/status", controller.GetMessageStatus)
	}
}
