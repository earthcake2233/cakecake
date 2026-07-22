package coverval

import (
	"mime/multipart"
	"testing"
)

func makeHeader(name string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: name,
		Size:     size,
	}
}

func TestValidateCoverHeader_ValidJPG(t *testing.T) {
	fh := makeHeader("cover.jpg", 1024)
	if code := ValidateCoverHeader(fh); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestValidateCoverHeader_ValidPNG(t *testing.T) {
	fh := makeHeader("cover.png", 5*1024*1024)
	if code := ValidateCoverHeader(fh); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestValidateCoverHeader_ValidWEBP(t *testing.T) {
	fh := makeHeader("cover.webp", 1024)
	if code := ValidateCoverHeader(fh); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestValidateCoverHeader_InvalidExt(t *testing.T) {
	fh := makeHeader("cover.txt", 1024)
	if code := ValidateCoverHeader(fh); code != 40002 {
		t.Fatalf("expected 40002, got %d", code)
	}
}

func TestValidateCoverHeader_InvalidExtExe(t *testing.T) {
	fh := makeHeader("cover.exe", 1024)
	if code := ValidateCoverHeader(fh); code != 40002 {
		t.Fatalf("expected 40002, got %d", code)
	}
}

func TestValidateCoverHeader_SizeExceeded(t *testing.T) {
	fh := makeHeader("cover.jpg", 11*1024*1024)
	if code := ValidateCoverHeader(fh); code != 40003 {
		t.Fatalf("expected 40003, got %d", code)
	}
}

func TestValidateCoverHeader_Nil(t *testing.T) {
	if code := ValidateCoverHeader(nil); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestValidateAvatarHeader_ValidJPG(t *testing.T) {
	fh := makeHeader("avatar.jpg", 1024)
	if code := ValidateAvatarHeader(fh); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestValidateAvatarHeader_InvalidExt(t *testing.T) {
	fh := makeHeader("avatar.txt", 1024)
	if code := ValidateAvatarHeader(fh); code != 40015 {
		t.Fatalf("expected 40015, got %d", code)
	}
}

func TestValidateAvatarHeader_SizeExceeded(t *testing.T) {
	fh := makeHeader("avatar.jpg", 6*1024*1024)
	if code := ValidateAvatarHeader(fh); code != 40016 {
		t.Fatalf("expected 40016, got %d", code)
	}
}

func TestValidateAvatarHeader_Nil(t *testing.T) {
	if code := ValidateAvatarHeader(nil); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}
