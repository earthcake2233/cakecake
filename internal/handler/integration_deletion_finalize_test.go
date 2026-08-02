//go:build integration

package handler

import (
	"cakecake/internal/model/notification"
	"cakecake/internal/model/user"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAccountDeletion_Finalize(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	eff := time.Now().Add(-time.Hour)
	require.NoError(t, api.DB.Create(&user.User{
		ID: 1, Username: "doomed", PasswordHash: string(hash), CoinBalanceTenths: 230,
		DeletionRequestedAt: &eff, DeletionEffectiveAt: &eff,
	}).Error)
	require.NoError(t, api.DB.Create(&notification.Notification{RecipientID: 1, Type: "reply"}).Error)

	// Any deletion-adjacent call triggers finalization once the cooling period has passed.
	w := doJSON(r, "POST", "/api/v1/users/me/deletion/request", token, map[string]interface{}{"password": "password123"})
	// After finalization the account is closed, so the request itself returns 403.
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	var u user.User
	require.NoError(t, api.DB.First(&u, 1).Error)
	require.NotNil(t, u.AnonymizedAt)
	require.Contains(t, u.Username, "d")
	require.Empty(t, u.Nickname)

	// Notifications for the anonymized user are purged.
	var n int64
	require.NoError(t, api.DB.Model(&notification.Notification{}).Count(&n).Error)
	require.Zero(t, n)
}
