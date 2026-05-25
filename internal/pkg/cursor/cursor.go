package cursor

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// VideoListC is the cursor payload for published video list (F10 ordering).
type VideoListC struct {
	PlayCount    uint64    `json:"p"`
	CreatedAt    time.Time `json:"t"`
	DanmakuCount uint64    `json:"d"`
	ID           uint64    `json:"id"`
}

// Encode serializes cursor to opaque string.
func Encode(c VideoListC) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Decode parses cursor string.
func Decode(s string) (*VideoListC, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c VideoListC
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
