package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"minibili/internal/pkg/jwttoken"
)

func newTestJWTManger(t *testing.T) *jwttoken.Manager {
	t.Helper()
	m, err := jwttoken.NewManager("test-secret-key-32-chars-long!!")
	require.NoError(t, err)
	return m
}

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", JWTAuth(j), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidScheme(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", JWTAuth(j), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", JWTAuth(j), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ValidToken(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", JWTAuth(j), func(c *gin.Context) {
		uid, ok := UserID(c)
		require.True(t, ok)
		require.Equal(t, uint64(42), uid)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	access, _, _, err := j.IssuePair(42)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_AdminTokenOnUserRoute(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", JWTAuth(j), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	access, _, _, err := j.IssueAdminPair(1)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalJWT_NoHeader(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", OptionalJWT(j), func(c *gin.Context) {
		_, ok := UserID(c)
		require.False(t, ok)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalJWT_ValidToken(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", OptionalJWT(j), func(c *gin.Context) {
		uid, ok := UserID(c)
		require.True(t, ok)
		require.Equal(t, uint64(42), uid)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	access, _, _, err := j.IssuePair(42)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalJWT_InvalidToken(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/test", OptionalJWT(j), func(c *gin.Context) {
		_, ok := UserID(c)
		require.False(t, ok)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestUserID_NoContext(t *testing.T) {
	c := &gin.Context{}
	_, ok := UserID(c)
	require.False(t, ok)
}

func TestUserID_WrongType(t *testing.T) {
	c := &gin.Context{}
	c.Set(CtxUserIDKey, "not-a-uint64")
	_, ok := UserID(c)
	require.False(t, ok)
}

func TestAdminJWTAuth_MissingHeader(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/admin", AdminJWTAuth(j), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminJWTAuth_ValidToken(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/admin", AdminJWTAuth(j), func(c *gin.Context) {
		aid, ok := AdminID(c)
		require.True(t, ok)
		require.Equal(t, uint64(1), aid)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	access, _, _, err := j.IssueAdminPair(1)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAdminJWTAuth_UserTokenOnAdminRoute(t *testing.T) {
	r := setupGin()
	j := newTestJWTManger(t)
	r.GET("/admin", AdminJWTAuth(j), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	access, _, _, err := j.IssuePair(42)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminID_NoContext(t *testing.T) {
	c := &gin.Context{}
	_, ok := AdminID(c)
	require.False(t, ok)
}

func TestAdminID_WrongType(t *testing.T) {
	c := &gin.Context{}
	c.Set(CtxAdminIDKey, "not-a-uint64")
	_, ok := AdminID(c)
	require.False(t, ok)
}
