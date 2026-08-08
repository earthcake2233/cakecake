package worker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TranscodeJobRoundtrip(t *testing.T) {
	job := TranscodeJob{VideoID: 1, RawPath: "/tmp/video.mp4", CoverPath: "/tmp/cover.jpg", RetryCount: 1}
	b, err := json.Marshal(job)
	require.NoError(t, err)

	var j2 TranscodeJob
	require.NoError(t, json.Unmarshal(b, &j2))
	require.Equal(t, job.VideoID, j2.VideoID)
	require.Equal(t, job.RawPath, j2.RawPath)
	require.Equal(t, job.RetryCount, j2.RetryCount)
}

func Test_TranscodeJobZeroRetry(t *testing.T) {
	job := TranscodeJob{VideoID: 2, RawPath: "/tmp/v2.mp4"}
	b, _ := json.Marshal(job)
	var j2 TranscodeJob
	require.NoError(t, json.Unmarshal(b, &j2))
	require.Equal(t, 0, j2.RetryCount)
	require.Empty(t, j2.CoverPath)
}

func Test_TranscodeJobMarshalUnmarshal(t *testing.T) {
	cases := []TranscodeJob{
		{VideoID: 10, RawPath: "a.mp4", RetryCount: 3},
		{VideoID: 20, RawPath: "b.mp4", CoverPath: "c.jpg", RetryCount: 0},
		{VideoID: 30, RawPath: "d.mp4", RetryCount: 5},
	}
	for _, tc := range cases {
		b, _ := json.Marshal(tc)
		var got TranscodeJob
		require.NoError(t, json.Unmarshal(b, &got))
		require.Equal(t, tc.VideoID, got.VideoID)
		require.Equal(t, tc.RawPath, got.RawPath)
	}
}

func Test_TranscodeJobEmptyRawPath(t *testing.T) {
	job := TranscodeJob{VideoID: 1}
	b, _ := json.Marshal(job)
	require.Contains(t, string(b), `"video_id":1`)
	// Empty string fields still appear in JSON without omitempty
	require.Contains(t, string(b), `"raw_path":""`)
}
