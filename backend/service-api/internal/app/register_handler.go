package app

import "github.com/gin-gonic/gin"

func (a *appServer) RegisterHandlers() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	// middlewares
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// TODO: ping upstream services
	handler.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// TODO: modularize
	api := handler.Group("/api/v1")
	{
		// auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/login", a.controller.Login)
			auth.POST("/logout", a.controller.Logout)
		}

		// admin routes
		admin := api.Group("/admin")
		{
			admin.POST("/register", a.controller.RegisterAdmin)
		}

		// user routes
		users := api.Group("/users")
		{
			users.POST("/register", a.controller.RegisterUser)
		}
	}

	return handler
}
