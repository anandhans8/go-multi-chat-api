package routes

import (
	"net/http"

	"go-multi-chat-api/src/infrastructure/di"

	"github.com/gin-gonic/gin"
)

func ApplicationRouter(router *gin.Engine, appContext *di.ApplicationContext) {
	v1 := router.Group("/v1")

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Service is running",
		})
	})

	AuthRoutes(v1, appContext.AuthController)
	UserRoutes(v1, appContext.UserController, appContext)
	SignalRoutes(v1, appContext.SignalController)
	SendRoutes(v1, appContext.SendController)
}
