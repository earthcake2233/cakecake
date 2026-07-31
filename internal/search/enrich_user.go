package search

import (
	"minibili/internal/model/article"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"strings"

	"gorm.io/gorm"

	"minibili/internal/pkg/useravatar"
	"minibili/internal/pkg/userlevel"
)

// EnrichUserHits fills profile stats, follow state, and recent archives from MySQL.
func EnrichUserHits(db *gorm.DB, viewer uint64, hits []UserHit) []UserHit {
	for i := range hits {
		var u user.User
		if err := db.First(&u, hits[i].Mid).Error; err != nil {
			continue
		}
		if user.IsUserAnonymized(&u) {
			hits[i].Uname = "已注销用户"
			hits[i].Usign = ""
			hits[i].Face = ""
			hits[i].Recent = nil
			continue
		}
		hits[i].Face = useravatar.PublicURL(&u)
		if hits[i].Usign == "" {
			hits[i].Usign = strings.TrimSpace(u.Sign)
		}
		hits[i].Level = userlevel.FromExperience(u.Experience).CurrentLevel

		var videoCnt, articleCnt int64
		_ = db.Model(&video.Video{}).Where("user_id = ? AND status = ?", u.ID, video.StatusPublished).Count(&videoCnt).Error
		_ = db.Model(&article.Article{}).Where("user_id = ? AND status = ?", u.ID, article.StatusPublished).Count(&articleCnt).Error
		hits[i].Archives = int(videoCnt + articleCnt)

		var fanCnt int64
		_ = db.Model(&user.UserFollow{}).Where("followee_id = ?", u.ID).Count(&fanCnt).Error
		hits[i].Fans = int(fanCnt)

		hits[i].FollowedByMe = false
		if viewer > 0 && viewer != u.ID {
			var rel int64
			_ = db.Model(&user.UserFollow{}).
				Where("follower_id = ? AND followee_id = ?", viewer, u.ID).
				Count(&rel).Error
			hits[i].FollowedByMe = rel > 0
		}

		hits[i].Recent = recentArchivesForUser(db, u.ID, 3)
	}
	return hits
}

func recentArchivesForUser(db *gorm.DB, userID uint64, limit int) []UserArchiveItem {
	if limit <= 0 {
		return nil
	}
	var videos []video.Video
	_ = db.Where("user_id = ? AND status = ?", userID, video.StatusPublished).
		Order("id DESC").
		Limit(limit).
		Find(&videos).Error
	out := make([]UserArchiveItem, 0, len(videos))
	for _, v := range videos {
		out = append(out, UserArchiveItem{
			Aid:     v.ID,
			Title:   v.Title,
			Pic:     v.CoverURL,
			Pubdate: v.CreatedAt.Unix(),
			Rtype:   "video",
		})
	}
	if len(out) >= limit {
		return out
	}
	remain := limit - len(out)
	var articles []article.Article
	_ = db.Where("user_id = ? AND status = ?", userID, article.StatusPublished).
		Order("COALESCE(published_at, created_at) DESC, id DESC").
		Limit(remain).
		Find(&articles).Error
	for _, a := range articles {
		pub := a.CreatedAt.Unix()
		if a.PublishedAt != nil {
			pub = a.PublishedAt.Unix()
		}
		out = append(out, UserArchiveItem{
			Aid:     a.ID,
			Title:   a.Title,
			Pic:     a.CoverURL,
			Pubdate: pub,
			Rtype:   "article",
		})
	}
	return out
}
