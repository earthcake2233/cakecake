package coverval

import (
	"bytes"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"cakecake/internal/errcode"
)

const (
	maxCoverBytes  = 10 * 1024 * 1024
	maxAvatarBytes = 5 * 1024 * 1024
	// magicProbeSize covers the longest signature checked (WebP: RIFF + WEBP at offset 8).
	magicProbeSize = 16
)

var allowedExt = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".webp": {},
}

// IsAllowedImageMagic reports whether head begins with a known image
// signature (JPEG / PNG / GIF / BMP / WebP). Cover validation must not trust
// the client-reported extension or MIME type alone, so every upload path
// checks the actual bytes.
func IsAllowedImageMagic(head []byte) bool {
	h := head
	if len(h) > magicProbeSize {
		h = h[:magicProbeSize]
	}
	switch {
	case bytes.HasPrefix(h, []byte{0xFF, 0xD8, 0xFF}): // JPEG
		return true
	case bytes.HasPrefix(h, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}): // PNG
		return true
	case bytes.HasPrefix(h, []byte{'G', 'I', 'F', '8'}): // GIF87a / GIF89a
		return true
	case bytes.HasPrefix(h, []byte{'B', 'M'}): // BMP
		return true
	case len(h) >= 12 &&
		bytes.Equal(h[:4], []byte{'R', 'I', 'F', 'F'}) &&
		bytes.Equal(h[8:12], []byte{'W', 'E', 'B', 'P'}): // WebP
		return true
	}
	return false
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
	f, err := fh.Open()
	if err != nil {
		return codeFormat
	}
	defer f.Close()
	head := make([]byte, magicProbeSize)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return codeFormat
	}
	if !IsAllowedImageMagic(head[:n]) {
		return codeFormat
	}
	return 0
}
