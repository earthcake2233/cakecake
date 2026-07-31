//go:build integration

package handler

import (
	"minibili/internal/model/comment"
	"minibili/internal/model/notification"
	"testing"
	"time"

	"strconv"
)

func Test_likeNotifPayloadSubject(t *testing.T) {
	api, _, _ := newTestAPI(t)
	tests := []struct {
		name string
		n    notification.Notification
		want string
	}{
		{name: "empty payload", n: notification.Notification{PayloadJSON: ""}, want: ""},
		{name: "article_comment", n: notification.Notification{PayloadJSON: `{"like_subject":"article_comment"}`}, want: "article_comment"},
		{name: "danmaku", n: notification.Notification{PayloadJSON: `{"like_subject":"danmaku"}`}, want: "danmaku"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := api.likeNotifPayloadSubject(&tt.n)
			if got != tt.want {
				t.Errorf("likeNotifPayloadSubject() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_consolidateDuplicateLikeAggregations(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "cda1", "CDA1", 10)
	for i := 0; i < 4; i++ {
		n := notification.Notification{
			RecipientID:     u.ID,
			Type:            "like_aggregation",
			RelatedID:       100,
			SenderNamesJSON: `["tester"]`,
			TotalLikes:      1,
			PayloadJSON:     `{"like_subject":"comment"}`,
			IsRead:          false,
			CreatedAt:       time.Now(),
		}
		api.DB.Create(&n)
	}
	api.consolidateDuplicateLikeAggregations(u.ID)
	var count int64
	api.DB.Model(&notification.Notification{}).Where("recipient_id = ? AND type = ? AND related_id = ?", u.ID, "like_aggregation", 100).Count(&count)
	t.Logf("remaining: %d", count)
}

func Test_consolidateLikeAggregationNotifs(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "cln1", "CLN1", 10)
	for i := 0; i < 3; i++ {
		n := notification.Notification{
			RecipientID:     u.ID,
			Type:            "like_aggregation",
			RelatedID:       uint64(100 + i),
			SenderNamesJSON: `["tester"]`,
			TotalLikes:      1,
			PayloadJSON:     `{"like_subject":"comment"}`,
			IsRead:          false,
			CreatedAt:       time.Now(),
		}
		api.DB.Create(&n)
	}
	api.consolidateLikeAggregationNotifs(u.ID, 100)
	var count int64
	api.DB.Model(&notification.Notification{}).Where("recipient_id = ?", u.ID).Count(&count)
	t.Logf("remaining: %d", count)
}

func Test_likeAggTotalFromDB(t *testing.T) {
	api, _, _ := newTestAPI(t)
	total := api.likeAggTotalFromDB(99999, false)
	if total != 0 {
		t.Errorf("got %d, want 0", total)
	}
	total = api.likeAggTotalFromDB(99999, true)
	if total != 0 {
		t.Errorf("got %d, want 0", total)
	}
}

func Test_likeAggTopLikerNames(t *testing.T) {
	api, _, _ := newTestAPI(t)
	names := api.likeAggTopLikerNames(99999, false, 3)
	if len(names) != 0 {
		t.Errorf("got %v, want nil", names)
	}
	names = api.likeAggTopLikerNames(99999, true, 3)
	if len(names) != 0 {
		t.Errorf("got %v, want nil", names)
	}
}

func Test_formatNotificationBasic(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "fnb1", "FNB1", 10)
	n := notification.Notification{
		ID: 1, RecipientID: u.ID, Type: "reply", SenderNamesJSON: `["sender"]`, TotalLikes: 0,
		CommentPreview: "Test", PayloadJSON: `{}`, IsRead: true, CreatedAt: time.Now(),
	}
	result := api.formatNotification(n)
	if result == nil {
		t.Fatal("nil result")
	}
	if result["type"] != "reply" {
		t.Errorf("type=%v", result["type"])
	}
	if result["is_read"] != true {
		t.Errorf("is_read=%v", result["is_read"])
	}
}

func Test_formatNotificationLikeAgg(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "fna1", "FNA1", 10)
	n := notification.Notification{
		ID: 2, RecipientID: u.ID, Type: "like_aggregation", SenderNamesJSON: `["a","b"]`,
		TotalLikes: 5, PayloadJSON: `{"like_subject":"comment"}`, IsRead: false, CreatedAt: time.Now(),
	}
	result := api.formatNotification(n)
	if result == nil {
		t.Fatal("nil")
	}
	if result["type"] != "like_aggregation" {
		t.Errorf("type=%v", result["type"])
	}
	if result["total_likes"] != 5 {
		t.Errorf("total_likes=%v", result["total_likes"])
	}
}

func Test_formatNotificationDanmakuLike(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "fnd1", "FND1", 10)
	n := notification.Notification{
		ID: 3, RecipientID: u.ID, Type: "like_aggregation", SenderNamesJSON: `["dl"]`,
		TotalLikes: 3, PayloadJSON: `{"like_subject":"danmaku"}`, IsRead: false, CreatedAt: time.Now(),
	}
	result := api.formatNotification(n)
	if result == nil {
		t.Fatal("nil")
	}
	if result["like_target"] != "弹幕" {
		t.Errorf("like_target=%v", result["like_target"])
	}
}

func Test_likeNotifTopSenders(t *testing.T) {
	api, _, _ := newTestAPI(t)
	urls, ids := api.likeNotifTopSenders([]string{}, 0)
	if len(urls) != 0 {
		t.Errorf("urls=%v", urls)
	}
	if len(ids) != 0 {
		t.Errorf("ids=%v", ids)
	}
	urls, ids = api.likeNotifTopSenders([]string{"nonexistent"}, 2)
	t.Logf("urls=%v, ids=%v", urls, ids)
}

func Test_clearCommentDislike(t *testing.T) {
	api, _, _ := newTestAPI(t)
	ok, err := api.clearCommentDislike(1, 99999)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for non-existent")
	}
}

func Test_clearCommentLike(t *testing.T) {
	api, _, _ := newTestAPI(t)
	// non-existent comment ID
	var cm comment.Comment
	ok, err := api.clearCommentLike(1, 99999, &cm)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false")
	}
}

func Test_resolveReplyInboxTarget(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "rir1", "RIR1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "Test Video")
	cm := comment.Comment{
		UserID:  u.ID,
		VideoID: v.ID,
		Content: "test comment",
	}
	api.DB.Create(&cm)
	tests := []struct {
		name string
		n    notification.Notification
		want replyInboxTarget
		ok   bool
	}{
		{name: "reply_received", n: notification.Notification{
			Type: "reply_received", RelatedID: cm.ID,
			PayloadJSON: `{"video_id":` + strconv.FormatUint(v.ID, 10) + `}`,
		}, want: replyInboxTarget{CommentID: cm.ID, VideoID: v.ID}, ok: true},
		{name: "like_agg", n: notification.Notification{Type: "like_aggregation"},
			want: replyInboxTarget{}, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := api.resolveReplyInboxTarget(&tt.n)
			if ok != tt.ok {
				t.Errorf("ok=%v, want %v", ok, tt.ok)
			}
			if got.CommentID != tt.want.CommentID || got.VideoID != tt.want.VideoID || got.ArticleID != tt.want.ArticleID {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
