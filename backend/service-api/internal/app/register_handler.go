package app

import (
	"watch-party/pkg/auth"
	"watch-party/pkg/model"
	middleware "watch-party/service-api/internal/app/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (a *appServer) RegisterHandlers() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	handler := gin.New()

	// middlewares
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// cors middleware
	// corsConfig := cors.Config{
	// 	AllowOrigins:     a.config.CORS.AllowedOrigins,
	// 	AllowMethods:     a.config.CORS.AllowedMethods,
	// 	AllowHeaders:     a.config.CORS.AllowedHeaders,
	// 	AllowCredentials: true,
	// }
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Content-Type", "Authorization", "X-Requested-With", "X-Guest-Token"}
	handler.Use(cors.New(corsConfig))

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

		// user routes - public registration for freemium users
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
		userRoutes.GET("/rooms", a.roomController.GetRooms)
		userRoutes.GET("/rooms/:id", a.roomController.GetRoom)
		userRoutes.POST("/rooms/:id/invite", a.roomController.InviteUser)
		userRoutes.POST("/rooms/join", a.roomController.JoinRoom)
		userRoutes.GET("/rooms/join", a.roomController.JoinRoomByToken)
		userRoutes.GET("/rooms/join/:room_id", a.roomController.JoinRoomByID)

		// guest management - host only
		userRoutes.GET("/rooms/:id/guest-requests", a.roomController.GetPendingGuestRequests)
		userRoutes.POST("/rooms/:id/guest-requests/:requestId/approve", a.roomController.ApproveGuestRequest)

		// room access management - for authenticated users
		userRoutes.POST("/rooms/:id/room-access", a.roomController.RequestRoomAccess)
		userRoutes.GET("/rooms/:id/room-access", a.roomController.GetPendingRoomAccessRequests)
		userRoutes.POST("/rooms/:id/room-access/:userId/approve", a.roomController.ApproveRoomAccessRequest)
		userRoutes.GET("/rooms/:id/room-access/status", a.roomController.CheckRoomAccessRequestStatus)
	}

	// public routes (no authentication required)
	publicRoutes := api.Group("")
	{
		// guest access requests (no auth needed to request access)
		publicRoutes.POST("/rooms/:id/request-access", a.roomController.RequestGuestAccess)
		publicRoutes.GET("/guest/validate/:token", a.roomController.ValidateGuestSession)
		publicRoutes.GET("/guest-requests/:requestId/status", a.roomController.CheckGuestRequestStatus)
	}

	// guest protected routes (require guest token authentication)
	guestAuth := middleware.GuestAuthForRoom(a.roomService)
	guestRoutes := api.Group("/guest")
	guestRoutes.Use(guestAuth)
	{
		// guest access to room info (requires guest token)
		guestRoutes.GET("/rooms/:id", a.roomController.GetRoomForGuest)
	}

	// webhook routes (no authentication required for external services)
	webhookRoutes := api.Group("/webhooks")
	{
		// upload completion webhooks
		webhookRoutes.POST("/upload-complete", a.webhookController.HandleUploadComplete)
	}

	// CDN-friendly video access routes (returns signed URLs)
	streamingAuth := middleware.StreamingAuthMiddleware(jwtManager, a.roomService)
	videoRoutes := api.Group("/videos")
	videoRoutes.Use(streamingAuth) // support both JWT and guest token authentication
	{
		videoRoutes.GET("/:movieId/hls", a.videoAccessController.GetHLSMasterPlaylistURL)
		videoRoutes.POST("/:movieId/urls", a.videoAccessController.GetVideoFileURLs)
		videoRoutes.GET("/:movieId/direct", a.videoAccessController.GetDirectVideoURL)
		videoRoutes.POST("/:movieId/seek", a.videoAccessController.GetSegmentByTime)
	}

	return handler
}
