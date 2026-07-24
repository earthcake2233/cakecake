package errcode

import (
	"testing"
)

func TestGetMsg_Success(t *testing.T) {
	if msg := GetMsg(CodeSuccess); msg != "ok" {
		t.Fatalf("GetMsg(0) = %q, want \"ok\"", msg)
	}
}

func TestGetMsg_KnownCodes(t *testing.T) {
	codes := []int{
		CodeSuccess,
		CodeParamError,
		CodeCoverFormat,
		CodeCoverSize,
		CodeDanmakuCooldown,
		CodeDanmakuSensitive,
		CodeUsernameExists,
		CodeMultipartParseError,
		CodeUploadMissingFile,
		CodeVideoProbeFailed,
		CodeVideoDurationExceeded,
		CodeVideoFileTooLarge,
		CodeTitleInvalid,
		CodeIntroTooLong,
		CodeInvalidColor,
		CodeAvatarFormat,
		CodeAvatarSize,
		CodeDeletionRevokeExpired,
		CodeAlreadyCoined,
		CodeCannotCoinSelf,
		CodeInsufficientCoins,
		CodeCommentSensitive,
		CodeVideoUploadDisabled,
		CodeTooManyRequests,
		CodeUnauthorized,
		CodeInvalidLogin,
		CodeForbidden,
		CodePasswordMismatch,
		CodeAccountClosed,
		CodeCommentsClosed,
		CodeDanmakuClosed,
		CodeUserBlocked,
		CodeAdminDisabled,
		CodeNotFound,
		CodeSearchUnavailable,
		CodeInternalError,
	}

	for _, code := range codes {
		msg := GetMsg(code)
		if msg == "" {
			t.Fatalf("GetMsg(%d) returned empty message", code)
		}
	}
}

func TestGetMsg_UnknownCode(t *testing.T) {
	msg := GetMsg(99999)
	expected := GetMsg(CodeParamError)
	if msg != expected {
		t.Fatalf("GetMsg(99999) = %q, want %q (CodeParamError message)", msg, expected)
	}
}
