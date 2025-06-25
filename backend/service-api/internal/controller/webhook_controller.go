package controller

import (
	"net/http"
	"watch-party/pkg/events"
	"watch-party/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebhookController handles webhook events
type WebhookController struct {
	uploadHandler events.Handler
}

// NewWebhookController creates a new webhook controller
func NewWebhookController(uploadHandler events.Handler) *WebhookController {
	return &WebhookController{
		uploadHandler: uploadHandler,
	}
}

// HandleUploadComplete handles file upload completion webhook
func (wc *WebhookController) HandleUploadComplete(c *gin.Context) {
	var event events.UploadEvent
	err := c.ShouldBindJSON(&event)
	if err != nil {
		logger.Error(err, "failed to bind upload completion event")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event data"})
		return
	}

	// validate required fields
	if event.MovieID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "movie_id is required"})
		return
	}

	if event.FilePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_path is required"})
		return
	}

	// process the upload completion event
	err = wc.uploadHandler.HandleUploadComplete(c.Request.Context(), &event)
	if err != nil {
		logger.Error(err, "failed to handle upload completion")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process upload completion"})
		return
	}

	logger.Infof("upload completion processed successfully for movie %s", event.MovieID)
	c.JSON(http.StatusOK, gin.H{
		"message": "upload completion processed successfully",
	})
}

// extractMovieIDFromPath extracts movie ID from file path
// Assumes format: uploads/{movieID}_{timestamp}.ext
func extractMovieIDFromPath(path string) string {
	// simple extraction - in production you might want more robust parsing
	if len(path) < 20 { // UUID is 36 chars, so path should be longer
		return ""
	}

	// find the start of the UUID (after "uploads/")
	startIdx := 0
	if path[:8] == "uploads/" {
		startIdx = 8
	}

	// find the end of the UUID (before the first underscore after the UUID)
	endIdx := startIdx + 36 // UUID length
	if endIdx <= len(path) && startIdx < len(path) {
		return path[startIdx:endIdx]
	}

	return ""
}
