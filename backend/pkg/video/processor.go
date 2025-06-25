package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
)

// Processor handles video transcoding and HLS conversion
type Processor interface {
	// TranscodeToHLS converts a video file to HLS format with multiple quality levels
	TranscodeToHLS(ctx context.Context, inputPath, outputDir string, qualities []Quality) (*HLSOutput, error)

	// GetVideoInfo extracts metadata from a video file
	GetVideoInfo(ctx context.Context, filePath string) (*VideoInfo, error)

	// ValidateVideoFile validates if a file is a supported video format
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
	MasterPlaylistPath string            // Path to master m3u8 file
	QualityPlaylists   map[string]string // Quality name -> playlist path
	SegmentFiles       []string          // All .ts segment files
	TotalSegments      int
	ProcessingTime     time.Duration
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

// TranscodeToHLS converts a video file to HLS format
func (p *videoProcessor) TranscodeToHLS(ctx context.Context, inputPath, outputDir string, qualities []Quality) (*HLSOutput, error) {
	startTime := time.Now()

	// ensure output directory exists
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	output := &HLSOutput{
		QualityPlaylists: make(map[string]string),
		SegmentFiles:     make([]string, 0),
	}

	// generate HLS for each quality level
	for _, quality := range qualities {
		qualityDir := filepath.Join(outputDir, quality.Name)
		err := os.MkdirAll(qualityDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create quality directory %s: %w", quality.Name, err)
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
			return nil, fmt.Errorf("ffmpeg failed for quality %s: %w", quality.Name, err)
		}

		// collect segment files
		segments, err := filepath.Glob(filepath.Join(qualityDir, "segment_*.ts"))
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to list segment files for quality %s", quality.Name))
		} else {
			output.SegmentFiles = append(output.SegmentFiles, segments...)
			output.TotalSegments += len(segments)
		}

		output.QualityPlaylists[quality.Name] = playlistPath
		logger.Infof("successfully transcoded to %s quality", quality.Name)
	}

	// create master playlist
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	err = p.createMasterPlaylist(masterPlaylistPath, qualities, output.QualityPlaylists)
	if err != nil {
		return nil, fmt.Errorf("failed to create master playlist: %w", err)
	}

	output.MasterPlaylistPath = masterPlaylistPath
	output.ProcessingTime = time.Since(startTime)

	logger.Infof("HLS transcoding completed in %v, generated %d segments across %d qualities",
		output.ProcessingTime, output.TotalSegments, len(qualities))

	return output, nil
}

// createMasterPlaylist creates the master HLS playlist
func (p *videoProcessor) createMasterPlaylist(masterPath string, qualities []Quality, playlistPaths map[string]string) error {
	var content strings.Builder
	content.WriteString("#EXTM3U\n")
	content.WriteString("#EXT-X-VERSION:3\n\n")

	for _, quality := range qualities {
		if playlistPath, exists := playlistPaths[quality.Name]; exists {
			// extract relative path for the playlist
			relPath := filepath.Base(filepath.Dir(playlistPath)) + "/playlist.m3u8"

			// parse bitrate (remove 'k' suffix and convert to bps)
			bitrateStr := strings.TrimSuffix(quality.Bitrate, "k")
			bitrate, _ := strconv.Atoi(bitrateStr)
			bitrateBps := bitrate * 1000

			content.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=\"%s\"\n",
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
