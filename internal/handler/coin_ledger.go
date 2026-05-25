package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/bvid"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/usercoin"
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
	limit := 10
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	offset := 0
	if v, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && v >= 0 {
		offset = v
	}

	q := a.DB.Model(&model.CoinLedger{}).Where("user_id = ? AND created_at >= ?", uid, since)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		a.Log.Error("count coin ledger", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var rows []model.CoinLedger
	if err := q.Order("created_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
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

func formatCoinLedgerItem(row *model.CoinLedger) gin.H {
	change := float64(row.DeltaTenths) / float64(usercoin.TenthsPerCoin)
	reason := coinLedgerReasonText(row)
	return gin.H{
		"created_at": row.CreatedAt.Format("2006-01-02 15:04:05"),
		"change":     change,
		"reason":     reason,
	}
}

func coinLedgerReasonText(row *model.CoinLedger) string {
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
