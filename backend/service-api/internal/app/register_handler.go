package app

import (
	"watch-party/pkg/auth"
	"watch-party/pkg/model"

	"github.com/gin-gonic/gin"
)

func (a *appServer) RegisterHandlers() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	// middlewares
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// Create JWT middleware
	jwtManager := auth.NewJWTManager(a.config.JWTSecret)
	authMiddleware := auth.AuthMiddleware(jwtManager)
	adminMiddleware := auth.RequireRole(model.RoleAdmin)

	// Health check
	handler.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// API routes
	api := handler.Group("/api/v1")

	// Public routes (no authentication required)
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

	// Admin-only routes (authentication + admin role required)
	adminRoutes := api.Group("/admin")
	adminRoutes.Use(authMiddleware)
	adminRoutes.Use(adminMiddleware)
	{
		// Movies management - Admin only
		adminRoutes.POST("/movies", a.movieController.UploadMovie)
		adminRoutes.GET("/movies", a.movieController.GetMovies)
		adminRoutes.GET("/movies/:id", a.movieController.GetMovie)
		adminRoutes.PUT("/movies/:id", a.movieController.UpdateMovie)
		adminRoutes.DELETE("/movies/:id", a.movieController.DeleteMovie)
		adminRoutes.GET("/movies/:id/stream", a.movieController.GetMovieStreamURL)
		adminRoutes.GET("/my-movies", a.movieController.GetMyMovies)
	}

	// Authenticated user routes
	userRoutes := api.Group("")
	userRoutes.Use(authMiddleware)
	{
		// Room management - authenticated users
		userRoutes.POST("/rooms", a.roomController.CreateRoom)
		userRoutes.GET("/rooms/:id", a.roomController.GetRoom)
		userRoutes.POST("/rooms/:id/invite", a.roomController.InviteUser)
		userRoutes.POST("/rooms/join", a.roomController.JoinRoom)
		userRoutes.GET("/rooms/join", a.roomController.JoinRoomByToken)
		userRoutes.GET("/rooms/join/:room_id", a.roomController.JoinRoomByID)

		// Guest management - host only
		userRoutes.GET("/rooms/:id/guest-requests", a.roomController.GetPendingGuestRequests)
		userRoutes.POST("/rooms/:id/guest-requests/:requestId/approve", a.roomController.ApproveGuestRequest)
	}

	// Public routes (no authentication required)
	publicRoutes := api.Group("")
	{
		// Guest access requests
		publicRoutes.POST("/rooms/:id/request-access", a.roomController.RequestGuestAccess)
		publicRoutes.GET("/guest/validate/:token", a.roomController.ValidateGuestSession)
	}

	// File serving for local storage (if needed)
	// This will serve files from the uploads directory for local storage
	if a.config.Storage.Provider == "local" {
		api.Static("/files", a.config.Storage.LocalPath)
	}

	return handler
}
