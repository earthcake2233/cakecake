package service

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
)

// UserService handles user profile business logic.
type UserService struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUserService(db *gorm.DB, log *zap.Logger) *UserService {
	return &UserService{db: db, log: log}
}

type UserProfile struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Sign      string `json:"sign"`
	CakeID    string `json:"cake_id"`
	CreatedAt string `json:"created_at"`
	Gender    string `json:"gender"`
	Birthday  string `json:"birthday"`
	Announcement string `json:"announcement"`
}

// GetMe returns the current user's profile.
func (s *UserService) GetMe(ctx context.Context, userID uint64) (*UserProfile, error) {
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &UserProfile{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname,
		AvatarURL: u.AvatarURL, Sign: u.Sign, CakeID: u.CakeID,
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		Gender: u.Gender, Birthday: u.Birthday, Announcement: u.SpaceAnnouncement,
	}, nil
}

// UpdateProfile updates a user's profile fields (nickname, sign, gender, birthday).
func (s *UserService) UpdateProfile(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	allowed := map[string]bool{"nickname": true, "sign": true, "gender": true, "birthday": true}
	filtered := make(map[string]interface{})
	for k, v := range updates {
		if allowed[k] { filtered[k] = v }
	}
	if len(filtered) == 0 { return nil }
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(filtered).Error
}

// UpdateUsername updates the username (validated by caller).
func (s *UserService) UpdateUsername(ctx context.Context, userID uint64, newName string) error {
	var count int64
	s.db.WithContext(ctx).Model(&model.User{}).Where("username = ? AND id != ?", newName, userID).Count(&count)
	if count > 0 { return ErrParamError }
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("username", newName).Error
}

// UpdatePassword updates the password hash.
func (s *UserService) UpdatePassword(ctx context.Context, userID uint64, newHash string) error {
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("password_hash", newHash).Error
}

// GetUserPublic returns a user's public profile.
func (s *UserService) GetUserPublic(ctx context.Context, userID uint64) (*UserProfile, error) {
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &UserProfile{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname,
		AvatarURL: u.AvatarURL, Sign: u.Sign, CakeID: u.CakeID,
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		Gender: u.Gender, Birthday: u.Birthday, Announcement: u.SpaceAnnouncement,
	}, nil
}
// UpdateAnnouncement updates the user's space announcement.
func (s *UserService) UpdateAnnouncement(ctx context.Context, userID uint64, announcement string) error {
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).
		Update("space_announcement", announcement).Error
}
