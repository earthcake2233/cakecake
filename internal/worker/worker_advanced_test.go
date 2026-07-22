package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"

	"minibili/internal/logger"
	"minibili/internal/model"
)

type mockAcknowledger struct {
	ackCalled  bool
	nackCalled bool
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.ackCalled = true
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	m.nackCalled = true
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	m.nackCalled = true
	return nil
}

func init() {
	logger.L = zap.NewNop()
}

func TestTruncate_EdgeCases_Adv(t *testing.T) {
	tests := []struct {
		input string
		limit int
		want  string
	}{
		{"hello", 5, "hello"},
		{"hello", 10, "hello"},
		{"hello", 3, "hel"},
		{"", 5, ""},
		{"abc", 0, ""},
		{"abcdef", 6, "abcdef"},
		{"hello world", 5, "hello"},
	}
	for _, tc := range tests {
		got := truncate(tc.input, tc.limit)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.limit, got, tc.want)
		}
	}
}

func TestCleanupPaths_Nxst(t *testing.T) {
	cleanupPaths("/tmp/nonexistent_file_xyz", "/tmp/nonexistent2_xyz")
}

func TestCleanupPaths_MixNxst(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "keep.txt")
	f2 := filepath.Join(dir, "missing.txt")
	os.WriteFile(f1, []byte("data"), 0644)
	cleanupPaths(f1, f2)
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Error("expected f1 to be removed")
	}
}

func TestTranscodeJob_JSON_Adv(t *testing.T) {
	job := TranscodeJob{VideoID: 42, RawPath: "/tmp/raw.mp4", CoverPath: "/tmp/cover.jpg", RetryCount: 2}
	data, err := json.Marshal(job)
	require.NoError(t, err)
	var decoded TranscodeJob
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), decoded.VideoID)
	assert.Equal(t, "/tmp/raw.mp4", decoded.RawPath)
	assert.Equal(t, "/tmp/cover.jpg", decoded.CoverPath)
	assert.Equal(t, 2, decoded.RetryCount)
}

func TestTranscodeJob_MinJSON_Adv(t *testing.T) {
	job := TranscodeJob{VideoID: 1, RawPath: "/tmp/v.mp4"}
	data, err := json.Marshal(job)
	require.NoError(t, err)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	assert.Equal(t, float64(1), m["video_id"])
	assert.Equal(t, "/tmp/v.mp4", m["raw_path"])
}

func TestHandleDelivery_BadJSON_Adv(t *testing.T) {
	mockAck := &mockAcknowledger{}
	d := amqp.Delivery{Acknowledger: mockAck, Body: []byte("not json")}
	handleDelivery(nil, nil, nil, nil, nil, nil, nil, d)
	assert.True(t, mockAck.ackCalled)
}

func TestHandleDelivery_EmptyBody_Adv(t *testing.T) {
	mockAck := &mockAcknowledger{}
	d := amqp.Delivery{Acknowledger: mockAck, Body: []byte{}}
	handleDelivery(nil, nil, nil, nil, nil, nil, nil, d)
	assert.True(t, mockAck.ackCalled)
}

func TestHandleDelivery_NilOSS_Adv(t *testing.T) {
	db := setupWorkerDB_Adv(t)
	db.Create(&model.Video{Title: "No OSS", Status: "transcoding"})
	job := TranscodeJob{VideoID: 1, RawPath: "/tmp/v.mp4"}
	body, _ := json.Marshal(job)
	mockAck := &mockAcknowledger{}
	d := amqp.Delivery{Acknowledger: mockAck, Body: body}
	handleDelivery(nil, nil, db, nil, nil, nil, nil, d)
	assert.True(t, mockAck.ackCalled)
	var v model.Video
	db.First(&v, 1)
	assert.Equal(t, "failed", v.Status)
}

func setupWorkerDB_Adv(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Video{}))
	return db
}
