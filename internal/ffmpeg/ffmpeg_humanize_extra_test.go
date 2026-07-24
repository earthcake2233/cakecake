package ffmpeg

import (
	"strings"
	"testing"
)

func TestHumanizeFailReason_KnownFailuresInFFmpegOutput(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"ffmpeg version 6.0\nInvalid data found when processing input",
			"无法识别该视频，文件可能已损坏或格式不受支持，请换文件后重试。"},
		{"ffmpeg version 6.0\nmoov atom not found",
			"视频文件不完整或已损坏，请重新导出为完整 MP4 后再上传。"},
		{"ffmpeg version 6.0\nUnsupported codec",
			"视频编码格式不受支持，请转为 H.264（AVC）后再上传。"},
		{"ffmpeg version 6.0\nunknown decoder",
			"不支持该视频的解码方式，请用常见编码重新导出后再上传。"},
		{"ffmpeg version 6.0\nerror opening input",
			"打开视频文件失败，请确认文件可读且路径有效。"},
		{"ffmpeg version 6.0\nNo such file or directory",
			"处理时找不到文件，请重新上传。"},
		{"ffmpeg version 6.0\nPermission denied",
			"没有权限读取或写入文件，请检查服务器配置。"},
		{"ffmpeg version 6.0\noperation not permitted",
			"操作被拒绝，请检查运行环境权限。"},
	}
	for _, tc := range tests {
		got := HumanizeFailReason(tc.raw)
		if got != tc.want {
			t.Errorf("HumanizeFailReason(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestHumanizeFailReason_BriefEnglish(t *testing.T) {
	msg := "timeout"
	got := HumanizeFailReason(msg)
	if got == "" || got == defaultTranscodeFail {
		t.Errorf("HumanizeFailReason(%q) = %q, want original", msg, got)
	}
}

func TestHumanizeFailReason_LongEnglish(t *testing.T) {
	msg := strings.Repeat("x", 500)
	got := HumanizeFailReason(msg)
	if got != defaultTranscodeFail {
		t.Errorf("HumanizeFailReason(long english) = %q, want default", got)
	}
}

func TestHumanizeFailReason_FFmpegBannerEdgeCases(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"ffmpeg version 6.0\nconfiguration: --enable-gpl --enable-nonfree", defaultTranscodeFail},
		{"built with --enable-gpl and --enable-version3 and --enable-static", defaultTranscodeFail},
		{"libavutil      58.  2.100 / 58.  2.100\nlibavcodec     60.  3.100 / 60.  3.100", defaultTranscodeFail},
	}
	for _, tc := range tests {
		got := HumanizeFailReason(tc.raw)
		if got != tc.want {
			t.Errorf("HumanizeFailReason(...) = %q, want %q", got, tc.want)
		}
	}
}

func TestHumanizeFailReason_ShortChinese(t *testing.T) {
	msg := "视频上传成功"
	got := HumanizeFailReason(msg)
	if got != msg {
		t.Errorf("HumanizeFailReason(%q) = %q, want %q", msg, got, msg)
	}
}

func TestHumanizeFailReason_LongChinese(t *testing.T) {
	msg := strings.Repeat("你好世界", 60)
	got := HumanizeFailReason(msg)
	if len([]rune(got)) > 220 {
		t.Errorf("HumanizeFailReason(long chinese) len=%d, want truncated", len([]rune(got)))
	}
}

func TestHumanizeFailReason_EmptyInput(t *testing.T) {
	if got := HumanizeFailReason(""); got != "" {
		t.Errorf("HumanizeFailReason empty = %q, want empty", got)
	}
	if got := HumanizeFailReason("  \n\t  "); got != "" {
		t.Errorf("HumanizeFailReason whitespace = %q, want empty", got)
	}
}

func TestHasHan(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"hello", false},
		{"你好", true},
		{"hello世界", true},
		{"", false},
		{"123!@#", false},
		{"中文混合abc", true},
	}
	for _, tc := range tests {
		got := hasHan(tc.s)
		if got != tc.want {
			t.Errorf("hasHan(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestLimitRunes(t *testing.T) {
	tests := []struct {
		s   string
		max int
	}{
		{"hello", 10},
		{"hello", 3},
		{"你好世界", 2},
		{"abc", 0},
		{"abc", -1},
		{"", 5},
	}
	for _, tc := range tests {
		got := limitRunes(tc.s, tc.max)
		if tc.max <= 0 {
			if got != "" {
				t.Errorf("limitRunes(%q, %d) = %q, want empty", tc.s, tc.max, got)
			}
			continue
		}
		if len([]rune(got)) > tc.max && !strings.HasSuffix(got, "…") {
			t.Errorf("limitRunes(%q, %d) = %q (len=%d), expected truncation", tc.s, tc.max, got, len([]rune(got)))
		}
	}
}

func TestLooksLikeFFmpegNoise(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"ffmpeg version 8.1", true},
		{"configuration: --enable-gpl --enable-version3", true},
		{"--enable-gpl and --enable-nonfree and --enable-static", true},
		{"libavutil 58 libavcodec 60", true},
		{"normal error message", false},
		{"", false},
		{"--enable-", false},
		{"configuration: value", false},
	}
	for _, tc := range tests {
		got := looksLikeFFmpegNoise(tc.s)
		if got != tc.want {
			t.Errorf("looksLikeFFmpegNoise(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestMatchKnownFailure(t *testing.T) {
	tests := []struct {
		lower string
		want  string
	}{
		{"invalid data found when processing input", "无法识别该视频，文件可能已损坏或格式不受支持，请换文件后重试。"},
		{"moov atom not found", "视频文件不完整或已损坏，请重新导出为完整 MP4 后再上传。"},
		{"unsupported codec", "视频编码格式不受支持，请转为 H.264（AVC）后再上传。"},
		{"unknown decoder", "不支持该视频的解码方式，请用常见编码重新导出后再上传。"},
		{"error while decoding", "解码视频时出错，文件可能损坏或编码异常，请换源文件后重试。"},
		{"error decoding", "解码视频时出错，文件可能损坏或编码异常，请换源文件后重试。"},
		{"error opening input", "打开视频文件失败，请确认文件可读且路径有效。"},
		{"error opening output", "打开视频文件失败，请确认文件可读且路径有效。"},
		{"no such file or directory", "处理时找不到文件，请重新上传。"},
		{"permission denied", "没有权限读取或写入文件，请检查服务器配置。"},
		{"operation not permitted", "操作被拒绝，请检查运行环境权限。"},
		{"accessdenied", "存储服务拒绝访问，请检查云存储权限配置。"},
		{"access denied", "存储服务拒绝访问，请检查云存储权限配置。"},
		{"nosuchbucket", "存储桶不存在，请检查云存储配置。"},
		{"no such bucket", "存储桶不存在，请检查云存储配置。"},
		{"signaturedoesnotmatch", "存储服务签名校验失败，请检查密钥配置。"},
		{"context deadline exceeded", "上传或处理超时，请检查网络后重试。"},
		{"i/o timeout", "上传或处理超时，请检查网络后重试。"},
		{"connection refused", "无法连接依赖服务，请稍后重试或联系管理员。"},
		{"unknown error", ""},
		{"", ""},
	}
	for _, tc := range tests {
		got := matchKnownFailure(tc.lower)
		if got != tc.want {
			t.Errorf("matchKnownFailure(%q) = %q, want %q", tc.lower, got, tc.want)
		}
	}
}
