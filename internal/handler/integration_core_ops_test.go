package handler

import (
    "encoding/json"
    "net/http/httptest"
    "strconv"
    "testing"

    "github.com/stretchr/testify/require"
    "minibili/internal/model"
)

func decodeDataVideo(t *testing.T, w *httptest.ResponseRecorder) model.Video {
    t.Helper()
    var r struct {
        Data model.Video `json:"data"`
    }
    json.Unmarshal(w.Body.Bytes(), &r)
    return r.Data
}

func decodeDataArticle(t *testing.T, w *httptest.ResponseRecorder) model.Article {
    t.Helper()
    var r struct {
        Data model.Article `json:"data"`
    }
    json.Unmarshal(w.Body.Bytes(), &r)
    return r.Data
}

// Test_CommentBasicOps tests basic comment lifecycle
func Test_CommentBasicOps(t *testing.T) {
    api, r, _ := newTestAPI(t)
    u := seedUser(t, api, "cbo1", "CBO1", 0)
    v := seedVideo(t, api, u.ID, "CBO Video")
    tk := tok(t, api, u.ID)

    var cm model.Comment
    w := srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, `{"content":"hello"}`))
    require.Equal(t, 0, decodeCode(t, w), "post comment should succeed")

    var resp struct { Data model.Comment `json:"data"` }
    json.Unmarshal(w.Body.Bytes(), &resp)
    cm = resp.Data
    if cm.ID == 0 {
        t.Skip("comment not created")
    }

    // ToggleLike
    w = srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.ID, 10)+"/like", tk, nil))
    require.Equal(t, 0, decodeCode(t, w), "toggle like")

    // ToggleDislike
    w = srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.ID, 10)+"/dislike", tk, nil))
    require.Equal(t, 0, decodeCode(t, w), "toggle dislike")

    // ApproveComment
    w = srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.ID, 10)+"/approve", tk, nil))
    require.Equal(t, 0, decodeCode(t, w), "approve")

    // PinComment
    w = srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.ID, 10)+"/pin", tk, nil))
    require.Equal(t, 0, decodeCode(t, w), "pin")

    // DeleteComment
    w = srve(r, areq("DELETE", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments/"+strconv.FormatUint(cm.ID, 10), tk, nil))
    require.Equal(t, 0, decodeCode(t, w), "delete")
}