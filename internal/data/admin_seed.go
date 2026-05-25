package data

import (
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/model"
)

// SeedDefaultAdmin creates the first admin when table is empty and env seed is set.
func SeedDefaultAdmin(db *gorm.DB, cfg *config.C, lg *zap.Logger) error {
	if cfg == nil {
		return nil
	}
	user := strings.TrimSpace(cfg.AdminSeedUsername)
	pass := cfg.AdminSeedPassword
	if user == "" || pass == "" {
		return nil
	}
	var n int64
	if err := db.Model(&model.Admin{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a := model.Admin{
		Username:     user,
		PasswordHash: string(hash),
		DisplayName:  "运营管理员",
		Status:       "active",
	}
	if err := db.Create(&a).Error; err != nil {
		return err
	}
	if lg != nil {
		lg.Info("seed default admin created", zap.String("username", user), zap.Uint64("admin_id", a.ID))
	}
	return nil
}
