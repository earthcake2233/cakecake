package coverval

import (
	"mime/multipart"
	"path/filepath"
	"strings"

	"minibili/internal/errcode"
)

const (
	maxCoverBytes  = 10 * 1024 * 1024
	maxAvatarBytes = 5 * 1024 * 1024
)

var allowedExt = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".webp": {},
}

// ValidateCoverHeader checks extension and size (Skill S-005).
func ValidateCoverHeader(fh *multipart.FileHeader) (code int) {
	return validateImageHeader(fh, maxCoverBytes, errcode.CodeCoverFormat, errcode.CodeCoverSize)
}

// ValidateAvatarHeader checks extension and size for user avatar (Rule R-BIZ-8, same ext set as S-005, 5MB cap).
func ValidateAvatarHeader(fh *multipart.FileHeader) (code int) {
	return validateImageHeader(fh, maxAvatarBytes, errcode.CodeAvatarFormat, errcode.CodeAvatarSize)
}

func validateImageHeader(fh *multipart.FileHeader, maxBytes int64, codeFormat, codeSize int) (code int) {
	if fh == nil {
		return 0
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if _, ok := allowedExt[ext]; !ok {
		return codeFormat
	}
	if fh.Size > maxBytes {
		return codeSize
	}
	return 0
}
