package user

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/user"
	"cakecake/internal/service"
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"cakecake/internal/data"
	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/pkg/username"
	"time"
)

// AdminProvider is the admin storage boundary (Phase 1: *gorm.DB impl; Phase 2+: gRPC client).
type AdminProvider interface {
	FindAdminByUsername(ctx context.Context, username string) (*admin.Admin, error)
	GetAdminByID(ctx context.Context, id uint64) (*admin.Admin, error)
	UpdateAdminLoginTime(ctx context.Context, id uint64, t time.Time) error
}

type AdminProviderImpl struct {
	db *gorm.DB
}

func NewAdminProvider(db *gorm.DB) *AdminProviderImpl {
	return &AdminProviderImpl{db: db}
}

func (p *AdminProviderImpl) FindAdminByUsername(ctx context.Context, username string) (*admin.Admin, error) {
	var adm admin.Admin
	if err := p.db.WithContext(ctx).Where("username = ?", username).First(&adm).Error; err != nil {
		return nil, err
	}
	return &adm, nil
}

func (p *AdminProviderImpl) GetAdminByID(ctx context.Context, id uint64) (*admin.Admin, error) {
	var adm admin.Admin
	if err := p.db.WithContext(ctx).First(&adm, id).Error; err != nil {
		return nil, err
	}
	return &adm, nil
}

func (p *AdminProviderImpl) UpdateAdminLoginTime(ctx context.Context, id uint64, t time.Time) error {
	return p.db.WithContext(ctx).Model(&admin.Admin{}).Where("id = ?", id).Update("last_login_at", t).Error
}

// AuthService handles authentication business logic.
type AuthService struct {
	users  service.UserProvider
	admins AdminProvider
	rdb    *redis.Client
	log    *zap.Logger
	jwt    *jwttoken.Manager
	cfg    AuthConfig
}

// AdminRefreshTokenInvalid reports whether an admin refresh token was already invalidated.
func (s *AuthService) AdminRefreshTokenInvalid(ctx context.Context, tokenID string) bool {
	return s.rdb.Exists(ctx, data.AdminRefreshInvalidKey(tokenID)).Val() == 1
}

// MarkAdminRefreshTokenInvalid records an admin refresh token as used/invalid.
func (s *AuthService) MarkAdminRefreshTokenInvalid(ctx context.Context, tokenID string) error {
	return s.rdb.Set(ctx, data.AdminRefreshInvalidKey(tokenID), "1", data.RefreshInvalidTTL).Err()
}

// AuthConfig carries config values needed by AuthService.
type AuthConfig struct {
	AgentBotUsername string
}

// NewAuthService creates an AuthService.
func NewAuthService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, jwt *jwttoken.Manager, cfg AuthConfig) *AuthService {
	return &AuthService{users: service.NewUserProvider(db), admins: NewAdminProvider(db), rdb: rdb, log: log, jwt: jwt, cfg: cfg}
}

// UserBrief carries minimal user info for the handler.
type UserBrief struct {
	ID       uint64
	Username string
}

// LookupUser returns a user by username (for pre-authentication checks in handler).
func (s *AuthService) LookupUser(ctx context.Context, reqUsername string) (*UserBrief, error) {
	u, err := s.users.GetUserByUsername(ctx, strings.TrimSpace(reqUsername))
	if err != nil {
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
		return nil, service.ErrParamError
	}
	lowName := strings.ToLower(uname)
	if strings.EqualFold(uname, strings.TrimSpace(s.cfg.AgentBotUsername)) ||
		lowName == "minibili_ai" || strings.HasPrefix(lowName, "ai_") {
		return nil, service.ErrParamError
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(reqPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, service.ErrInternalError
	}
	u := user.User{Username: uname, PasswordHash: string(hash)}
	err = s.users.CreateUserWithCakeID(ctx, &u)
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			return nil, &service.SvcError{Code: 40006, Msg: "username exists"}
		}
		s.log.Error("register insert", zap.Error(err))
		return nil, service.ErrInternalError
	}
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
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, &service.SvcError{Code: 40100, Msg: "user not found"}
	}
	if user.IsUserAnonymized(u) {
		return nil, &service.SvcError{Code: 40100, Msg: "invalid credentials"}
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, &service.SvcError{Code: 40100, Msg: "invalid credentials"}
	}
	access, refresh, _, err := s.jwt.IssuePair(u.ID)
	if err != nil {
		return nil, service.ErrInternalError
	}
	_ = s.users.MarkLogin(ctx, u.ID)
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
		return nil, service.ErrUnauthorized
	}
	u, err := s.users.GetUserByID(ctx, uid)
	if err == nil && user.IsUserAnonymized(u) {
		return nil, &service.SvcError{Code: 40302, Msg: "account closed"}
	}
	if s.rdb.Exists(ctx, data.RefreshInvalidKey(tokenID)).Val() == 1 {
		return nil, service.ErrUnauthorized
	}
	_ = s.rdb.Set(ctx, data.RefreshInvalidKey(tokenID), "1", data.RefreshInvalidTTL).Err()
	access, refresh, _, err := s.jwt.IssuePair(uid)
	if err != nil {
		return nil, service.ErrInternalError
	}
	return &RefreshResult{AccessToken: access, RefreshToken: refresh}, nil
}

// FindAdminByUsername looks up an admin by username.
func (s *AuthService) FindAdminByUsername(ctx context.Context, username string) (*admin.Admin, error) {
	return s.admins.FindAdminByUsername(ctx, username)
}

// GetAdminByID looks up an admin by ID.
func (s *AuthService) GetAdminByID(ctx context.Context, id uint64) (*admin.Admin, error) {
	return s.admins.GetAdminByID(ctx, id)
}

// UpdateAdminLoginTime records the admin login timestamp.
func (s *AuthService) UpdateAdminLoginTime(ctx context.Context, id uint64, t time.Time) error {
	return s.admins.UpdateAdminLoginTime(ctx, id, t)
}
