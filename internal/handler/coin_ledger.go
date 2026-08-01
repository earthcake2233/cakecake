package handler

import (
	"cakecake/internal/model/user"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cakecake/internal/middleware"
	"cakecake/internal/pkg/bvid"
	"go.uber.org/zap"

	"cakecake/internal/errcode"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/usercoin"
)

// ListMeCoinLedger returns paginated coin change history for the personal-center page.
// Query: range=month|week (default month), limit, offset.
func (a *API) ListMeCoinLedger(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	rng := strings.TrimSpace(c.Query("range"))
	if rng == "" {
		rng = "month"
	}
	var since time.Time
	now := time.Now()
	switch rng {
	case "week":
		since = now.AddDate(0, 0, -7)
	default:
		since = now.AddDate(0, 0, -30)
		rng = "month"
	}
	limit := parseLimit(c, 10, 100)
	offset := 0
	if v, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && v >= 0 {
		offset = v
	}

	total, rows, err := a.UserSvc.ListCoinLedger(c.Request.Context(), uid, since, limit, offset)
	if err != nil {
		a.Log.Error("list coin ledger", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, formatCoinLedgerItem(&rows[i]))
	}
	resp.OK(c, gin.H{
		"range":    rng,
		"total":    total,
		"has_more": int64(offset+len(rows)) < total,
		"items":    items,
	})
}

func formatCoinLedgerItem(row *user.CoinLedger) gin.H {
	change := float64(row.DeltaTenths) / float64(usercoin.TenthsPerCoin)
	reason := coinLedgerReasonText(row)
	return gin.H{
		"created_at": row.CreatedAt.Format("2006-01-02 15:04:05"),
		"change":     change,
		"reason":     reason,
	}
}

func coinLedgerReasonText(row *user.CoinLedger) string {
	switch row.ReasonType {
	case usercoin.ReasonLoginReward:
		return "登录奖励"
	case usercoin.ReasonNicknameChange:
		return "修改昵称"
	case usercoin.ReasonVideoTip:
		if row.VideoID > 0 {
			return "给视频 " + bvid.Encode(row.VideoID) + " 打赏"
		}
		return "给视频打赏"
	case usercoin.ReasonVideoTipIncome:
		if row.VideoID > 0 {
			return "给视频 " + bvid.Encode(row.VideoID) + " 打赏"
		}
		return "给视频打赏"
	default:
		return "硬币变动"
	}
}
