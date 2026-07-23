package toolkit

import "minibili/internal/aigateway"

// Tool names (constants for admin registry keys).
const (
	ToolSearchVideos    = "search_videos"
	ToolGetVideoDetail  = "get_video_detail"
	ToolGetTrending     = "get_trending"
	ToolGetVideoComments = "get_video_comments"
	ToolGetVideoDanmaku = "get_video_danmaku"
)

// AllToolNames returns all defined tool names.
func AllToolNames() []string {
	return []string{
		ToolSearchVideos,
		ToolGetVideoDetail,
		ToolGetTrending,
		ToolGetVideoComments,
		ToolGetVideoDanmaku,
	}
}

// DefineTools returns the tool definitions for enabled tools.
// enabled is a map of tool name -> bool.
func DefineTools(enabled map[string]bool) []aigateway.ToolDef {
	if enabled == nil {
		enabled = defaultEnabled()
	}
	var tools []aigateway.ToolDef
	for _, name := range AllToolNames() {
		if !enabled[name] {
			continue
		}
		tools = append(tools, definition(name))
	}
	return tools
}

func defaultEnabled() map[string]bool {
	m := make(map[string]bool)
	for _, name := range AllToolNames() {
		m[name] = true
	}
	return m
}

func definition(name string) aigateway.ToolDef {
	switch name {
	case ToolSearchVideos:
		return aigateway.ToolDef{
			Type: "function",
			Function: aigateway.ToolFuncDef{
				Name:        ToolSearchVideos,
				Description: "Search videos by keyword. Returns a list of matching videos with title, uploader, play count, and duration.",
				Parameters: Object(map[string]interface{}{
					"keyword":   S("Search keyword (required)."),
					"page":      I("Page number, starting from 1. Default 1."),
					"page_size": I("Results per page. Max 20. Default 10."),
				}, "keyword"),
			},
		}
	case ToolGetVideoDetail:
		return aigateway.ToolDef{
			Type: "function",
			Function: aigateway.ToolFuncDef{
				Name:        ToolGetVideoDetail,
				Description: "Get detailed information about a video by its ID. Returns title, description, uploader, play/like/comment counts, duration, and zone.",
				Parameters: Object(map[string]interface{}{
					"video_id": I("The unique video ID (required)."),
				}, "video_id"),
			},
		}
	case ToolGetTrending:
		return aigateway.ToolDef{
			Type: "function",
			Function: aigateway.ToolFuncDef{
				Name:        ToolGetTrending,
				Description: "Get the current trending/hot search list. Returns ranked list of trending keywords with popularity badges.",
				Parameters: Object(map[string]interface{}{
					"limit": I("Number of trending items to return. Max 20. Default 10."),
				}),
			},
		}
	case ToolGetVideoComments:
		return aigateway.ToolDef{
			Type: "function",
			Function: aigateway.ToolFuncDef{
				Name:        ToolGetVideoComments,
				Description: "Get comments for a specific video. Returns a list of comments with author, content, and like count.",
				Parameters: Object(map[string]interface{}{
					"video_id": I("The unique video ID (required)."),
					"page":     I("Page number. Default 1."),
					"page_size": I("Comments per page. Max 20. Default 10."),
				}, "video_id"),
			},
		}
	case ToolGetVideoDanmaku:
		return aigateway.ToolDef{
			Type: "function",
			Function: aigateway.ToolFuncDef{
				Name:        ToolGetVideoDanmaku,
				Description: "Get danmaku (bullet comments) for a specific video. Returns recent danmaku entries with content, time offset, and type.",
				Parameters: Object(map[string]interface{}{
					"video_id": I("The unique video ID (required)."),
					"limit":    I("Number of danmaku entries to return. Max 50. Default 20."),
				}, "video_id"),
			},
		}
	default:
		return aigateway.ToolDef{}
	}
}