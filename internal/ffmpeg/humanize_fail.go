package ffmpeg

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const defaultTranscodeFail = "视频转码失败，请确认文件完整、格式常见（如 MP4），或转为 H.264 视频 + AAC 音频后重新上传。"

// HumanizeFailReason turns FFmpeg stderr, OSS errors, or other technical logs
// into a short user-facing Chinese message. Empty input returns empty.
func HumanizeFailReason(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)

	// Already operator-facing Chinese / short note from our code
	if !looksLikeFFmpegNoise(s) && hasHan(s) && utf8.RuneCountInString(s) <= 200 {
		return limitRunes(s, 220)
	}
	if !looksLikeFFmpegNoise(s) && utf8.RuneCountInString(s) <= 80 && !strings.Contains(s, "--enable-") {
		return limitRunes(s, 120)
	}

	if msg := matchKnownFailure(lower); msg != "" {
		return msg
	}
	if looksLikeFFmpegNoise(s) || strings.Count(s, "\n") > 8 || utf8.RuneCountInString(s) > 400 {
		return defaultTranscodeFail
	}
	return defaultTranscodeFail
}

func looksLikeFFmpegNoise(s string) bool {
	ls := strings.ToLower(s)
	if strings.Contains(ls, "ffmpeg version") {
		return true
	}
	if strings.Contains(ls, "configuration:") && strings.Contains(ls, "--enable") {
		return true
	}
	if strings.Count(s, "--enable-") >= 2 {
		return true
	}
	if strings.Contains(ls, "libavutil") && strings.Contains(ls, "libavcodec") {
		return true
	}
	return false
}

func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func limitRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}

func matchKnownFailure(lower string) string {
	// FFmpeg / ffprobe media
	if strings.Contains(lower, "invalid data found when processing input") {
		return "无法识别该视频，文件可能已损坏或格式不受支持，请换文件后重试。"
	}
	if strings.Contains(lower, "moov atom not found") {
		return "视频文件不完整或已损坏，请重新导出为完整 MP4 后再上传。"
	}
	if strings.Contains(lower, "unsupported codec") {
		return "视频编码格式不受支持，请转为 H.264（AVC）后再上传。"
	}
	if strings.Contains(lower, "unknown decoder") {
		return "不支持该视频的解码方式，请用常见编码重新导出后再上传。"
	}
	if strings.Contains(lower, "error while decoding") || strings.Contains(lower, "error decoding") {
		return "解码视频时出错，文件可能损坏或编码异常，请换源文件后重试。"
	}
	if strings.Contains(lower, "error opening input") || strings.Contains(lower, "error opening output") {
		return "打开视频文件失败，请确认文件可读且路径有效。"
	}
	// Filesystem
	if strings.Contains(lower, "no such file or directory") {
		return "处理时找不到文件，请重新上传。"
	}
	if strings.Contains(lower, "permission denied") {
		return "没有权限读取或写入文件，请检查服务器配置。"
	}
	if strings.Contains(lower, "operation not permitted") {
		return "操作被拒绝，请检查运行环境权限。"
	}
	// Network / OSS (English fragments)
	if strings.Contains(lower, "accessdenied") || strings.Contains(lower, "access denied") {
		return "存储服务拒绝访问，请检查云存储权限配置。"
	}
	if strings.Contains(lower, "nosuchbucket") || strings.Contains(lower, "no such bucket") {
		return "存储桶不存在，请检查云存储配置。"
	}
	if strings.Contains(lower, "signaturedoesnotmatch") {
		return "存储服务签名校验失败，请检查密钥配置。"
	}
	if strings.Contains(lower, "context deadline exceeded") || strings.Contains(lower, "i/o timeout") {
		return "上传或处理超时，请检查网络后重试。"
	}
	if strings.Contains(lower, "connection refused") {
		return "无法连接依赖服务，请稍后重试或联系管理员。"
	}
	return ""
}
