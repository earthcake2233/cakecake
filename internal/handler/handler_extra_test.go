package handler

import (
	"mime/multipart"
	"testing"
	"time"

	"minibili/internal/model"
)

// ---------- parseOptionalUnix ----------

func TestParseOptionalUnix(t *testing.T) {
	now := time.Unix(1710000000, 0)
	tests := []struct {
		name string
		p    *int64
		want *time.Time
	}{
		{"nil ptr", nil, nil},
		{"zero value", ptrInt64(0), nil},
		{"negative", ptrInt64(-1), nil},
		{"valid", ptrInt64(1710000000), &now},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseOptionalUnix(tc.p)
			if tc.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil, got nil")
			}
			if !got.Equal(*tc.want) {
				t.Errorf("got %v, want %v", got.Unix(), tc.want.Unix())
			}
		})
	}
}

func ptrInt64(v int64) *int64 { return &v }

// ---------- bannerImageExt ----------

func TestBannerImageExt(t *testing.T) {
	tests := []struct {
		name string
		fh   *multipart.FileHeader
		want string
	}{
		{"jpg", &multipart.FileHeader{Filename: "photo.jpg"}, "jpg"},
		{"jpeg to jpg", &multipart.FileHeader{Filename: "image.jpeg"}, "jpg"},
		{"png", &multipart.FileHeader{Filename: "banner.png"}, "png"},
		{"webp", &multipart.FileHeader{Filename: "pic.webp"}, "webp"},
		{"uppercase JPG", &multipart.FileHeader{Filename: "photo.JPG"}, "jpg"},
		{"no ext", &multipart.FileHeader{Filename: "photo"}, "jpg"},
		{"dot only", &multipart.FileHeader{Filename: "photo."}, "jpg"},
		{"gif", &multipart.FileHeader{Filename: "anim.gif"}, "gif"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bannerImageExt(tc.fh)
			if got != tc.want {
				t.Errorf("bannerImageExt(%+v) = %q, want %q", tc.fh, got, tc.want)
			}
		})
	}
}

// ---------- adminVideoStatusFilter ----------

func TestAdminVideoStatusFilter(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"all", nil},
		{"  all  ", nil},
		{"pending_review", []string{"pending_review"}},
		{"pending", []string{"pending_review"}},
		{"published", []string{"published"}},
		{"passed", []string{"published"}},
		{"rejected", []string{"rejected"}},
		{"processing", []string{"processing"}},
		{"failed", []string{"failed"}},
		{"custom_status", []string{"custom_status"}},
		{"  draft  ", []string{"draft"}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := adminVideoStatusFilter(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d]=%q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// ---------- adminArticleStatusFilter ----------

func TestAdminArticleStatusFilter(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"all", nil},
		{"pending_review", []string{"pending_review"}},
		{"pending", []string{"pending_review"}},
		{"published", []string{"published"}},
		{"passed", []string{"published"}},
		{"rejected", []string{"rejected"}},
		{"draft", []string{"draft"}},
		{"  draft  ", []string{"draft"}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := adminArticleStatusFilter(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d]=%q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// ---------- adminVideoToJSON ----------

func TestAdminVideoToJSON(t *testing.T) {
	now := time.Now()
	v := &model.Video{
		ID:                1,
		Title:             "Test Video",
		Description:       "A description",
		Status:            "published",
		FailReason:        "  ",
		CoverURL:          "https://example.com/cover.jpg",
		VideoURL:          "https://example.com/video.mp4",
		DurationSec:       120.5,
		Zone:              "Life-Daily",
		UserID:            42,
		PlayCount:         1000,
		DanmakuCount:      50,
		CommentCount:      10,
		LikeCount:         200,
		FavCount:          50,
		CoinCount:         30,
		CommentsClosed:    false,
		CommentsCurated:   false,
		DraftRawPath:      "",
		DraftCoverPath:    "",
		ReviewedAt:        &now,
		ReviewedByAdminID: ptrUint64(5),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	out := adminVideoToJSON(v, "uploader")
	if out["title"] != "Test Video" {
		t.Errorf("title = %q", out["title"])
	}
	if out["fail_reason"] != "" {
		t.Errorf("fail_reason should be trimmed, got %q", out["fail_reason"])
	}
	if out["reviewed_by_admin_id"] != uint64(5) {
		t.Errorf("reviewed_by_admin_id = %v", out["reviewed_by_admin_id"])
	}
	if _, ok := out["reviewed_at"]; !ok {
		t.Error("reviewed_at should be present")
	}
	if out["uploader_name"] != "uploader" {
		t.Errorf("uploader_name = %q", out["uploader_name"])
	}
	// Without reviewed fields
	v2 := &model.Video{ID: 2, Title: "No Review", Status: "draft"}
	out2 := adminVideoToJSON(v2, "")
	if _, ok := out2["reviewed_at"]; ok {
		t.Error("reviewed_at should NOT be present")
	}
	if _, ok := out2["reviewed_by_admin_id"]; ok {
		t.Error("reviewed_by_admin_id should NOT be present")
	}
}

func ptrUint64(v uint64) *uint64 { return &v }

// ---------- adminArticleToJSON ----------

func TestAdminArticleToJSON(t *testing.T) {
	now := time.Now()
	pubAt := now.Add(-time.Hour)
	art := &model.Article{
		ID:                10,
		Title:             "My Article",
		CoverURL:          "https://example.com/cover.png",
		BodyMD:            "# Hello\nThis is **bold**.",
		Status:            "published",
		FailReason:        "  ",
		UserID:            7,
		ViewCount:         500,
		CommentCount:      20,
		CoinCount:         10,
		FavCount:          5,
		ForwardCount:      2,
		CommentsClosed:    false,
		CommentsCurated:   false,
		TagsJSON:          `["tech"]`,
		PublishedAt:       &pubAt,
		ReviewedAt:        &now,
		ReviewedByAdminID: ptrUint64(3),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	out := adminArticleToJSON(art, "author")
	if out["title"] != "My Article" {
		t.Errorf("title = %q", out["title"])
	}
	if out["fail_reason"] != "" {
		t.Errorf("fail_reason should be trimmed, got %q", out["fail_reason"])
	}
	if out["uploader_name"] != "author" {
		t.Errorf("uploader_name = %q", out["uploader_name"])
	}
	if out["user_id"] != uint64(7) {
		t.Errorf("user_id = %v", out["user_id"])
	}
	if _, ok := out["reviewed_at"]; !ok {
		t.Error("reviewed_at should be present")
	}
	if out["reviewed_by_admin_id"] != uint64(3) {
		t.Errorf("reviewed_by_admin_id = %v", out["reviewed_by_admin_id"])
	}
	if out["body_html"] == "" {
		t.Error("body_html should be rendered")
	}
	// Without reviewed fields
	art2 := &model.Article{ID: 11, Title: "Draft", BodyMD: "draft", Status: "draft"}
	out2 := adminArticleToJSON(art2, "")
	if _, ok := out2["reviewed_at"]; ok {
		t.Error("reviewed_at should NOT be present without reviewed fields")
	}
	if _, ok := out2["reviewed_by_admin_id"]; ok {
		t.Error("reviewed_by_admin_id should NOT be present")
	}
}

// ---------- adminDynamicToJSON ----------

func TestAdminDynamicToJSON(t *testing.T) {
	now := time.Now()
	d := &model.UserDynamic{
		ID:              99,
		Title:           "My Day",
		Content:         "Had a great day!",
		ImagesJSON:      `["https://img.example.com/1.jpg","https://img.example.com/2.jpg"]`,
		UserID:          42,
		LikeCount:       100,
		CommentCount:    5,
		CommentsClosed:  false,
		CommentsCurated: false,
		CreatedAt:       now,
	}
	out := adminDynamicToJSON(d, "nickname")
	if out["title"] != "My Day" {
		t.Errorf("title = %q", out["title"])
	}
	if out["cover_url"] != "https://img.example.com/1.jpg" {
		t.Errorf("cover_url = %q", out["cover_url"])
	}
	if out["uploader_name"] != "nickname" {
		t.Errorf("uploader_name = %q", out["uploader_name"])
	}
	if out["user_id"] != uint64(42) {
		t.Errorf("user_id = %v", out["user_id"])
	}
	// Empty images
	d2 := &model.UserDynamic{ID: 100, Title: "No Images", Content: "text"}
	out2 := adminDynamicToJSON(d2, "")
	imgs := out2["images"].([]string)
	if len(imgs) != 0 {
		t.Errorf("expected empty images, got %v", imgs)
	}
	if out2["cover_url"] != "" {
		t.Errorf("expected empty cover_url, got %q", out2["cover_url"])
	}
}

// ---------- bannerToJSON ----------

func TestBannerToJSON(t *testing.T) {
	now := time.Now()
	b := &model.HomeBanner{
		ID:          5,
		Title:       "Summer Sale",
		ImageURL:    "https://img.example.com/banner.jpg",
		LinkType:    "url",
		LinkTarget:  "https://example.com",
		SortOrder:   1,
		Enabled:     true,
		StartAt:     &now,
		EndAt:       nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	out := bannerToJSON(b)
	if out["title"] != "Summer Sale" {
		t.Errorf("title = %q", out["title"])
	}
	if out["link_type"] != "url" {
		t.Errorf("link_type = %q", out["link_type"])
	}
	if out["sort_order"] != 1 {
		t.Errorf("sort_order = %v", out["sort_order"])
	}
}

// ---------- hotSearchOpToJSON ----------

func TestHotSearchOpToJSON(t *testing.T) {
	now := time.Now()
	op := &model.HotSearchOp{
		ID:           7,
		OpType:       "pin",
		Keyword:      "summer",
		DisplayTitle: "Summer Hot",
		Badge:        "Hot",
		PinRank:      1,
		Enabled:      true,
		StartAt:      &now,
		EndAt:        nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	out := hotSearchOpToJSON(op)
	if out["op_type"] != "pin" {
		t.Errorf("op_type = %q", out["op_type"])
	}
	if out["keyword"] != "summer" {
		t.Errorf("keyword = %q", out["keyword"])
	}
	if out["display_title"] != "Summer Hot" {
		t.Errorf("display_title = %q", out["display_title"])
	}
	if out["pin_rank"] != 1 {
		t.Errorf("pin_rank = %v", out["pin_rank"])
	}
}

// ---------- deletionCoolingDays ----------

func TestDeletionCoolingDays(t *testing.T) {
	for i := 0; i < 100; i++ {
		days := deletionCoolingDays()
		if days < 7 || days > 30 {
			t.Errorf("deletionCoolingDays() = %d, want between 7 and 30", days)
		}
	}
}

// ---------- videoStatusAllowsMediaReplace ----------

func TestVideoStatusAllowsMediaReplace(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"failed", true},
		{"rejected", true},
		{"draft", false},
		{"published", false},
		{"pending_review", false},
		{"processing", false},
		{"", false},
		{"unknown", false},
	}
	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			got := videoStatusAllowsMediaReplace(tc.status)
			if got != tc.want {
				t.Errorf("videoStatusAllowsMediaReplace(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

// ---------- folderIDsFromMap ----------

func TestFolderIDsFromMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[uint64]bool
		want []uint64
	}{
		{"nil", nil, []uint64{}},
		{"empty", map[uint64]bool{}, []uint64{}},
		{"single", map[uint64]bool{1: true}, []uint64{1}},
		{"multiple", map[uint64]bool{1: true, 2: true, 3: true}, []uint64{1, 2, 3}},
		{"skip zero", map[uint64]bool{0: true, 1: true}, []uint64{1}},
		{"all zero", map[uint64]bool{0: true}, []uint64{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := folderIDsFromMap(tc.m)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d, want %d: %v", len(got), len(tc.want), got)
			}
			seen := make(map[uint64]bool)
			for _, id := range got {
				if seen[id] {
					t.Errorf("duplicate %d", id)
				}
				seen[id] = true
				if !tc.m[id] {
					t.Errorf("unexpected %d", id)
				}
			}
		})
	}
}

// ---------- uploaderNameForAPI ----------

func TestUploaderNameForAPI_Extra(t *testing.T) {
	anon := time.Now()
	tests := []struct {
		name string
		u    *model.User
		want string
	}{
		{"nil user", nil, ""},
		{"nickname set", &model.User{Nickname: "CoolNick", Username: "user123"}, "CoolNick"},
		{"nickname with spaces", &model.User{Nickname: "  Spaced  ", Username: "user456"}, "Spaced"},
		{"no nickname", &model.User{Username: "plain_user"}, "plain_user"},
		{"empty nickname", &model.User{Nickname: "", Username: "fallback"}, "fallback"},
		{"anonymized with nickname", &model.User{Nickname: "OldNick", Username: "user789", AnonymizedAt: &anon}, "已注销用户"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := uploaderNameForAPI(tc.u)
			if got != tc.want {
				t.Errorf("uploaderNameForAPI(%+v) = %q, want %q", tc.u, got, tc.want)
			}
		})
	}
}

// ---------- validateArticleContent ----------

func TestValidateArticleContent_Extra(t *testing.T) {
	title81 := string(make([]rune, 81))
	body100001 := string(make([]rune, 100001))

	tests := []struct {
		name    string
		title   string
		bodyMD  string
		publish bool
		want    bool
	}{
		// publish mode
		{"publish valid", "Title", "Body", true, true},
		{"publish empty title", "", "Body", true, false},
		{"publish empty body", "Title", "", true, false},
		{"publish both empty", "", "", true, false},
		{"publish title too long", title81, "Body", true, false},
		{"publish body too long", "Title", body100001, true, false},
		{"publish trimmed spaces", "  Title  ", "  Body  ", true, true},
		// draft mode
		{"draft valid both", "Title", "Body", false, true},
		{"draft title only", "Title", "", false, true},
		{"draft body only", "", "Body", false, true},
		{"draft both empty", "", "", false, false},
		{"draft title too long", title81, "Body", false, false},
		{"draft body too long", "Title", body100001, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateArticleContent(tc.title, tc.bodyMD, tc.publish)
			if got != tc.want {
				t.Errorf("validateArticleContent(%q, bodyMD(%d), %v) = %v, want %v",
					tc.title, len(tc.bodyMD), tc.publish, got, tc.want)
			}
		})
	}
}

// ---------- formatDanmakuPlayTime extra edge ----------



// ---------- extra edge cases ----------

func TestAdminVideoToJSONMinimal(t *testing.T) {
	v := &model.Video{ID: 1, Title: "Minimal", Description: "desc", Status: "draft"}
	out := adminVideoToJSON(v, "")
	if out["duration_sec"] != float64(0) {
		t.Errorf("duration_sec = %v", out["duration_sec"])
	}
}

func TestAdminArticleToJSONMinimal(t *testing.T) {
	art := &model.Article{ID: 1, Title: "Minimal", BodyMD: "body"}
	out := adminArticleToJSON(art, "")
	if out["published_at"] != "" {
		t.Errorf("published_at = %q, want empty", out["published_at"])
	}
}


func TestParseOptionalUnixEdge(t *testing.T) {
	large := int64(32503680000) // year 3000
	got := parseOptionalUnix(&large)
	if got == nil {
		t.Fatal("expected non-nil for large timestamp")
	}
	if got.Unix() != large {
		t.Errorf("got %d, want %d", got.Unix(), large)
	}
}

func TestDeletionCoolingDaysRange(t *testing.T) {
	hist := make(map[int]int)
	for i := 0; i < 1000; i++ {
		d := deletionCoolingDays()
		if d < 7 || d > 30 {
			t.Fatalf("out of range: %d", d)
		}
		hist[d]++
	}
	if len(hist) < 4 {
		t.Errorf("expected at least 4 distinct values, got %d", len(hist))
	}
}

