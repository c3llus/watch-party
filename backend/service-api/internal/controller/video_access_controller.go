package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
	movieService "watch-party/service-api/internal/service/movie"
	roomService "watch-party/service-api/internal/service/room"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VideoAccessController handles CDN-friendly video access requests
type VideoAccessController struct {
	storageProvider storage.Provider
	movieService    movieService.Service
	roomService     *roomService.Service
}

// NewVideoAccessController creates a new video access controller
func NewVideoAccessController(storageProvider storage.Provider, movieService movieService.Service, roomService *roomService.Service) *VideoAccessController {
	return &VideoAccessController{
		storageProvider: storageProvider,
		movieService:    movieService,
		roomService:     roomService,
	}
}

// validateGuestAccess validates guest token and checks if guest has access to the movie
func (vac *VideoAccessController) validateGuestAccess(ctx *gin.Context, guestToken string, movieID uuid.UUID) (*uuid.UUID, error) {
	if len(guestToken) < 32 {
		return nil, fmt.Errorf("invalid guest token format")
	}

	// get guest session by token
	guestSession, err := vac.roomService.ValidateGuestSession(ctx.Request.Context(), guestToken)
	if err != nil {
		logger.Error(err, "failed to validate guest session")
		return nil, fmt.Errorf("invalid or expired guest token: %w", err)
	}

	// check if session is expired
	if time.Now().After(guestSession.ExpiresAt) {
		return nil, fmt.Errorf("guest session expired")
	}

	// check if the room contains the requested movie
	hasAccess, err := vac.roomService.CheckRoomContainsMovie(ctx.Request.Context(), guestSession.RoomID, movieID)
	if err != nil {
		logger.Error(err, "failed to check room movie access for guest")
		return nil, fmt.Errorf("failed to validate movie access: %w", err)
	}

	if !hasAccess {
		return nil, fmt.Errorf("guest does not have access to this movie")
	}

	return &guestSession.RoomID, nil
}

// validateUserAccess validates user access to the movie
func (vac *VideoAccessController) validateUserAccess(ctx *gin.Context, userID uuid.UUID, movieID uuid.UUID) error {
	// check if user has access to this specific movie through room membership
	hasAccess, err := vac.roomService.CheckUserMovieAccess(ctx.Request.Context(), userID, movieID)
	if err != nil {
		logger.Error(err, "failed to check user movie access")
		return fmt.Errorf("failed to validate movie access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("user does not have access to this movie")
	}

	return nil
}

// GetHLSMasterPlaylistURL handles GET /api/v1/videos/{movieId}/hls
func (vac *VideoAccessController) GetHLSMasterPlaylistURL(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// authentication is already handled by middleware

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
	masterPath := "hls/" + movieID.String() + "/master.m3u8"

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

	// authentication is already handled by middleware

	// validate file count to prevent abuse and ensure reasonable batch sizes
	if len(request.Files) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many files requested (max 1000)"})
		return
	}

	// log the request for monitoring
	authType := c.GetString("auth_type")
	logger.Infof("batch URL request for movieID=%s, fileCount=%d, authType=%s",
		movieID.String(), len(request.Files), authType)

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

	logger.Infof("generating signed URLs for movieID=%s, basePath=%s, files=%v", movieID.String(), basePath, fullPaths)

	// generate signed URLs for all files with enhanced security
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

	// check for guest token first
	guestToken := c.Query("guestToken")
	var userID *uuid.UUID

	if guestToken != "" {
		// validate guest has access to a room that contains this movie
		roomID, err := vac.validateGuestAccess(c, guestToken, movieID)
		if err != nil {
			logger.Error(err, "guest access validation failed for direct video")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
			return
		}
		logger.Infof("guest access validated for direct video: movie %s in room %s", movieID.String(), roomID.String())
	} else {
		// for authenticated users, extract user ID from JWT context
		if userIDValue, exists := c.Get("user_id"); exists {
			if uid, ok := userIDValue.(uuid.UUID); ok {
				userID = &uid
			}
		}

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// validate user has access to this specific movie
		err = vac.validateUserAccess(c, *userID, movieID)
		if err != nil {
			logger.Error(err, "user access validation failed for direct video")
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		logger.Infof("user access validated for direct video: movie %s by user %s", movieID.String(), userID.String())
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

// GetSegmentByTime handles POST /api/v1/videos/{movieId}/seek
// Returns segment information for a specific timestamp to support seeking
func (vac *VideoAccessController) GetSegmentByTime(c *gin.Context) {
	movieIDStr := c.Param("movieId")
	movieID, err := uuid.Parse(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// parse request body
	var request struct {
		Time         float64 `json:"time" binding:"required"` // target time in seconds
		Quality      string  `json:"quality,omitempty"`       // optional quality (e.g., "1080p"), defaults to highest
		PreloadCount int     `json:"preload_count,omitempty"` // number of segments to preload after target (default: 3)
	}

	err = c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// set defaults
	if request.PreloadCount <= 0 {
		request.PreloadCount = 3
	}
	if request.PreloadCount > 10 {
		request.PreloadCount = 10 // limit to prevent abuse
	}

	// authentication is already handled by middleware
	authType := c.GetString("auth_type")
	logger.Infof("seek request for movieID=%s, time=%.2fs, quality=%s, authType=%s",
		movieID.String(), request.Time, request.Quality, authType)

	// verify movie exists and is available
	movie, err := vac.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		logger.Error(err, "failed to get movie for seek")
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

	// get playlist info from storage to calculate segment timing
	basePath := "hls/" + movieID.String() + "/"

	// determine quality - for now use 1080p as default
	quality := request.Quality
	if quality == "" {
		quality = "1080p"
	}

	playlistPath := basePath + quality + "/playlist.m3u8"

	// get signed URL for playlist to read segment information
	playlistURLs, err := vac.storageProvider.GenerateSignedURLs(c.Request.Context(), []string{playlistPath}, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Minute * 10, // short expiry for playlist
		CacheControl: "private, max-age=60",
	})
	if err != nil {
		logger.Error(err, "failed to generate playlist URL for seek")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access video playlist"})
		return
	}

	playlistURL, exists := playlistURLs[playlistPath]
	if !exists {
		logger.Error(fmt.Errorf("playlist URL not generated"), "playlist URL missing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "playlist URL not available"})
		return
	}

	// fetch and parse playlist to find target segment
	// NOTE: In a production system, you'd want to cache playlist parsing results
	segments, totalDuration, err := vac.parsePlaylistForSeek(c.Request.Context(), playlistURL)
	if err != nil {
		logger.Error(err, "failed to parse playlist for seek")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse video playlist"})
		return
	}

	// find target segment and preload segments
	targetSegmentIndex, segmentStartTime := vac.findSegmentByTime(segments, request.Time)

	if targetSegmentIndex < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("time %.2f is out of range (video duration: %.2f)", request.Time, totalDuration),
		})
		return
	}

	// calculate segments to preload
	endIndex := targetSegmentIndex + request.PreloadCount
	if endIndex >= len(segments) {
		endIndex = len(segments)
	}

	// build file list for segments
	segmentFiles := make([]string, 0, endIndex-targetSegmentIndex)
	for i := targetSegmentIndex; i < endIndex; i++ {
		segmentFiles = append(segmentFiles, quality+"/"+segments[i].Filename)
	}

	// generate signed URLs for segments
	fullPaths := make([]string, len(segmentFiles))
	for i, file := range segmentFiles {
		fullPaths[i] = basePath + file
	}

	signedURLs, err := vac.storageProvider.GenerateSignedURLs(c.Request.Context(), fullPaths, &storage.CDNSignedURLOptions{
		ExpiresIn:    time.Hour * 2,
		CacheControl: "public, max-age=86400",
	})
	if err != nil {
		logger.Error(err, "failed to generate segment URLs for seek")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate segment URLs"})
		return
	}

	// map back to original file names
	fileURLs := make(map[string]string)
	for i, file := range segmentFiles {
		fullPath := fullPaths[i]
		url, exists := signedURLs[fullPath]
		if exists {
			fileURLs[file] = url
		}
	}

	// build response with seeking information
	response := gin.H{
		"movie_id":             movieID.String(),
		"target_time":          request.Time,
		"target_segment_index": targetSegmentIndex,
		"segment_start_time":   segmentStartTime,
		"total_duration":       totalDuration,
		"quality":              quality,
		"file_urls":            fileURLs,
		"segments":             segments[targetSegmentIndex:endIndex], // include segment metadata
		"expires_at":           time.Now().Add(time.Hour * 2).Format(time.RFC3339),
	}

	c.Header("Cache-Control", "private, max-age=60") // shorter cache for seek responses
	c.JSON(http.StatusOK, response)
}

// SegmentInfo represents a video segment with timing information
type SegmentInfo struct {
	Index     int     `json:"index"`
	Filename  string  `json:"filename"`
	Duration  float64 `json:"duration"`
	StartTime float64 `json:"start_time"`
}

// parsePlaylistForSeek parses an HLS playlist and returns segment information
func (vac *VideoAccessController) parsePlaylistForSeek(ctx context.Context, playlistURL string) ([]SegmentInfo, float64, error) {
	// create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(playlistURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch playlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("playlist request failed with status: %d", resp.StatusCode)
	}

	// read playlist content
	body := make([]byte, 0, 1024*1024) // 1MB buffer
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			body = append(body, buffer[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, 0, fmt.Errorf("failed to read playlist: %w", err)
		}
	}

	// parse playlist
	content := string(body)
	lines := strings.Split(content, "\n")

	var segments []SegmentInfo
	var currentDuration float64
	var totalDuration float64
	segmentIndex := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// parse segment duration
		if strings.HasPrefix(line, "#EXTINF:") {
			// extract duration from #EXTINF:duration,
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				durationPart := strings.Split(parts[1], ",")[0]
				if duration, err := parseFloat64(durationPart); err == nil {
					currentDuration = duration
				}
			}
		} else if line != "" && !strings.HasPrefix(line, "#") {
			// this is a segment filename
			segment := SegmentInfo{
				Index:     segmentIndex,
				Filename:  line,
				Duration:  currentDuration,
				StartTime: totalDuration,
			}
			segments = append(segments, segment)
			totalDuration += currentDuration
			segmentIndex++
		}
	}

	return segments, totalDuration, nil
}

// findSegmentByTime finds the segment that contains the given time
func (vac *VideoAccessController) findSegmentByTime(segments []SegmentInfo, targetTime float64) (int, float64) {
	for i, segment := range segments {
		if targetTime >= segment.StartTime && targetTime < segment.StartTime+segment.Duration {
			return i, segment.StartTime
		}
	}

	// if time is beyond the last segment, return the last segment
	if len(segments) > 0 {
		lastSegment := segments[len(segments)-1]
		if targetTime >= lastSegment.StartTime {
			return len(segments) - 1, lastSegment.StartTime
		}
	}

	return -1, 0
}

// helper function to parse float64 safely
func parseFloat64(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// handle common cases
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// use Go's built-in parser
	return strconv.ParseFloat(s, 64)
}
