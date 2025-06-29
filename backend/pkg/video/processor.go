package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
)

// Processor handles video transcoding and HLS conversion
type Processor interface {
	TranscodeToHLS(ctx context.Context, inputPath, outputDir, storagePrefix string, qualities []Quality) (*HLSOutput, error)
	GetVideoInfo(ctx context.Context, filePath string) (*VideoInfo, error)
	ValidateVideoFile(ctx context.Context, filePath string) error
}

// Quality represents a video quality level for HLS transcoding
type Quality struct {
	Name       string // e.g., "360p", "720p", "1080p"
	Width      int
	Height     int
	Bitrate    string // e.g., "1000k", "2500k", "5000k"
	SegmentDur int    // segment duration in seconds
}

// VideoInfo contains metadata about a video file
type VideoInfo struct {
	Duration   float64 // duration in seconds
	Width      int
	Height     int
	Bitrate    int64
	FrameRate  float64
	VideoCodec string
	AudioCodec string
	FileSize   int64
}

// HLSOutput contains information about the generated HLS files
type HLSOutput struct {
	MasterPlaylistURL   string            // URL to master m3u8 file in storage
	QualityPlaylistURLs map[string]string // Quality name -> playlist URL in storage
	SegmentURLs         []string          // All .ts segment URLs in storage
	TotalSegments       int
	ProcessingTime      time.Duration
}

// QualityResult holds the result of processing a single quality level
type QualityResult struct {
	Quality      Quality
	PlaylistURL  string
	SegmentURLs  []string
	SegmentCount int
	Error        error
}

// videoProcessor implements the Processor interface using FFmpeg
type videoProcessor struct {
	storageProvider storage.Provider
	tempDir         string
	ffmpegPath      string
	ffprobePath     string
}

// NewProcessor creates a new video processor
func NewProcessor(storageProvider storage.Provider, tempDir string) Processor {
	return &videoProcessor{
		storageProvider: storageProvider,
		tempDir:         tempDir,
		ffmpegPath:      "ffmpeg",  // assumes ffmpeg is in PATH
		ffprobePath:     "ffprobe", // assumes ffprobe is in PATH
	}
}

// Default quality levels for HLS transcoding
var DefaultQualities = []Quality{
	{Name: "360p", Width: 640, Height: 360, Bitrate: "1000k", SegmentDur: 6},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: "2500k", SegmentDur: 6},
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "5000k", SegmentDur: 6},
}

// TranscodeToHLS converts a video file to HLS format and uploads to storage
func (p *videoProcessor) TranscodeToHLS(ctx context.Context, inputPath, outputDir, storagePrefix string, qualities []Quality) (*HLSOutput, error) {
	startTime := time.Now()

	// ensure output directory exists
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// defer cleanup of local files
	defer func() {
		if err := os.RemoveAll(outputDir); err != nil {
			logger.Error(err, "failed to cleanup temporary directory")
		}
	}()

	// channel to collect results from goroutines
	resultsChan := make(chan QualityResult, len(qualities))
	var wg sync.WaitGroup

	// process each quality level concurrently
	for _, quality := range qualities {
		wg.Add(1)
		go func(q Quality) {
			defer wg.Done()
			result := p.processQuality(ctx, inputPath, outputDir, storagePrefix, q)
			resultsChan <- result
		}(quality)
	}

	// wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// collect results
	output := &HLSOutput{
		QualityPlaylistURLs: make(map[string]string),
		SegmentURLs:         make([]string, 0),
	}

	qualityPlaylistPaths := make(map[string]string) // for master playlist creation
	var processingErrors []error

	for result := range resultsChan {
		if result.Error != nil {
			processingErrors = append(processingErrors,
				fmt.Errorf("quality %s failed: %w", result.Quality.Name, result.Error))
			continue
		}

		output.QualityPlaylistURLs[result.Quality.Name] = result.PlaylistURL
		output.SegmentURLs = append(output.SegmentURLs, result.SegmentURLs...)
		output.TotalSegments += result.SegmentCount

		// store for master playlist creation
		qualityPlaylistPaths[result.Quality.Name] = result.Quality.Name + "/playlist.m3u8"

		logger.Infof("successfully processed and uploaded %s quality (%d segments)",
			result.Quality.Name, result.SegmentCount)
	}

	// check if any processing failed
	if len(processingErrors) > 0 {
		logger.Error(nil, fmt.Sprintf("some qualities failed to process: %v", processingErrors))
		// continue with successful qualities if at least one succeeded
		if len(output.QualityPlaylistURLs) == 0 {
			return nil, fmt.Errorf("all quality levels failed to process")
		}
	}

	// create and upload master playlist
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	err = p.createMasterPlaylist(masterPlaylistPath, qualities, qualityPlaylistPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to create master playlist: %w", err)
	}

	// upload master playlist
	masterStoragePath := storagePrefix + "/master.m3u8"
	err = p.storageProvider.UploadFromPath(ctx, masterPlaylistPath, masterStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload master playlist: %w", err)
	}

	// get master playlist URL
	masterURL, err := p.storageProvider.GetPublicURL(ctx, masterStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get master playlist URL: %w", err)
	}

	output.MasterPlaylistURL = masterURL
	output.ProcessingTime = time.Since(startTime)

	logger.Infof("HLS transcoding completed in %v, generated %d segments across %d qualities",
		output.ProcessingTime, output.TotalSegments, len(output.QualityPlaylistURLs))

	return output, nil
}

// processQuality handles transcoding and uploading for a single quality level
func (p *videoProcessor) processQuality(ctx context.Context, inputPath, outputDir, storagePrefix string, quality Quality) QualityResult {
	result := QualityResult{Quality: quality}

	qualityDir := filepath.Join(outputDir, quality.Name)
	err := os.MkdirAll(qualityDir, 0755)
	if err != nil {
		result.Error = fmt.Errorf("failed to create quality directory %s: %w", quality.Name, err)
		return result
	}

	playlistPath := filepath.Join(qualityDir, "playlist.m3u8")
	segmentPattern := filepath.Join(qualityDir, "segment_%03d.ts")

	// build ffmpeg command for this quality
	cmd := exec.CommandContext(ctx,
		p.ffmpegPath,
		"-i", inputPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-b:v", quality.Bitrate,
		"-s", fmt.Sprintf("%dx%d", quality.Width, quality.Height),
		"-hls_time", strconv.Itoa(quality.SegmentDur),
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		"-f", "hls",
		playlistPath,
	)

	logger.Infof("transcoding to %s: %s", quality.Name, cmd.String())

	// run ffmpeg command
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err, fmt.Sprintf("ffmpeg command failed for quality %s: %s", quality.Name, string(cmdOutput)))
		result.Error = fmt.Errorf("ffmpeg failed for quality %s: %w", quality.Name, err)
		return result
	}

	// collect segment files
	segments, err := filepath.Glob(filepath.Join(qualityDir, "segment_*.ts"))
	if err != nil {
		result.Error = fmt.Errorf("failed to list segment files for quality %s: %w", quality.Name, err)
		return result
	}

	// upload segments concurrently
	segmentUploadChan := make(chan string, len(segments))
	segmentErrorChan := make(chan error, len(segments))
	var segmentWg sync.WaitGroup

	for _, segmentPath := range segments {
		segmentWg.Add(1)
		go func(localPath string) {
			defer segmentWg.Done()

			filename := filepath.Base(localPath)
			storagePath := fmt.Sprintf("%s/%s/%s", storagePrefix, quality.Name, filename)

			err := p.storageProvider.UploadFromPath(ctx, localPath, storagePath)
			if err != nil {
				segmentErrorChan <- fmt.Errorf("failed to upload segment %s: %w", filename, err)
				return
			}

			// get public URL for the segment
			segmentURL, err := p.storageProvider.GetPublicURL(ctx, storagePath)
			if err != nil {
				segmentErrorChan <- fmt.Errorf("failed to get URL for segment %s: %w", filename, err)
				return
			}

			segmentUploadChan <- segmentURL
		}(segmentPath)
	}

	// wait for all segment uploads to complete
	go func() {
		segmentWg.Wait()
		close(segmentUploadChan)
		close(segmentErrorChan)
	}()

	// collect segment URLs and check for errors
	segmentURLs := make([]string, 0, len(segments))
	for segmentURL := range segmentUploadChan {
		segmentURLs = append(segmentURLs, segmentURL)
	}

	// check for upload errors
	for uploadErr := range segmentErrorChan {
		if uploadErr != nil {
			result.Error = uploadErr
			return result
		}
	}

	// upload playlist file
	playlistStoragePath := fmt.Sprintf("%s/%s/playlist.m3u8", storagePrefix, quality.Name)
	err = p.storageProvider.UploadFromPath(ctx, playlistPath, playlistStoragePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to upload playlist for quality %s: %w", quality.Name, err)
		return result
	}

	// get playlist URL
	playlistURL, err := p.storageProvider.GetPublicURL(ctx, playlistStoragePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to get playlist URL for quality %s: %w", quality.Name, err)
		return result
	}

	result.PlaylistURL = playlistURL
	result.SegmentURLs = segmentURLs
	result.SegmentCount = len(segments)

	return result
}

// createMasterPlaylist creates the master HLS playlist
func (p *videoProcessor) createMasterPlaylist(masterPath string, qualities []Quality, playlistPaths map[string]string) error {
	var content strings.Builder
	content.WriteString("#EXTM3U\n")
	content.WriteString("#EXT-X-VERSION:3\n\n")

	for _, quality := range qualities {
		if relPath, exists := playlistPaths[quality.Name]; exists {
			// parse bitrate (remove 'k' suffix and convert to bps)
			bitrateStr := strings.TrimSuffix(quality.Bitrate, "k")
			bitrate, _ := strconv.Atoi(bitrateStr)
			bitrateBps := bitrate * 1000

			// include proper codec information for audio support
			// avc1.42E01E = H.264 Baseline Profile Level 3.0
			// mp4a.40.2 = AAC-LC (Low Complexity)
			content.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"avc1.42E01E,mp4a.40.2\",NAME=\"%s\"\n",
				bitrateBps, quality.Width, quality.Height, quality.Name))
			content.WriteString(fmt.Sprintf("%s\n\n", relPath))
		}
	}

	return os.WriteFile(masterPath, []byte(content.String()), 0644)
}

// GetVideoInfo extracts metadata from a video file using ffprobe
func (p *videoProcessor) GetVideoInfo(ctx context.Context, filePath string) (*VideoInfo, error) {
	cmd := exec.CommandContext(ctx,
		p.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// parse ffprobe output (simplified - you might want to use a JSON parser)
	info := &VideoInfo{}

	// this is a simplified parser - in production you'd want proper JSON parsing
	outputStr := string(output)
	if strings.Contains(outputStr, `"codec_type": "video"`) {
		// extract basic info - this is very simplified
		info.Duration = 0      // would parse from "duration" field
		info.Width = 1920      // would parse from "width" field
		info.Height = 1080     // would parse from "height" field
		info.Bitrate = 5000000 // would parse from "bit_rate" field
	}

	return info, nil
}

// ValidateVideoFile validates if a file is a supported video format
func (p *videoProcessor) ValidateVideoFile(ctx context.Context, filePath string) error {
	cmd := exec.CommandContext(ctx,
		p.ffprobePath,
		"-v", "quiet",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to validate video file: %w", err)
	}

	if strings.TrimSpace(string(output)) != "video" {
		return fmt.Errorf("file does not contain a valid video stream")
	}

	return nil
}
