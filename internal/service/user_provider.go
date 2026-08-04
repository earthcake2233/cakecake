package service

import (
	"cakecake/internal/model/user"
	"cakecake/internal/pkg/dailyreward"
	"context"
	"time"

	"gorm.io/gorm"

	"cakecake/internal/pkg/usercoin"
)

// UserProviderImpl implements UserProvider using *gorm.DB (Phase 1 monolith).
type UserProviderImpl struct {
	db *gorm.DB
}

func NewUserProvider(db *gorm.DB) *UserProviderImpl {
	return &UserProviderImpl{db: db}
}

func (p *UserProviderImpl) GetUser(ctx context.Context, id uint64) (UserInfo, error) {
	var u user.User
	if err := p.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return UserInfo{}, err
	}
	return ToUserInfo(&u), nil
}

func (p *UserProviderImpl) GetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]UserInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]UserInfo, len(users))
	for i := range users {
		result[users[i].ID] = ToUserInfo(&users[i])
	}
	return result, nil
}

func (p *UserProviderImpl) BatchCurrentLevels(ctx context.Context, ids []uint64) (map[uint64]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	th := []uint64{0, 20, 150, 450, 1080, 2880}
	result := make(map[uint64]int, len(users))
	for _, u := range users {
		lv := 1
		for j := len(th) - 1; j >= 0; j-- {
			if u.Experience >= th[j] {
				lv = j + 1
				break
			}
		}
		result[u.ID] = lv
	}
	return result, nil
}

func ToUserInfo(u *user.User) UserInfo {
	name := u.Username
	avatar := u.AvatarURL
	if user.IsUserAnonymized(u) {
		name = "已注销用户"
		avatar = ""
	} else if u.Nickname != "" {
		name = u.Nickname
	}
	return UserInfo{
		ID: u.ID, Username: u.Username, Nickname: name, AvatarURL: avatar,
		CoinBalanceTenths: u.CoinBalanceTenths,
		AnonymizedAt:      u.AnonymizedAt,
	}
}

func (p *UserProviderImpl) DecrementCoins(ctx context.Context, userID uint64, amount int) error {
	cost := usercoin.CostTenths(amount)
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ? AND coin_balance_tenths >= ?", userID, cost).
		UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths - ?", cost)).Error
}

func (p *UserProviderImpl) GetUserByID(ctx context.Context, id uint64) (*user.User, error) {
	var u user.User
	if err := p.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (p *UserProviderImpl) GetUserByUsername(ctx context.Context, name string) (*user.User, error) {
	var u user.User
	if err := p.db.WithContext(ctx).Where("username = ?", name).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (p *UserProviderImpl) BatchGetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]*user.User, error) {
	out := make(map[uint64]*user.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		u := users[i]
		out[u.ID] = &u
	}
	return out, nil
}

func (p *UserProviderImpl) UsernameTaken(ctx context.Context, name string, excludeID uint64) bool {
	var count int64
	p.db.WithContext(ctx).Model(&user.User{}).Where("username = ? AND id != ?", name, excludeID).Count(&count)
	return count > 0
}

func (p *UserProviderImpl) UpdateUsername(ctx context.Context, id uint64, name string) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Update("username", name).Error
}

func (p *UserProviderImpl) UpdateUserFields(ctx context.Context, id uint64, fields map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Updates(fields).Error
}

func (p *UserProviderImpl) UpdatePasswordHash(ctx context.Context, id uint64, hash string) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}

func (p *UserProviderImpl) UpdateAnnouncement(ctx context.Context, id uint64, announcement string) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).
		Update("space_announcement", announcement).Error
}

func (p *UserProviderImpl) UpdateAvatar(ctx context.Context, id uint64, objectKey string) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Update("avatar_url", objectKey).Error
}

func (p *UserProviderImpl) SetDeletion(ctx context.Context, id uint64, requestedAt, effectiveAt *time.Time) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deletion_requested_at": requestedAt,
		"deletion_effective_at": effectiveAt,
	}).Error
}

func (p *UserProviderImpl) GetPasswordHash(ctx context.Context, id uint64) (string, error) {
	var result struct{ PasswordHash string }
	if err := p.db.WithContext(ctx).Raw("SELECT password_hash FROM users WHERE id = ?", id).Scan(&result).Error; err != nil {
		return "", err
	}
	return result.PasswordHash, nil
}

func (p *UserProviderImpl) EnsureCakeID(ctx context.Context, u *user.User) error {
	if u.CakeID != "" {
		return nil
	}
	cakeID := user.FormatCakeID(u.ID)
	if err := p.db.WithContext(ctx).Model(u).Update("cake_id", cakeID).Error; err != nil {
		return err
	}
	u.CakeID = cakeID
	return nil
}

func (p *UserProviderImpl) ListCoinLedger(ctx context.Context, userID uint64, since time.Time, limit, offset int) (total int64, rows []user.CoinLedger, err error) {
	q := p.db.WithContext(ctx).Model(&user.CoinLedger{}).Where("user_id = ? AND created_at >= ?", userID, since)
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func (p *UserProviderImpl) CreateUserWithCakeID(ctx context.Context, u *user.User) error {
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		cid := user.FormatCakeID(u.ID)
		return tx.Model(u).Update("cake_id", cid).Error
	})
	if err != nil {
		return err
	}
	return p.db.WithContext(ctx).First(u, u.ID).Error
}

func (p *UserProviderImpl) MarkLogin(ctx context.Context, userID uint64) error {
	return dailyreward.MarkLogin(p.db, userID)
}

func (p *UserProviderImpl) WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return p.db.WithContext(ctx).Transaction(fn)
}
