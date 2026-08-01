package service

import (
	"cakecake/internal/model/user"
	"context"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"time"
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
	ID                       uint64 `json:"id"`
	Username                 string `json:"username"`
	Nickname                 string `json:"nickname"`
	AvatarURL                string `json:"avatar_url"`
	Sign                     string `json:"sign"`
	CakeID                   string `json:"cake_id"`
	CreatedAt                string `json:"created_at"`
	Gender                   string `json:"gender"`
	Birthday                 string `json:"birthday"`
	Announcement             string `json:"announcement"`
	PrivacyPublicFavorites   bool   `json:"privacy_public_favorites"`
	PrivacyPublicRecentCoins bool   `json:"privacy_public_recent_coins"`
	PrivacyPublicFollowing   bool   `json:"privacy_public_following"`
	PrivacyPublicFans        bool   `json:"privacy_public_fans"`
}

// GetMe returns the current user's profile.
func (s *UserService) GetMe(ctx context.Context, userID uint64) (*UserProfile, error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &UserProfile{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname,
		AvatarURL: u.AvatarURL, Sign: u.Sign, CakeID: u.CakeID,
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		Gender:    u.Gender, Birthday: u.Birthday, Announcement: u.SpaceAnnouncement,
		PrivacyPublicFavorites: u.PrivacyPublicFavorites, PrivacyPublicRecentCoins: u.PrivacyPublicRecentCoins, PrivacyPublicFollowing: u.PrivacyPublicFollowing, PrivacyPublicFans: u.PrivacyPublicFans,
	}, nil
}

// UpdateProfile updates a user's profile fields (nickname, sign, gender, birthday).
func (s *UserService) UpdateProfile(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	allowed := map[string]bool{"nickname": true, "sign": true, "gender": true, "birthday": true}
	filtered := make(map[string]interface{})
	for k, v := range updates {
		if allowed[k] {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Updates(filtered).Error
}

// UpdateUsername updates the username (validated by caller).
func (s *UserService) UpdateUsername(ctx context.Context, userID uint64, newName string) error {
	var count int64
	s.db.WithContext(ctx).Model(&user.User{}).Where("username = ? AND id != ?", newName, userID).Count(&count)
	if count > 0 {
		return ErrParamError
	}
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("username", newName).Error
}

// UpdatePassword updates the password hash.
func (s *UserService) UpdatePassword(ctx context.Context, userID uint64, newHash string) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("password_hash", newHash).Error
}

// GetUserPublic returns a user's public profile.

// BatchGetUsers returns a map of userID->User for the given IDs.
func (s *UserService) BatchGetUsers(ctx context.Context, userIDs []uint64) map[uint64]*user.User {
	out := make(map[uint64]*user.User, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	var users []user.User
	s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users)
	for i := range users {
		u := users[i]
		out[u.ID] = &u
	}
	return out
}

func (s *UserService) GetUserPublic(ctx context.Context, userID uint64) (*UserProfile, error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &UserProfile{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname,
		AvatarURL: u.AvatarURL, Sign: u.Sign, CakeID: u.CakeID,
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		Gender:    u.Gender, Birthday: u.Birthday, Announcement: u.SpaceAnnouncement,
		PrivacyPublicFavorites: u.PrivacyPublicFavorites, PrivacyPublicRecentCoins: u.PrivacyPublicRecentCoins, PrivacyPublicFollowing: u.PrivacyPublicFollowing, PrivacyPublicFans: u.PrivacyPublicFans,
	}, nil
}

// GetUserBrief returns a user's display name and avatar URL.
func (s *UserService) GetUserBrief(ctx context.Context, userID uint64) (name, avatar string, err error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return "用户", "", err
	}
	name = user.DisplayUsername(&u)
	if nick := strings.TrimSpace(u.Nickname); nick != "" {
		name = nick
	}
	return name, u.AvatarURL, nil
}

// UpdateAnnouncement updates the user's space announcement.
func (s *UserService) UpdateAnnouncement(ctx context.Context, userID uint64, announcement string) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).
		Update("space_announcement", announcement).Error
}

// GetUserByID returns a raw user model for internal use.
func (s *UserService) GetUserByID(ctx context.Context, userID uint64) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// RequestDeletion sets the deletion_requested_at and deletion_effective_at fields.
func (s *UserService) RequestDeletion(ctx context.Context, userID uint64, requestedAt, effectiveAt time.Time) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"deletion_requested_at": requestedAt,
		"deletion_effective_at": effectiveAt,
	}).Error
}

// RevokeDeletion clears the deletion_requested_at and deletion_effective_at fields.
func (s *UserService) RevokeDeletion(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"deletion_requested_at": nil,
		"deletion_effective_at": nil,
	}).Error
}

// GetPrivacySettings returns the privacy settings for a user.
func (s *UserService) GetPrivacySettings(ctx context.Context, userID uint64) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePrivacySettings updates space privacy toggles.
func (s *UserService) UpdatePrivacySettings(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Updates(updates).Error
}

// GetPasswordHash returns the password hash for a user.
func (s *UserService) GetPasswordHash(ctx context.Context, userID uint64) (string, error) {
	var result struct{ PasswordHash string }
	if err := s.db.WithContext(ctx).Raw("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&result).Error; err != nil {
		return "", err
	}
	return result.PasswordHash, nil
}

// UpdateAvatar updates the avatar_url for a user.
func (s *UserService) UpdateAvatar(ctx context.Context, userID uint64, objectKey string) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("avatar_url", objectKey).Error
}

// EnsureCakeID sets a CakeID for the user if one is not already set.
func (s *UserService) EnsureCakeID(ctx context.Context, u *user.User) error {
	if u.CakeID != "" {
		return nil
	}
	cakeID := user.FormatCakeID(u.ID)
	if err := s.db.WithContext(ctx).Model(u).Update("cake_id", cakeID).Error; err != nil {
		return err
	}
	u.CakeID = cakeID
	return nil
}

// ListCoinLedger returns paginated coin change history for a user.
func (s *UserService) ListCoinLedger(ctx context.Context, userID uint64, since time.Time, limit, offset int) (total int64, rows []user.CoinLedger, err error) {
	q := s.db.WithContext(ctx).Model(&user.CoinLedger{}).Where("user_id = ? AND created_at >= ?", userID, since)
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

// FinalizeDeletion performs the final account anonymization within a transaction.
func (s *UserService) FinalizeDeletion(ctx context.Context, uid uint64, fn func(tx *gorm.DB) error) error {
	tx := s.db.WithContext(ctx).Begin()
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
