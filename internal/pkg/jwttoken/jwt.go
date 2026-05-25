package jwttoken

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	claimTypeAccess  = "access"
	claimTypeRefresh = "refresh"
	PrincipalUser    = "user"
	PrincipalAdmin   = "admin"
)

// Claims is the JWT payload (Skill S-009).
type Claims struct {
	Principal string `json:"principal,omitempty"` // user | admin; empty legacy tokens = user
	UserID    uint64 `json:"user_id"`
	TokenID   string `json:"token_id"`
	Type      string `json:"type"`
	jwt.RegisteredClaims
}

// Manager issues and parses JWTs.
type Manager struct {
	secret []byte
}

func NewManager(secret string) (*Manager, error) {
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is empty")
	}
	return &Manager{secret: []byte(secret)}, nil
}

// IssuePair returns access token, refresh token, and token id (Skill S-009).
func (m *Manager) IssuePair(userID uint64) (access string, refresh string, tokenID string, err error) {
	tokenID = uuid.NewString()
	now := time.Now()
	accessClaims := Claims{
		Principal: PrincipalUser,
		UserID:    userID,
		TokenID:   tokenID,
		Type:      claimTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshClaims := Claims{
		Principal: PrincipalUser,
		UserID:    userID,
		TokenID:   tokenID,
		Type:      claimTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	access, err = at.SignedString(m.secret)
	if err != nil {
		return "", "", "", err
	}
	refresh, err = rt.SignedString(m.secret)
	if err != nil {
		return "", "", "", err
	}
	return access, refresh, tokenID, nil
}

// ParseAccess validates an access token and returns user id and token id.
func (m *Manager) ParseAccess(token string) (userID uint64, tokenID string, err error) {
	claims, err := m.parse(token)
	if err != nil {
		return 0, "", err
	}
	if claims.Type != claimTypeAccess {
		return 0, "", fmt.Errorf("not access token")
	}
	if claims.Principal != "" && claims.Principal != PrincipalUser {
		return 0, "", fmt.Errorf("not user token")
	}
	return claims.UserID, claims.TokenID, nil
}

// ParseRefresh validates a refresh token.
func (m *Manager) ParseRefresh(token string) (userID uint64, tokenID string, err error) {
	claims, err := m.parse(token)
	if err != nil {
		return 0, "", err
	}
	if claims.Type != claimTypeRefresh {
		return 0, "", fmt.Errorf("not refresh token")
	}
	if claims.Principal != "" && claims.Principal != PrincipalUser {
		return 0, "", fmt.Errorf("not user token")
	}
	return claims.UserID, claims.TokenID, nil
}

func (m *Manager) parse(token string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
