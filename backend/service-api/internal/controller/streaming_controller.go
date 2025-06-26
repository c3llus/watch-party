package controller

import (
	"fmt"
	"net/http"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
	movieService "watch-party/service-api/internal/service/movie"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamingController handles video streaming HTTP requests via signed URLs
type StreamingController struct {
	storageProvider storage.Provider
	movieService    movieService.Service
}

// NewStreamingController creates a new streaming controller
func NewStreamingController(storageProvider storage.Provider, movieService movieService.Service) *StreamingController {
	return &StreamingController{
		storageProvider: storageProvider,
		movieService:    movieService,
	}
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
	masterPath := fmt.Sprintf("transcoded/%s/master.m3u8", movieID.String())

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
	playlistPath := fmt.Sprintf("transcoded/%s/%s.m3u8", movieID.String(), quality)

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
	segmentPath := fmt.Sprintf("transcoded/%s/%s", movieID.String(), segment)

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

	// parse request body to get list of paths
	var request struct {
		Paths []string `json:"paths" binding:"required"`
	}

	err = c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	if len(request.Paths) == 0 || len(request.Paths) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paths must contain 1-100 items"})
		return
	}

	// construct full paths with movie ID prefix
	fullPaths := make([]string, len(request.Paths))
	for i, path := range request.Paths {
		fullPaths[i] = fmt.Sprintf("transcoded/%s/%s", movieID.String(), path)
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
