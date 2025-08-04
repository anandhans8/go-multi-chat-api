package routes

import (
	"go-multi-chat-api/src/infrastructure/di"
	"go-multi-chat-api/src/infrastructure/rest/controllers/user"
	"go-multi-chat-api/src/infrastructure/rest/middlewares"

	"github.com/gin-gonic/gin"
)

func UserRoutes(router *gin.RouterGroup, controller user.IUserController, appContext *di.ApplicationContext) {
	u := router.Group("/user")
	u.Use(middlewares.AuthJWTMiddleware())
	{
		// Normal member operations - any authenticated user can access these
		u.GET("/:id", controller.GetUsersByID)
		u.GET("/search", controller.SearchPaginated)
		u.GET("/search-property", controller.SearchByProperty)

		// Admin-only operations - only users with admin role can access these
		adminCheck := middlewares.RequiresRoleMiddleware("admin", appContext.Logger)

		// Only admin can create new users
		u.POST("/", adminCheck, controller.NewUser)

		// Only admin can get all users
		u.GET("/", adminCheck, controller.GetAllUsers)

		// Only admin can update users
		u.PUT("/:id", adminCheck, controller.UpdateUser)

		// Only admin can delete users
		u.DELETE("/:id", adminCheck, controller.DeleteUser)
	}
}
