package service

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/username"
)

// AuthService handles authentication business logic.
type AuthService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
	jwt *jwttoken.Manager
	cfg AuthConfig
}

// AuthConfig carries config values needed by AuthService.
type AuthConfig struct {
	AgentBotUsername string
}

// NewAuthService creates an AuthService.
func NewAuthService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, jwt *jwttoken.Manager, cfg AuthConfig) *AuthService {
	return &AuthService{db: db, rdb: rdb, log: log, jwt: jwt, cfg: cfg}
}

// UserBrief carries minimal user info for the handler.
type UserBrief struct {
	ID       uint64
	Username string
}

// LookupUser returns a user by username (for pre-authentication checks in handler).
func (s *AuthService) LookupUser(ctx context.Context, reqUsername string) (*UserBrief, error) {
	var u model.User
	if err := s.db.WithContext(ctx).Where("username = ?", strings.TrimSpace(reqUsername)).First(&u).Error; err != nil {
		return nil, err
	}
	return &UserBrief{ID: u.ID, Username: u.Username}, nil
}

// RegisterResult is returned after successful registration.
type RegisterResult struct {
	UserID   uint64
	Username string
	CakeID   string
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, reqUsername, reqPassword string) (*RegisterResult, error) {
	uname := strings.TrimSpace(reqUsername)
	if !username.Valid(uname) || len(reqPassword) < 8 {
		return nil, ErrParamError
	}
	lowName := strings.ToLower(uname)
	if strings.EqualFold(uname, strings.TrimSpace(s.cfg.AgentBotUsername)) ||
		lowName == "minibili_ai" || strings.HasPrefix(lowName, "ai_") {
		return nil, ErrParamError
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(reqPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrInternalError
	}
	u := model.User{Username: uname, PasswordHash: string(hash)}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		cid := model.FormatCakeID(u.ID)
		return tx.Model(&u).Update("cake_id", cid).Error
	})
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			return nil, &SvcError{Code: 40006, Msg: "username exists"}
		}
		s.log.Error("register insert", zap.Error(err))
		return nil, ErrInternalError
	}
	_ = s.db.WithContext(ctx).First(&u, u.ID)
	return &RegisterResult{UserID: u.ID, Username: u.Username, CakeID: u.CakeID}, nil
}

// AuthenticateResult is returned after successful authentication.
type AuthenticateResult struct {
	UserID       uint64
	Username     string
	AccessToken  string
	RefreshToken string
}

// Authenticate verifies password and issues JWT pair for an existing user.
func (s *AuthService) Authenticate(ctx context.Context, userID uint64, password string) (*AuthenticateResult, error) {
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, &SvcError{Code: 40100, Msg: "user not found"}
	}
	if model.IsUserAnonymized(&u) {
		return nil, &SvcError{Code: 40100, Msg: "invalid credentials"}
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, &SvcError{Code: 40100, Msg: "invalid credentials"}
	}
	access, refresh, _, err := s.jwt.IssuePair(u.ID)
	if err != nil {
		return nil, ErrInternalError
	}
	_ = dailyreward.MarkLogin(s.db, u.ID)
	s.log.Info("user login success", zap.String("username", u.Username))
	return &AuthenticateResult{
		UserID:       u.ID,
		Username:     u.Username,
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

// RefreshResult is returned after successful token refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

// Refresh rotates a refresh token, invalidating the old one.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	uid, tokenID, err := s.jwt.ParseRefresh(strings.TrimSpace(refreshToken))
	if err != nil {
		return nil, ErrUnauthorized
	}
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, uid).Error; err == nil && model.IsUserAnonymized(&u) {
		return nil, &SvcError{Code: 40302, Msg: "account closed"}
	}
	if s.rdb.Exists(ctx, data.RefreshInvalidKey(tokenID)).Val() == 1 {
		return nil, ErrUnauthorized
	}
	_ = s.rdb.Set(ctx, data.RefreshInvalidKey(tokenID), "1", data.RefreshInvalidTTL).Err()
	access, refresh, _, err := s.jwt.IssuePair(uid)
	if err != nil {
		return nil, ErrInternalError
	}
	return &RefreshResult{AccessToken: access, RefreshToken: refresh}, nil
}
