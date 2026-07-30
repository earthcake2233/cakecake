package service

import (
	"minibili/internal/model/user"
	"context"

	"gorm.io/gorm"

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
	return toUserInfo(&u), nil
}

func (p *UserProviderImpl) GetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]UserInfo, error) {
	if len(ids) == 0 { return nil, nil }
	var users []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]UserInfo, len(users))
	for i := range users {
		result[users[i].ID] = toUserInfo(&users[i])
	}
	return result, nil
}

func (p *UserProviderImpl) BatchCurrentLevels(ctx context.Context, ids []uint64) (map[uint64]int, error) {
	if len(ids) == 0 { return nil, nil }
	var users []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	th := []uint64{0, 20, 150, 450, 1080, 2880}
	result := make(map[uint64]int, len(users))
	for _, u := range users {
		lv := 1
		for j := len(th) - 1; j >= 0; j-- {
			if u.Experience >= th[j] { lv = j + 1; break }
		}
		result[u.ID] = lv
	}
	return result, nil
}

func toUserInfo(u *user.User) UserInfo {
	name := u.Username
	if u.Nickname != "" { name = u.Nickname }
	return UserInfo{
		ID: u.ID, Username: u.Username, Nickname: name, AvatarURL: u.AvatarURL,
		CoinBalanceTenths: u.CoinBalanceTenths,
	}
}

func (p *UserProviderImpl) DecrementCoins(ctx context.Context, userID uint64, amount int) error {
    return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ? AND coins >= ?", userID, amount).
        UpdateColumn("coins", gorm.Expr("coins - ?", amount)).Error
}
