package user

import (
	"cakecake/internal/model/user"
	"cakecake/internal/pkg/dbtx"
	"cakecake/internal/service"
	"context"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"time"
)

// UserService handles user profile business logic.
type UserService struct {
	users service.UserProvider
	log   *zap.Logger
}

// NewUserService creates a UserService with storage and logger.
func NewUserService(db *gorm.DB, log *zap.Logger) *UserService {
	return &UserService{users: service.NewUserProvider(db), log: log}
}

// UserProfile is the full profile DTO returned for the caller's own account.
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
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
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
	return s.users.UpdateUserFields(ctx, userID, filtered)
}

// UpdateUsername updates the username (validated by caller).
func (s *UserService) UpdateUsername(ctx context.Context, userID uint64, newName string) error {
	if s.users.UsernameTaken(ctx, newName, userID) {
		return service.ErrParamError
	}
	return s.users.UpdateUsername(ctx, userID, newName)
}

// UpdatePassword updates the password hash.
func (s *UserService) UpdatePassword(ctx context.Context, userID uint64, newHash string) error {
	return s.users.UpdatePasswordHash(ctx, userID, newHash)
}

// GetUserPublic returns a user's public profile.

// BatchGetUsers returns a map of userID->User for the given IDs.
func (s *UserService) BatchGetUsers(ctx context.Context, userIDs []uint64) map[uint64]*user.User {
	out, _ := s.users.BatchGetUsersByIDs(ctx, userIDs)
	return out
}

// GetUserPublic returns the public profile view of a user.
func (s *UserService) GetUserPublic(ctx context.Context, userID uint64) (*UserProfile, error) {
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
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
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return "用户", "", err
	}
	name = user.DisplayUsername(u)
	if nick := strings.TrimSpace(u.Nickname); nick != "" {
		name = nick
	}
	return name, u.AvatarURL, nil
}

// UpdateAnnouncement updates the user's space announcement.
func (s *UserService) UpdateAnnouncement(ctx context.Context, userID uint64, announcement string) error {
	return s.users.UpdateAnnouncement(ctx, userID, announcement)
}

// GetUserByID returns a raw user model for internal use.
func (s *UserService) GetUserByID(ctx context.Context, userID uint64) (*user.User, error) {
	return s.users.GetUserByID(ctx, userID)
}

// RequestDeletion sets the deletion_requested_at and deletion_effective_at fields.
func (s *UserService) RequestDeletion(ctx context.Context, userID uint64, requestedAt, effectiveAt time.Time) error {
	return s.users.SetDeletion(ctx, userID, &requestedAt, &effectiveAt)
}

// RevokeDeletion clears the deletion_requested_at and deletion_effective_at fields.
func (s *UserService) RevokeDeletion(ctx context.Context, userID uint64) error {
	return s.users.SetDeletion(ctx, userID, nil, nil)
}

// GetPrivacySettings returns the privacy settings for a user.
func (s *UserService) GetPrivacySettings(ctx context.Context, userID uint64) (*user.User, error) {
	return s.users.GetUserByID(ctx, userID)
}

// UpdatePrivacySettings updates space privacy toggles.
func (s *UserService) UpdatePrivacySettings(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	return s.users.UpdateUserFields(ctx, userID, updates)
}

// GetPasswordHash returns the password hash for a user.
func (s *UserService) GetPasswordHash(ctx context.Context, userID uint64) (string, error) {
	return s.users.GetPasswordHash(ctx, userID)
}

// UpdateAvatar updates the avatar_url for a user.
func (s *UserService) UpdateAvatar(ctx context.Context, userID uint64, objectKey string) error {
	return s.users.UpdateAvatar(ctx, userID, objectKey)
}

// EnsureCakeID sets a CakeID for the user if one is not already set.
func (s *UserService) EnsureCakeID(ctx context.Context, u *user.User) error {
	return s.users.EnsureCakeID(ctx, u)
}

// ListCoinLedger returns paginated coin change history for a user.
func (s *UserService) ListCoinLedger(ctx context.Context, userID uint64, since time.Time, limit, offset int) (total int64, rows []user.CoinLedger, err error) {
	return s.users.ListCoinLedger(ctx, userID, since, limit, offset)
}

// FinalizeDeletion performs the final account anonymization within a transaction.
func (s *UserService) FinalizeDeletion(ctx context.Context, uid uint64, fn func(tx dbtx.Tx) error) error {
	return s.users.WithTx(ctx, fn)
}
