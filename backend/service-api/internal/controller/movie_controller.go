package controller

import (
	"net/http"
	"strconv"
	"strings"
	"watch-party/pkg/logger"
	"watch-party/pkg/model"
	movieService "watch-party/service-api/internal/service/movie"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MovieController handles movie-related HTTP requests
type MovieController struct {
	movieService movieService.Service
}

// NewMovieController creates a new movie controller
func NewMovieController(movieService movieService.Service) *MovieController {
	return &MovieController{
		movieService: movieService,
	}
}

// UploadMovie handles movie upload - ADMIN ONLY
func (mc *MovieController) UploadMovie(c *gin.Context) {
	// get uploader ID from context (set by auth middleware)
	uploaderID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, ok := uploaderID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
		return
	}

	// parse form data
	var req model.UploadMovieRequest
	err := c.ShouldBind(&req)
	if err != nil {
		logger.Error(err, "failed to bind upload movie request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request data"})
		return
	}

	// get uploaded file
	file, err := c.FormFile("video")
	if err != nil {
		logger.Error(err, "failed to get uploaded file")
		c.JSON(http.StatusBadRequest, gin.H{"error": "video file is required"})
		return
	}

	// upload movie
	movie, err := mc.movieService.UploadMovie(c.Request.Context(), &req, file, userID)
	if err != nil {
		logger.Error(err, "failed to upload movie")

		// TODO: error dict
		if err.Error() == "unsupported video format" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported video format"})
			return
		}
		if strings.Contains(err.Error(), "file size too large") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload movie"})
		return
	}

	logger.Infof("movie uploaded successfully: %s", movie.Title)
	c.JSON(http.StatusCreated, gin.H{
		"message": "movie uploaded successfully",
		"movie":   movie,
	})
}

// GetMovies handles listing all movies - ADMIN ONLY
func (mc *MovieController) GetMovies(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	response, err := mc.movieService.GetMovies(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error(err, "failed to get movies list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve movies"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMovie handles getting a specific movie - ADMIN ONLY
func (mc *MovieController) GetMovie(c *gin.Context) {
	// parse movie ID
	idStr := c.Param("id")
	movieID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	movie, err := mc.movieService.GetMovie(c.Request.Context(), movieID)
	if err != nil {
		if err.Error() == "movie not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
		logger.Error(err, "failed to get movie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve movie"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"movie": movie})
}

// UpdateMovie handles updating movie metadata - ADMIN ONLY
func (mc *MovieController) UpdateMovie(c *gin.Context) {
	// parse movie ID
	idStr := c.Param("id")
	movieID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	// parse request body
	var req model.UploadMovieRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		logger.Error(err, "failed to bind update movie request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request data"})
		return
	}

	movie, err := mc.movieService.UpdateMovie(c.Request.Context(), movieID, &req)
	if err != nil {
		if err.Error() == "movie not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
		logger.Error(err, "failed to update movie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update movie"})
		return
	}

	logger.Infof("movie updated successfully: %s", movie.Title)
	c.JSON(http.StatusOK, gin.H{
		"message": "movie updated successfully",
		"movie":   movie,
	})
}

// DeleteMovie handles deleting a movie - ADMIN ONLY
func (mc *MovieController) DeleteMovie(c *gin.Context) {
	// parse movie ID
	idStr := c.Param("id")
	movieID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	err = mc.movieService.DeleteMovie(c.Request.Context(), movieID)
	if err != nil {
		if err.Error() == "movie not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
		logger.Error(err, "failed to delete movie")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete movie"})
		return
	}

	logger.Infof("movie deleted successfully: %s", movieID)
	c.JSON(http.StatusOK, gin.H{"message": "movie deleted successfully"})
}

// GetMovieStreamURL handles getting a stream URL for a movie - ADMIN ONLY
func (mc *MovieController) GetMovieStreamURL(c *gin.Context) {
	// parse movie ID
	idStr := c.Param("id")
	movieID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
		return
	}

	streamURL, err := mc.movieService.GetMovieStreamURL(c.Request.Context(), movieID)
	if err != nil {
		if err.Error() == "movie not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
		logger.Error(err, "failed to get movie stream URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stream URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stream_url": streamURL,
	})
}

// GetMyMovies handles getting movies uploaded by the current user - ADMIN ONLY
func (mc *MovieController) GetMyMovies(c *gin.Context) {
	// Get user ID from context
	uploaderID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, ok := uploaderID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	response, err := mc.movieService.GetMoviesByUploader(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		logger.Error(err, "failed to get user's movies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve movies"})
		return
	}

	c.JSON(http.StatusOK, response)
}
