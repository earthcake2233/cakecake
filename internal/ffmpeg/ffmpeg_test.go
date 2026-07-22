package ffmpeg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit_Defaults(t *testing.T) {
	Init("", "")
	require.Equal(t, "ffprobe", FFprobeExe())
	require.Equal(t, "ffmpeg", FFmpegExe())
}

func TestInit_CustomPaths(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()

	Init("/usr/local/bin/ffprobe", "/usr/local/bin/ffmpeg")
	require.Equal(t, "/usr/local/bin/ffprobe", FFprobeExe())
	require.Equal(t, "/usr/local/bin/ffmpeg", FFmpegExe())
}

func TestInit_TrimWhitespace(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()

	Init("  /custom/ffprobe  ", "  /custom/ffmpeg  ")
	require.Equal(t, "/custom/ffprobe", FFprobeExe())
	require.Equal(t, "/custom/ffmpeg", FFmpegExe())
}

func TestInit_OnlyFfprobe(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()

	Init("/custom/ffprobe", "")
	require.Equal(t, "/custom/ffprobe", FFprobeExe())
	require.Equal(t, "ffmpeg", FFmpegExe())
}

func TestInit_OnlyFfmpeg(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()

	Init("", "/custom/ffmpeg")
	require.Equal(t, "ffprobe", FFprobeExe())
	require.Equal(t, "/custom/ffmpeg", FFmpegExe())
}

func TestIsPermanentTransienceFailure_InvalidData(t *testing.T) {
	require.True(t, IsPermanentTranscodeFailure("Invalid data found when processing input"))
}

func TestIsPermanentTransienceFailure_UnsupportedCodec(t *testing.T) {
	require.True(t, IsPermanentTranscodeFailure("Unsupported codec"))
}

func TestIsPermanentTransienceFailure_NoSuchFile(t *testing.T) {
	require.True(t, IsPermanentTranscodeFailure("No such file or directory"))
}

func TestIsPermanentTransienceFailure_PermissionDenied(t *testing.T) {
	require.True(t, IsPermanentTranscodeFailure("Permission denied"))
}

func TestIsPermanentTransienceFailure_CaseSensitive(t *testing.T) {
	require.True(t, IsPermanentTranscodeFailure("No such file or directory"))
	require.True(t, IsPermanentTranscodeFailure("Invalid data found when processing input"))
	require.False(t, IsPermanentTranscodeFailure("no such file or directory"))
}

func TestIsPermanentTransienceFailure_OtherError(t *testing.T) {
	require.False(t, IsPermanentTranscodeFailure("Connection reset by peer"))
	require.False(t, IsPermanentTranscodeFailure(""))
	require.False(t, IsPermanentTranscodeFailure("some random error"))
}

func TestIsPermanentTransienceFailure_SubstringMatch(t *testing.T) {
	require.True(t, IsPermanentTranscodeFailure("prefix No such file or directory suffix"))
}
