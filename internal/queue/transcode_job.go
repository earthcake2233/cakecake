package queue

// TranscodeJob is the JSON payload on the transcode queue.
type TranscodeJob struct {
	VideoID    uint64 `json:"video_id"`
	RawPath    string `json:"raw_path"`
	CoverPath  string `json:"cover_path,omitempty"`
	RetryCount int    `json:"retry_count"`
}
