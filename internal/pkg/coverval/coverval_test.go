package coverval

import (
	"bytes"
	"mime/multipart"
	"testing"
)

var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	pngMagic  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	gifMagic  = []byte{'G', 'I', 'F', '8', '9', 'a'}
	bmpMagic  = []byte{'B', 'M', 0x00, 0x00}
	webpMagic = []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P'}
)

func makeHeader(name string, content []byte) *multipart.FileHeader {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", name)
	if err != nil {
		panic(err)
	}
	if _, err := fw.Write(content); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	r := multipart.NewReader(&buf, w.Boundary())
	form, err := r.ReadForm(1024 * 1024)
	if err != nil {
		panic(err)
	}
	return form.File["file"][0]
}

func TestIsAllowedImageMagic(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want bool
	}{
		{"jpeg", jpegMagic, true},
		{"png", pngMagic, true},
		{"gif", gifMagic, true},
		{"bmp", bmpMagic, true},
		{"webp", webpMagic, true},
		{"empty", nil, false},
		{"text", []byte("hello world"), false},
		{"fake-png", []byte{0x89, 'P', 'N', 'G'}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAllowedImageMagic(tc.head); got != tc.want {
				t.Fatalf("IsAllowedImageMagic(%x) = %v, want %v", tc.head, got, tc.want)
			}
		})
	}
}

func TestValidateCoverHeader_ValidFormats(t *testing.T) {
	formats := []struct {
		name string
		head []byte
	}{
		{"jpg", jpegMagic},
		{"jpeg", jpegMagic},
		{"png", pngMagic},
		{"gif", gifMagic},
		{"bmp", bmpMagic},
		{"webp", webpMagic},
	}
	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			fh := makeHeader("cover."+f.name, f.head)
			if code := ValidateCoverHeader(fh); code != 0 {
				t.Fatalf("expected 0, got %d", code)
			}
		})
	}
}

func TestValidateCoverHeader_InvalidExt(t *testing.T) {
	fh := makeHeader("cover.txt", jpegMagic)
	if code := ValidateCoverHeader(fh); code != 40002 {
		t.Fatalf("expected 40002, got %d", code)
	}
	fh = makeHeader("cover.exe", jpegMagic)
	if code := ValidateCoverHeader(fh); code != 40002 {
		t.Fatalf("expected 40002, got %d", code)
	}
}

func TestValidateCoverHeader_MagicMismatch(t *testing.T) {
	// Extension says image, content says otherwise: must be rejected by magic bytes.
	fh := makeHeader("cover.jpg", []byte("not an image at all"))
	if code := ValidateCoverHeader(fh); code != 40002 {
		t.Fatalf("expected 40002, got %d", code)
	}
}

func TestValidateCoverHeader_SizeExceeded(t *testing.T) {
	big := make([]byte, 11*1024*1024)
	copy(big, jpegMagic)
	fh := makeHeader("cover.jpg", big)
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
	fh := makeHeader("avatar.jpg", jpegMagic)
	if code := ValidateAvatarHeader(fh); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestValidateAvatarHeader_InvalidExt(t *testing.T) {
	fh := makeHeader("avatar.txt", jpegMagic)
	if code := ValidateAvatarHeader(fh); code != 40015 {
		t.Fatalf("expected 40015, got %d", code)
	}
}

func TestValidateAvatarHeader_MagicMismatch(t *testing.T) {
	fh := makeHeader("avatar.png", []byte("plain text"))
	if code := ValidateAvatarHeader(fh); code != 40015 {
		t.Fatalf("expected 40015, got %d", code)
	}
}

func TestValidateAvatarHeader_SizeExceeded(t *testing.T) {
	big := make([]byte, 6*1024*1024)
	copy(big, pngMagic)
	fh := makeHeader("avatar.png", big)
	if code := ValidateAvatarHeader(fh); code != 40016 {
		t.Fatalf("expected 40016, got %d", code)
	}
}
