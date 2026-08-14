package queue

// TranscodeJob is the JSON payload on the transcode queue.
type TranscodeJob struct {
	VideoID    uint64 `json:"video_id"`
	RawPath    string `json:"raw_path"`
	CoverPath  string `json:"cover_path,omitempty"`
	RawKey     string `json:"raw_key,omitempty"`
	CoverKey   string `json:"cover_key,omitempty"`
	JobID      string `json:"job_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	RetryCount int    `json:"retry_count"`
}
