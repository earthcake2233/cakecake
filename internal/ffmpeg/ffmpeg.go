package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var ffprobeExe = "ffprobe"
var ffmpegExe = "ffmpeg"

// Init sets ffprobe/ffmpeg executable names or absolute paths (from FFPROBE_PATH / FFMPEG_PATH).
func Init(ffprobePath, ffmpegPath string) {
	if strings.TrimSpace(ffprobePath) != "" {
		ffprobeExe = strings.TrimSpace(ffprobePath)
	}
	if strings.TrimSpace(ffmpegPath) != "" {
		ffmpegExe = strings.TrimSpace(ffmpegPath)
	}
}

// FFprobeExe returns the configured ffprobe command (for logs).
func FFprobeExe() string { return ffprobeExe }

// FFmpegExe returns the configured ffmpeg command (for logs).
func FFmpegExe() string { return ffmpegExe }

// CheckFFprobe runs ffprobe -version once at startup to surface PATH / installation issues.
func CheckFFprobe() error {
	return exec.Command(ffprobeExe, "-version").Run()
}

// ProbeDurationSeconds returns media duration using ffprobe. The context
// bounds the probe so a hung ffprobe cannot block an upload request forever.
func ProbeDurationSeconds(ctx context.Context, path string) (float64, error) {
	cmd := exec.CommandContext(ctx, ffprobeExe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe: %w: %s", err, out.String())
	}
	s := strings.TrimSpace(out.String())
	if s == "" {
		return 0, fmt.Errorf("empty ffprobe output")
	}
	return strconv.ParseFloat(s, 64)
}

// TranscodeToH264MP4 converts input to H.264 + AAC MP4. The context bounds
// the ffmpeg process: on timeout/cancel the process is killed so a hung
// encoder cannot block the worker forever.
func TranscodeToH264MP4(ctx context.Context, inputPath, outputPath string) (stderr string, err error) {
	cmd := exec.CommandContext(ctx, ffmpegExe,
		"-y",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-movflags", "+faststart",
		outputPath,
	)
	var eb bytes.Buffer
	cmd.Stderr = &eb
	if err := cmd.Run(); err != nil {
		return eb.String(), fmt.Errorf("ffmpeg: %w", err)
	}
	return eb.String(), nil
}

// ScreenshotJPEG captures a frame at t seconds as JPEG. The context bounds
// the ffmpeg process the same way as TranscodeToH264MP4.
func ScreenshotJPEG(ctx context.Context, inputPath, outputPath string, atSeconds float64) (stderr string, err error) {
	cmd := exec.CommandContext(ctx, ffmpegExe,
		"-y",
		"-ss", fmt.Sprintf("%.3f", atSeconds),
		"-i", inputPath,
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	)
	var eb bytes.Buffer
	cmd.Stderr = &eb
	if err := cmd.Run(); err != nil {
		return eb.String(), fmt.Errorf("ffmpeg screenshot: %w", err)
	}
	return eb.String(), nil
}

// IsPermanentTranscodeFailure classifies FFmpeg errors (Skill S-004).
func IsPermanentTranscodeFailure(stderr string) bool {
	patterns := []string{
		"Invalid data found when processing input",
		"Unsupported codec",
		"No such file or directory",
		"Permission denied",
	}
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			return true
		}
	}
	return false
}
