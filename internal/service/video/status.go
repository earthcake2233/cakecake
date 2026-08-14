package video

import "cakecake/internal/model/video"

// transcodeTransitions defines the legal status graph for the transcode
// pipeline. Every status change must pass ValidateTranscodeStatusTransition
// before being applied.
var transcodeTransitions = map[string][]string{
	video.StatusDraft:         {video.StatusProcessing, video.StatusPendingReview, video.StatusPublished},
	video.StatusProcessing:    {video.StatusPendingReview, video.StatusPublished, video.StatusFailed},
	video.StatusPendingReview: {video.StatusPublished, video.StatusRejected},
	video.StatusFailed:        {},
	video.StatusRejected:      {},
	video.StatusPublished:     {},
}

// ValidateTranscodeStatusTransition reports whether from -> to is a legal
// transcode state change.
func ValidateTranscodeStatusTransition(from, to string) bool {
	for _, next := range transcodeTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}
