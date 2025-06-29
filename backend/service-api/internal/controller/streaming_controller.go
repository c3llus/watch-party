package controller

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
	movieService "watch-party/service-api/internal/service/movie"
	roomService "watch-party/service-api/internal/service/room"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamingController handles video streaming HTTP requests via signed URLs
type StreamingController struct {
	storageProvider storage.Provider
	movieService    movieService.Service
	roomService     *roomService.Service
}

// NewStreamingController creates a new streaming controller
func NewStreamingController(storageProvider storage.Provider, movieService movieService.Service, roomService *roomService.Service) *StreamingController {
	return &StreamingController{
		storageProvider: storageProvider,
		movieService:    movieService,
		roomService:     roomService,
	}
}

// generateAuthHash creates a deterministic hash for caching based on user access level
func (sc *StreamingController) generateAuthHash(userID *uuid.UUID, guestToken string, movieID uuid.UUID) string {
	var authString string
	if userID != nil {
		authString = fmt.Sprintf("user:%s:movie:%s", userID.String(), movieID.String())
	} else {
		authString = fmt.Sprintf("guest:%s:movie:%s", guestToken, movieID.String())
	}

	hash := sha256.Sum256([]byte(authString))
	return fmt.Sprintf("%x", hash)[:16] // first 16 chars for brevity
}

// validateAccess validates user or guest access and returns auth hash
func (sc *StreamingController) validateAccess(c *gin.Context, movieID uuid.UUID) (string, error) {
	// check for guest token first
	guestToken := c.Query("guestToken")
	var userID *uuid.UUID

	if guestToken != "" {
		// validate guest access
		if len(guestToken) < 32 {
			return "", fmt.Errorf("invalid guest token format")
		}

		guestSession, err := sc.roomService.ValidateGuestSession(c.Request.Context(), guestToken)
		if err != nil {
			return "", fmt.Errorf("invalid or expired guest token")
		}

		if time.Now().After(guestSession.ExpiresAt) {
			return "", fmt.Errorf("guest session expired")
		}

		hasAccess, err := sc.roomService.CheckRoomContainsMovie(c.Request.Context(), guestSession.RoomID, movieID)
		if err != nil || !hasAccess {
			return "", fmt.Errorf("guest does not have access to this movie")
		}

		return sc.generateAuthHash(nil, guestToken, movieID), nil
	} else {
		// validate user access
		if userIDValue, exists := c.Get("user_id"); exists {
			if uid, ok := userIDValue.(uuid.UUID); ok {
				userID = &uid
			}
		}

		if userID == nil {
			return "", fmt.Errorf("authentication required")
		}

		hasAccess, err := sc.roomService.CheckUserMovieAccess(c.Request.Context(), *userID, movieID)
		if err != nil || !hasAccess {
			return "", fmt.Errorf("user does not have access to this movie")
		}

		return sc.generateAuthHash(userID, "", movieID), nil
	}
}

// generateAuthHashFromContext creates auth hash from middleware context
func (sc *StreamingController) generateAuthHashFromContext(c *gin.Context, movieID uuid.UUID) string {
	authType := c.GetString("auth_type")

	if authType == "jwt" {
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(uuid.UUID); ok {
				return sc.generateAuthHash(&uid, "", movieID)
			}
		}
	} else if authType == "guest" {
		guestToken := c.Query("guestToken")
		if guestToken == "" {
			guestToken = c.GetHeader("X-Guest-Token")
		}
		return sc.generateAuthHash(nil, guestToken, movieID)
	}

	// fallback - shouldn't happen with proper middleware
	return sc.generateAuthHash(nil, "", movieID)
}

// ProxyMasterPlaylist handles GET /api/v1/stream/{movieId}/playlist.m3u8
func (sc *StreamingController) ProxyMasterPlaylist(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// generate auth hash for caching (auth already validated by middleware)
	authHash := sc.generateAuthHashFromContext(c, movieID)

	// verify movie exists and is available
	movie, err := sc.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	if movie.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "video not ready",
			"status": movie.Status,
		})
		return
	}

	// construct path for master playlist
	masterPath := fmt.Sprintf("hls/%s/master.m3u8", movieID.String())

	// get and rewrite master playlist to use proxy URLs
	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), masterPath, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,          // 2 hours expiration
		CacheControl: "public, max-age=3600", // cache for 1 hour
		ContentType:  "application/vnd.apple.mpegurl",
	})
	if err != nil {
		logger.Error(err, "failed to generate signed URL for master playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate playlist URL"})
		return
	}

	// fetch the master playlist content
	resp, err := http.Get(signedURL)
	if err != nil {
		logger.Error(err, "failed to fetch master playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch playlist"})
		return
	}
	defer resp.Body.Close()

	// read playlist content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "failed to read master playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read playlist"})
		return
	}

	// rewrite playlist to use proxy URLs
	playlistContent := string(content)
	lines := strings.Split(playlistContent, "\n")

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// rewrite variant playlist URLs to go through proxy
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && strings.HasSuffix(trimmedLine, ".m3u8") {
			// convert "1080p.m3u8" to "/api/v1/stream/movieID/1080p/playlist.m3u8"
			quality := strings.TrimSuffix(trimmedLine, ".m3u8")
			proxyURL := fmt.Sprintf("/api/v1/stream/%s/%s/playlist.m3u8", movieID.String(), quality)
			if guestToken := c.Query("guestToken"); guestToken != "" {
				proxyURL += fmt.Sprintf("?guestToken=%s", guestToken)
			}
			lines[i] = proxyURL
		}
	}

	rewrittenContent := strings.Join(lines, "\n")

	// set CDN-friendly cache headers with auth awareness
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Vary", "Authorization")
	c.Header("X-Auth-Hash", authHash)
	c.Header("Content-Type", "application/vnd.apple.mpegurl")

	// return rewritten playlist content
	c.String(http.StatusOK, rewrittenContent)
}

// ProxyVideoSegment handles GET /api/v1/stream/{movieId}/{quality}/{segment}
func (sc *StreamingController) ProxyVideoSegment(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	quality := c.Param("quality")
	segment := c.Param("segment")

	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// generate auth hash for caching (auth already validated by middleware)
	authHash := sc.generateAuthHashFromContext(c, movieID)

	// validate parameters
	if quality == "" || segment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quality and segment parameters required"})
		return
	}

	// verify movie exists and is available
	movie, err := sc.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	if movie.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "video not ready",
			"status": movie.Status,
		})
		return
	}

	// construct path for video segment
	segmentPath := fmt.Sprintf("hls/%s/%s/%s", movieID.String(), quality, segment)

	// generate signed URL with long CDN cache for segments
	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), segmentPath, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 24,          // 24 hours expiration for segments
		CacheControl: "public, max-age=86400", // cache segments for 24 hours
		ContentType:  "video/mp2t",
	})
	if err != nil {
		logger.Error(err, "failed to generate signed URL for video segment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate segment URL"})
		return
	}

	// set aggressive CDN cache headers for segments
	c.Header("Cache-Control", "public, max-age=86400") // 24 hours
	c.Header("Vary", "Authorization")
	c.Header("X-Auth-Hash", authHash)
	c.Header("X-Content-Type", "video/mp2t")

	// redirect to signed URL - CDN will cache this redirect with auth hash
	c.Redirect(http.StatusFound, signedURL)
}

// ProxyVideoSegmentTimeWindow handles GET /api/v1/stream/{movieId}/{quality}/{segment} with time-window caching
func (sc *StreamingController) ProxyVideoSegmentTimeWindow(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	quality := c.Param("quality")
	segment := c.Param("segment")

	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// validate access (no auth hash needed for time-window approach)
	_, err = sc.validateAccess(c, movieID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// validate parameters
	if quality == "" || segment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quality and segment parameters required"})
		return
	}

	// verify movie exists and is available
	movie, err := sc.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	if movie.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "video not ready",
			"status": movie.Status,
		})
		return
	}

	// generate time window (rounded to 5-minute intervals)
	now := time.Now()
	timeWindow := now.Truncate(5 * time.Minute).Unix()

	// construct path for video segment with time window
	segmentPath := fmt.Sprintf("hls/%s/%s/%s", movieID.String(), quality, segment)

	// generate signed URL that's valid for the current time window
	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), segmentPath, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 24,        // 24 hours expiration for segments
		CacheControl: "public, max-age=300", // cache for 5 minutes (time window)
		ContentType:  "video/mp2t",
	})
	if err != nil {
		logger.Error(err, "failed to generate signed URL for video segment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate segment URL"})
		return
	}

	// set CDN cache headers with time window
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes
	c.Header("X-Time-Window", fmt.Sprintf("%d", timeWindow))
	c.Header("X-Content-Type", "video/mp2t")

	// add time window to URL for CDN cache differentiation
	if strings.Contains(signedURL, "?") {
		signedURL += fmt.Sprintf("&tw=%d", timeWindow)
	} else {
		signedURL += fmt.Sprintf("?tw=%d", timeWindow)
	}

	// redirect to signed URL with time window
	c.Redirect(http.StatusFound, signedURL)
}

// ProxyQualityPlaylist handles GET /api/v1/stream/{movieId}/{quality}/playlist.m3u8
func (sc *StreamingController) ProxyQualityPlaylist(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	quality := c.Param("quality")

	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// generate auth hash for caching (auth already validated by middleware)
	authHash := sc.generateAuthHashFromContext(c, movieID)

	// validate quality parameter
	if quality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quality parameter required"})
		return
	}

	// verify movie exists and is available
	movie, err := sc.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	if movie.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "video not ready",
			"status": movie.Status,
		})
		return
	}

	// construct path for quality-specific playlist
	playlistPath := fmt.Sprintf("hls/%s/%s.m3u8", movieID.String(), quality)

	// get and rewrite quality playlist to use proxy URLs for segments
	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), playlistPath, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,          // 2 hours expiration
		CacheControl: "public, max-age=1800", // cache for 30 minutes
		ContentType:  "application/vnd.apple.mpegurl",
	})
	if err != nil {
		logger.Error(err, "failed to generate signed URL for quality playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate playlist URL"})
		return
	}

	// fetch the quality playlist content
	resp, err := http.Get(signedURL)
	if err != nil {
		logger.Error(err, "failed to fetch quality playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch playlist"})
		return
	}
	defer resp.Body.Close()

	// read playlist content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "failed to read quality playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read playlist"})
		return
	}

	// rewrite playlist to use proxy URLs for segments
	playlistContent := string(content)
	lines := strings.Split(playlistContent, "\n")

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// rewrite segment URLs to go through proxy
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && strings.HasSuffix(trimmedLine, ".ts") {
			// convert "segment0.ts" to "/api/v1/stream/movieID/quality/segment0.ts"
			proxyURL := fmt.Sprintf("/api/v1/stream/%s/%s/%s", movieID.String(), quality, trimmedLine)
			if guestToken := c.Query("guestToken"); guestToken != "" {
				proxyURL += fmt.Sprintf("?guestToken=%s", guestToken)
			}
			lines[i] = proxyURL
		}
	}

	rewrittenContent := strings.Join(lines, "\n")

	// set CDN cache headers with auth awareness
	c.Header("Cache-Control", "public, max-age=1800") // 30 minutes
	c.Header("Vary", "Authorization")
	c.Header("X-Auth-Hash", authHash)
	c.Header("Content-Type", "application/vnd.apple.mpegurl")

	// return rewritten playlist content
	c.String(http.StatusOK, rewrittenContent)
}

// GetMasterPlaylistURL handles GET /api/v1/stream/{movieId}/playlist.m3u8
// Returns a signed URL for direct access to the master playlist
func (sc *StreamingController) GetMasterPlaylistURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// construct path for master playlist
	masterPath := fmt.Sprintf("hls/%s/master.m3u8", movieID.String())

	// generate signed URL for CDN access
	opts := &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,         // 2 hours expiration
		CacheControl: "public, max-age=300", // cache for 5 minutes
		ContentType:  "application/vnd.apple.mpegurl",
	}

	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), masterPath, opts)
	if err != nil {
		logger.Error(err, "failed to generate signed URL for master playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate playlist URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        signedURL,
		"expires_in": opts.ExpiresIn.Seconds(),
		"type":       "master_playlist",
	})
}

// GetMediaPlaylistURL handles GET /api/v1/stream/{movieId}/{quality}/playlist.m3u8
// Returns a signed URL for direct access to the quality-specific playlist
func (sc *StreamingController) GetMediaPlaylistURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	quality := c.Param("quality")

	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// validate quality parameter
	if quality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quality parameter required"})
		return
	}

	// construct path for quality-specific playlist
	playlistPath := fmt.Sprintf("hls/%s/%s.m3u8", movieID.String(), quality)

	// generate signed URL for CDN access
	opts := &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,         // 2 hours expiration
		CacheControl: "public, max-age=300", // cache for 5 minutes
		ContentType:  "application/vnd.apple.mpegurl",
	}

	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), playlistPath, opts)
	if err != nil {
		logger.Error(err, "failed to generate signed URL for media playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate playlist URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        signedURL,
		"expires_in": opts.ExpiresIn.Seconds(),
		"type":       "media_playlist",
		"quality":    quality,
	})
}

// GetVideoSegmentURL handles GET /api/v1/stream/{movieId}/{quality}/segment{n}.ts
// Returns a signed URL for direct access to the video segment
func (sc *StreamingController) GetVideoSegmentURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	quality := c.Param("quality")
	segment := c.Param("segment")

	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// validate parameters
	if quality == "" || segment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quality and segment parameters required"})
		return
	}

	// construct path for video segment
	segmentPath := fmt.Sprintf("hls/%s/%s", movieID.String(), segment)

	// generate signed URL for CDN access
	opts := &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 24,          // 24 hours expiration for segments
		CacheControl: "public, max-age=86400", // cache segments for 24 hours
		ContentType:  "video/mp2t",
	}

	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), segmentPath, opts)
	if err != nil {
		logger.Error(err, "failed to generate signed URL for video segment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate segment URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        signedURL,
		"expires_in": opts.ExpiresIn.Seconds(),
		"type":       "video_segment",
		"quality":    quality,
		"segment":    segment,
	})
}

// GetVideoURL handles GET /api/v1/stream/{movieId}/video
// Returns a signed URL for direct access to the original video file
func (sc *StreamingController) GetVideoURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// for now, we need to find the original file from the movie record
	// since the uploads path includes a timestamp. Let's modify this to query the database
	// for the actual original file path
	videoPath := fmt.Sprintf("uploads/%s", movieID.String()) // placeholder - will need movie service

	// generate signed URL for CDN access with range support
	opts := &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 4,          // 4 hours expiration for video files
		CacheControl: "public, max-age=3600", // cache for 1 hour
		ContentType:  "video/mp4",
	}

	signedURL, err := sc.storageProvider.GenerateCDNSignedURL(c.Request.Context(), videoPath, opts)
	if err != nil {
		logger.Error(err, "failed to generate signed URL for video")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate video URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":            signedURL,
		"expires_in":     opts.ExpiresIn.Seconds(),
		"type":           "video",
		"supports_range": true, // CDN/storage providers typically support HTTP Range requests
	})
}

// GetMultipleURLs handles POST /api/v1/stream/{movieId}/urls
// Returns signed URLs for multiple files (playlists and segments) in a single request
func (sc *StreamingController) GetMultipleURLs(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// parse request body
	var request struct {
		Files []string `json:"files" binding:"required"`
	}

	err = c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// validate file count to prevent abuse
	if len(request.Files) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many files requested (max 100)"})
		return
	}

	// verify movie exists and is available
	movie, err := sc.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		logger.Error(err, "failed to get movie for batch URL access")
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}

	if movie.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "video not ready",
			"status": movie.Status,
		})
		return
	}

	// build full paths for requested files
	basePath := "hls/" + movieID.String() + "/"
	fullPaths := make([]string, len(request.Files))
	for i, file := range request.Files {
		// check if file already contains the full path (avoid duplication)
		if strings.HasPrefix(file, basePath) {
			fullPaths[i] = file
		} else {
			fullPaths[i] = basePath + file
		}
	}

	// generate signed URLs for all paths
	opts := &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,          // 2 hours expiration
		CacheControl: "public, max-age=3600", // cache for 1 hour
	}

	signedURLs, err := sc.storageProvider.GenerateSignedURLs(c.Request.Context(), fullPaths, opts)
	if err != nil {
		logger.Error(err, "failed to generate multiple signed URLs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate URLs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"urls":       signedURLs,
		"expires_in": opts.ExpiresIn.Seconds(),
		"count":      len(signedURLs),
	})
}

// GetMasterPlaylistProxy handles GET /api/v1/stream/{movieId}/playlist.m3u8
func (sc *StreamingController) GetMasterPlaylistProxy(c *gin.Context) {
	sc.ProxyMasterPlaylist(c)
}

// GetMediaPlaylistProxy handles GET /api/v1/stream/{movieId}/{quality}/playlist.m3u8
func (sc *StreamingController) GetMediaPlaylistProxy(c *gin.Context) {
	sc.ProxyQualityPlaylist(c)
}

// GetVideoSegmentProxy handles GET /api/v1/stream/{movieId}/{quality}/{segment}
func (sc *StreamingController) GetVideoSegmentProxy(c *gin.Context) {
	sc.ProxyVideoSegment(c)
}
