package controller

import (
	"net/http"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
	movieService "watch-party/service-api/internal/service/movie"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VideoAccessController handles CDN-friendly video access requests
type VideoAccessController struct {
	storageProvider storage.Provider
	movieService    movieService.Service
}

// NewVideoAccessController creates a new video access controller
func NewVideoAccessController(storageProvider storage.Provider, movieService movieService.Service) *VideoAccessController {
	return &VideoAccessController{
		storageProvider: storageProvider,
		movieService:    movieService,
	}
}

// GetHLSMasterPlaylistURL handles GET /api/v1/videos/{movieId}/hls
func (vac *VideoAccessController) GetHLSMasterPlaylistURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// get movie to verify it exists and is available
	movie, err := vac.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		logger.Error(err, "failed to get movie for HLS access")
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

	// generate signed URL for master playlist
	masterPath := "transcoded/" + movieID.String() + "/master.m3u8"

	signedURL, err := vac.storageProvider.GenerateCDNSignedURL(c.Request.Context(), masterPath, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,          // 2 hours for HLS master playlist
		CacheControl: "public, max-age=3600", // cache for 1 hour
		ContentType:  "application/vnd.apple.mpegurl",
	})
	if err != nil {
		logger.Error(err, "failed to generate signed URL for HLS master playlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate video access URL"})
		return
	}

	// return CDN-friendly response
	response := gin.H{
		"movie_id":   movieID.String(),
		"hls_url":    signedURL,
		"expires_at": time.Now().Add(time.Hour * 2).Format(time.RFC3339),
		"cdn_info": gin.H{
			"cacheable":      true,
			"cache_duration": "1h",
		},
	}

	// set cache headers for this API response
	c.Header("Cache-Control", "private, max-age=300") // cache API response for 5 minutes
	c.JSON(http.StatusOK, response)
}

// GetVideoFileURLs handles POST /api/v1/videos/{movieId}/urls
func (vac *VideoAccessController) GetVideoFileURLs(c *gin.Context) {
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
	movie, err := vac.movieService.GetMovie(c.Request.Context(), movieID)
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
	basePath := "transcoded/" + movieID.String() + "/"
	fullPaths := make([]string, len(request.Files))
	for i, file := range request.Files {
		fullPaths[i] = basePath + file
	}

	// generate signed URLs for all files
	signedURLs, err := vac.storageProvider.GenerateSignedURLs(c.Request.Context(), fullPaths, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,           // 2 hours for video segments
		CacheControl: "public, max-age=86400", // cache segments for 24 hours
	})
	if err != nil {
		logger.Error(err, "failed to generate batch signed URLs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate video file URLs"})
		return
	}

	// map back to original file names
	fileURLs := make(map[string]string)
	for i, file := range request.Files {
		fullPath := fullPaths[i]
		url, exists := signedURLs[fullPath]
		if exists {
			fileURLs[file] = url
		}
	}

	response := gin.H{
		"movie_id":   movieID.String(),
		"file_urls":  fileURLs,
		"expires_at": time.Now().Add(time.Hour * 2).Format(time.RFC3339),
		"cdn_info": gin.H{
			"cacheable":      true,
			"cache_duration": "24h",
		},
	}

	c.Header("Cache-Control", "private, max-age=300") // cache API response for 5 minutes
	c.JSON(http.StatusOK, response)
}

// GetDirectVideoURL handles GET /api/v1/videos/{movieId}/direct
func (vac *VideoAccessController) GetDirectVideoURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// get movie to verify it exists and get original file path
	movie, err := vac.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		logger.Error(err, "failed to get movie for direct access")
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

	// generate signed URL for original video file
	signedURL, err := vac.storageProvider.GenerateCDNSignedURL(c.Request.Context(), movie.OriginalFilePath, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 4,          // 4 hours for direct video access
		CacheControl: "public, max-age=3600", // cache for 1 hour
		ContentType:  movie.MimeType,
	})
	if err != nil {
		logger.Error(err, "failed to generate signed URL for direct video")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate direct video URL"})
		return
	}

	response := gin.H{
		"movie_id":        movieID.String(),
		"direct_url":      signedURL,
		"expires_at":      time.Now().Add(time.Hour * 4).Format(time.RFC3339),
		"file_size":       movie.FileSize,
		"mime_type":       movie.MimeType,
		"supports_ranges": true,
		"cdn_info": gin.H{
			"cacheable":      true,
			"cache_duration": "1h",
		},
	}

	c.Header("Cache-Control", "private, max-age=300") // cache API response for 5 minutes
	c.JSON(http.StatusOK, response)
}
