package jwttoken

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// IssueAdminPair returns JWT pair for an operations admin (principal=admin).
func (m *Manager) IssueAdminPair(adminID uint64) (access string, refresh string, tokenID string, err error) {
	tokenID = uuid.NewString()
	now := time.Now()
	accessClaims := Claims{
		Principal: PrincipalAdmin,
		UserID:    adminID,
		TokenID:   tokenID,
		Type:      claimTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshClaims := Claims{
		Principal: PrincipalAdmin,
		UserID:    adminID,
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

// ParseAdminAccess validates an admin access token.
func (m *Manager) ParseAdminAccess(token string) (adminID uint64, tokenID string, err error) {
	claims, err := m.parse(token)
	if err != nil {
		return 0, "", err
	}
	if claims.Principal != PrincipalAdmin || claims.Type != claimTypeAccess {
		return 0, "", fmt.Errorf("not admin access token")
	}
	return claims.UserID, claims.TokenID, nil
}

// ParseAdminRefresh validates an admin refresh token.
func (m *Manager) ParseAdminRefresh(token string) (adminID uint64, tokenID string, err error) {
	claims, err := m.parse(token)
	if err != nil {
		return 0, "", err
	}
	if claims.Principal != PrincipalAdmin || claims.Type != claimTypeRefresh {
		return 0, "", fmt.Errorf("not admin refresh token")
	}
	return claims.UserID, claims.TokenID, nil
}
