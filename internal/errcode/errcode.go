package errcode

const (
	CodeSuccess                 = 0
	CodeParamError              = 40001
	CodeCoverFormat             = 40002
	CodeCoverSize               = 40003
	CodeDanmakuCooldown         = 40004
	CodeDanmakuSensitive        = 40005
	CodeUsernameExists          = 40006
	CodeMultipartParseError     = 40007
	CodeUploadMissingFile       = 40008
	CodeVideoProbeFailed        = 40009
	CodeVideoDurationExceeded   = 40010
	CodeVideoFileTooLarge       = 40011
	CodeTitleInvalid            = 40012
	CodeIntroTooLong            = 40013
	CodeInvalidColor            = 40014
	CodeAvatarFormat            = 40015
	CodeAvatarSize              = 40016
	CodeDeletionRevokeExpired = 40017
	CodeAlreadyCoined         = 40018
	CodeCannotCoinSelf        = 40019
	CodeInsufficientCoins     = 40020
	CodeCommentSensitive        = 40021
	CodeVideoUploadDisabled     = 40022
	CodeTooManyRequests         = 42900
	CodeUnauthorized            = 40100
	CodeInvalidLogin            = 40101
	CodeForbidden               = 40300
	CodePasswordMismatch        = 40301
	CodeAccountClosed           = 40302
	CodeCommentsClosed          = 40303
	CodeDanmakuClosed           = 40304
	CodeUserBlocked             = 40305
	CodeAdminDisabled           = 40306
	CodeNotFound                = 40400
	CodeSearchUnavailable       = 50301
	CodeInternalError           = 50000
)

var messages = map[int]string{
	CodeSuccess:            "ok",
	CodeParamError:         "参数错误",
	CodeCoverFormat:        "封面格式不支持，仅支持 JPEG/PNG/GIF/BMP/WEBP",
	CodeCoverSize:               "封面大小超过 10MB，请压缩后重新上传",
	CodeMultipartParseError:     "multipart 请求解析失败，请检查网络或稍后重试",
	CodeUploadMissingFile:       "未收到视频文件，请重新选择文件后再提交",
	CodeVideoProbeFailed:        "无法解析视频：请确认文件为有效视频；服务器 PATH 中需有 ffprobe，或在环境变量 FFPROBE_PATH 中填写其绝对路径",
	CodeVideoDurationExceeded:   "视频时长超过 30 分钟上限",
	CodeVideoFileTooLarge:       "视频文件超过 500 MB 上限",
	CodeTitleInvalid:            "标题须为 1–80 个字",
	CodeIntroTooLong:            "简介不能超过 2000 个字",
	CodeInvalidColor:            "弹幕颜色格式无效，请输入有效的十六进制色号（如 #FF0000）",
	CodeAvatarFormat:            "头像格式不支持，仅支持 JPEG/PNG/GIF/BMP/WEBP",
	CodeAvatarSize:              "头像大小超过 5MB，请压缩后重新上传",
	CodeDeletionRevokeExpired: "已超过撤销期限或账号已注销",
	CodeAlreadyCoined:         "已为该视频投过硬币",
	CodeCannotCoinSelf:      "不能给自己的视频投币",
	CodeInsufficientCoins:   "硬币不足",
	CodeCommentSensitive:    "评论内容包含违规信息",
	CodeVideoUploadDisabled: "云端视频上传已关闭，可先保存稿件信息；视频文件将由管理员线下处理",
	CodeTooManyRequests:    "请求过于频繁，请稍后重试",
	CodeDanmakuCooldown:    "发送过于频繁，请稍后再试",
	CodeDanmakuSensitive:   "弹幕内容包含违规信息",
	CodeUsernameExists:     "用户名已存在",
	CodeUnauthorized:       "未登录或 Token 已过期",
	CodeInvalidLogin:       "用户名或密码错误",
	CodeForbidden:          "无权限执行此操作",
	CodeCommentsClosed:     "UP主已关闭评论区",
	CodeDanmakuClosed:      "UP主已关闭弹幕",
	CodeUserBlocked:        "无法与对方互动，对方可能已将你加入黑名单",
	CodeAdminDisabled:    "管理员账号已禁用",
	CodePasswordMismatch:   "原密码错误",
	CodeAccountClosed:      "账号已注销",
	CodeNotFound:           "资源不存在",
	CodeSearchUnavailable:  "搜索服务暂不可用，请检查 Elasticsearch 配置",
	CodeInternalError:      "服务器内部错误",
}

// GetMsg returns the human-readable message for a business code.
func GetMsg(code int) string {
	if m, ok := messages[code]; ok {
		return m
	}
	return messages[CodeParamError]
}
