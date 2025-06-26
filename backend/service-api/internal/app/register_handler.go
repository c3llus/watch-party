package app

import (
	"watch-party/pkg/auth"
	"watch-party/pkg/model"
	middleware "watch-party/service-api/internal/app/middleware"

	"github.com/gin-gonic/gin"
)

func (a *appServer) RegisterHandlers() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	// middlewares
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// create JWT middleware
	jwtManager := auth.NewJWTManager(a.config.JWTSecret)
	authMiddleware := auth.AuthMiddleware(jwtManager)
	adminMiddleware := auth.RequireRole(model.RoleAdmin)

	// health check
	handler.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// api routes
	api := handler.Group("/api/v1")

	// public routes (no authentication required)
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

	// admin-only routes (authentication + admin role required)
	adminRoutes := api.Group("/admin")
	adminRoutes.Use(authMiddleware)
	adminRoutes.Use(adminMiddleware)
	{
		// movies management - admin only
		adminRoutes.POST("/movies", a.movieController.UploadMovie)
		adminRoutes.GET("/movies", a.movieController.GetMovies)
		adminRoutes.GET("/movies/:id", a.movieController.GetMovie)
		adminRoutes.GET("/movies/:id/status", a.movieController.GetMovieStatus)
		adminRoutes.PUT("/movies/:id", a.movieController.UpdateMovie)
		adminRoutes.DELETE("/movies/:id", a.movieController.DeleteMovie)
		adminRoutes.GET("/movies/:id/stream", a.movieController.GetMovieStreamURL)
		adminRoutes.GET("/my-movies", a.movieController.GetMyMovies)
	}

	// authenticated user routes
	userRoutes := api.Group("")
	userRoutes.Use(authMiddleware)
	{
		// room management - authenticated users
		userRoutes.POST("/rooms", a.roomController.CreateRoom)
		userRoutes.GET("/rooms/:id", a.roomController.GetRoom)
		userRoutes.POST("/rooms/:id/invite", a.roomController.InviteUser)
		userRoutes.POST("/rooms/join", a.roomController.JoinRoom)
		userRoutes.GET("/rooms/join", a.roomController.JoinRoomByToken)
		userRoutes.GET("/rooms/join/:room_id", a.roomController.JoinRoomByID)

		// guest management - host only
		userRoutes.GET("/rooms/:id/guest-requests", a.roomController.GetPendingGuestRequests)
		userRoutes.POST("/rooms/:id/guest-requests/:requestId/approve", a.roomController.ApproveGuestRequest)
	}

	// public routes (no authentication required)
	publicRoutes := api.Group("")
	{
		// guest access requests
		publicRoutes.POST("/rooms/:id/request-access", a.roomController.RequestGuestAccess)
		publicRoutes.GET("/guest/validate/:token", a.roomController.ValidateGuestSession)
	}

	// webhook routes (no authentication required for external services)
	webhookRoutes := api.Group("/webhooks")
	{
		// upload completion webhooks
		webhookRoutes.POST("/upload-complete", a.webhookController.HandleUploadComplete)
	}

	// streaming routes (protected by streaming-specific authentication)
	streamingAuth := middleware.StreamingAuthMiddleware(jwtManager, a.roomService)

	// cDN-friendly video access routes (returns signed URLs)
	videoRoutes := api.Group("/videos")
	videoRoutes.Use(authMiddleware) // use simple JWT auth instead of streaming auth
	{
		videoRoutes.GET("/:movieId/hls", a.videoAccessController.GetHLSMasterPlaylistURL)
		videoRoutes.POST("/:movieId/urls", a.videoAccessController.GetVideoFileURLs)
		videoRoutes.GET("/:movieId/direct", a.videoAccessController.GetDirectVideoURL)
	}

	// legacy streaming routes (direct streaming - kept for backward compatibility)
	streamRoutes := api.Group("/stream")
	streamRoutes.Use(streamingAuth)
	{
		// cDN-friendly signed URL endpoints for HLS streaming
		streamRoutes.GET("/:movieId/playlist.m3u8", a.streamingController.GetMasterPlaylistURL)
		streamRoutes.GET("/:movieId/:quality/playlist.m3u8", a.streamingController.GetMediaPlaylistURL)
		streamRoutes.GET("/:movieId/:quality/:segment", a.streamingController.GetVideoSegmentURL)

		// Direct video signed URL with range support
		streamRoutes.GET("/:movieId/video", a.streamingController.GetVideoURL)

		// Bulk signed URL generation endpoint
		streamRoutes.POST("/:movieId/urls", a.streamingController.GetMultipleURLs)
	}

	return handler
}
