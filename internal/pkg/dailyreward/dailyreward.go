package dailyreward

import (
	"time"

	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/pkg/usercoin"
)

const (
	ExpLogin       = 5
	ExpWatch       = 5
	ExpShare       = 5
	ExpCoinMax     = 50
	ExpPerCoinUnit = 10 // each「硬币」amount unit counts as 10 EXP toward the daily coin task
)

var cnLoc = time.FixedZone("CST", 8*3600)

func init() {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		cnLoc = loc
	}
}

// TodayDate is the reward calendar day in China time.
func TodayDate() string {
	return time.Now().In(cnLoc).Format("2006-01-02")
}

func dayBounds() (start, end time.Time) {
	d := TodayDate()
	start, _ = time.ParseInLocation("2006-01-02", d, cnLoc)
	end = start.Add(24 * time.Hour)
	return start, end
}

// TaskItem is one daily task for API responses.
type TaskItem struct {
	Exp  int  `json:"exp"`
	Done bool `json:"done"`
}

// CoinTask includes coin-task progress toward ExpCoinMax.
type CoinTask struct {
	Exp      int  `json:"exp"`
	Done     bool `json:"done"`
	Progress int  `json:"progress"`
	Max      int  `json:"max"`
}

// RewardsSnapshot is returned by GET /users/me/daily-rewards.
type RewardsSnapshot struct {
	Login TaskItem `json:"login"`
	Watch TaskItem `json:"watch"`
	Coin  CoinTask `json:"coin"`
	Share TaskItem `json:"share"`
}

// CoinProgress returns today's coin-task EXP progress (0–ExpCoinMax).
func CoinProgress(db *gorm.DB, uid uint64) int {
	start, end := dayBounds()
	var sum int64
	_ = db.Model(&model.VideoCoin{}).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", uid, start, end).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error
	p := int(sum) * ExpPerCoinUnit
	if p > ExpCoinMax {
		p = ExpCoinMax
	}
	return p
}

func ensureRow(db *gorm.DB, uid uint64) (*model.UserDailyTask, error) {
	date := TodayDate()
	var row model.UserDailyTask
	err := db.Where("user_id = ? AND task_date = ?", uid, date).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = model.UserDailyTask{UserID: uid, TaskDate: date}
		if err := db.Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func addUserExp(db *gorm.DB, uid uint64, delta uint64) error {
	if delta == 0 {
		return nil
	}
	return db.Model(&model.User{}).Where("id = ?", uid).
		UpdateColumn("experience", gorm.Expr("experience + ?", delta)).Error
}

// MarkLogin grants daily login reward once per calendar day.
func MarkLogin(db *gorm.DB, uid uint64) error {
	row, err := ensureRow(db, uid)
	if err != nil {
		return err
	}
	if row.LoginDone {
		return nil
	}
	if err := db.Model(row).Update("login_done", true).Error; err != nil {
		return err
	}
	if err := addUserExp(db, uid, ExpLogin); err != nil {
		return err
	}
	return usercoin.GrantDailyLoginCoin(db, uid)
}

// MarkWatch grants daily watch reward once per calendar day.
func MarkWatch(db *gorm.DB, uid uint64) error {
	row, err := ensureRow(db, uid)
	if err != nil {
		return err
	}
	if row.WatchDone {
		return nil
	}
	if err := db.Model(row).Update("watch_done", true).Error; err != nil {
		return err
	}
	return addUserExp(db, uid, ExpWatch)
}

// GrantCoinExp adds account EXP for newly earned coin-task progress (call after inserting VideoCoin).
func GrantCoinExp(db *gorm.DB, uid uint64, before, after int) error {
	delta := after - before
	if delta <= 0 {
		return nil
	}
	return addUserExp(db, uid, uint64(delta))
}

// BuildSnapshot builds the current daily-reward state for the user.
func BuildSnapshot(db *gorm.DB, uid uint64) (RewardsSnapshot, error) {
	row, err := ensureRow(db, uid)
	if err != nil {
		return RewardsSnapshot{}, err
	}
	cp := CoinProgress(db, uid)
	return RewardsSnapshot{
		Login: TaskItem{Exp: ExpLogin, Done: row.LoginDone},
		Watch: TaskItem{Exp: ExpWatch, Done: row.WatchDone},
		Coin: CoinTask{
			Exp:      ExpCoinMax,
			Done:     cp >= ExpCoinMax,
			Progress: cp,
			Max:      ExpCoinMax,
		},
		// PC 端无客户端分享链路，保持未完成。
		Share: TaskItem{Exp: ExpShare, Done: false},
	}, nil
}
