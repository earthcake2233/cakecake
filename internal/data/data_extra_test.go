package data

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func extDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func extDBWithMigrate(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db := extDB(t)
	require.NoError(t, db.AutoMigrate(models...))
	return db
}

func Test_RegisteredMigrationsAll(t *testing.T) {
	migs := RegisteredMigrations()
	require.Greater(t, len(migs), 0, "should have migrations")
	for i, m := range migs {
		require.Equal(t, i+1, m.Version, "migration %d version", i)
		require.NotEmpty(t, m.Name, "migration %d name", i)
		require.NotNil(t, m.Func, "migration %d func", i)
	}
}

func Test_AutoMigrateAll(t *testing.T) {
	err := AutoMigrateAll(extDB(t), zap.NewNop())
	require.NoError(t, err)
}

func Test_ResyncCuratedCountsEmpty(t *testing.T) {
	db := extDBWithMigrate(t, &model.Video{}, &model.Article{}, &model.Comment{}, &model.ArticleComment{})
	require.NoError(t, resyncCuratedVideoCommentCounts(db, zap.NewNop()))
	require.NoError(t, resyncCuratedArticleCommentCounts(db, zap.NewNop()))
}

func Test_BackfillUserCakeIDs(t *testing.T) {
	db := extDBWithMigrate(t, &model.User{})
	backfillUserCakeIDs(db, zap.NewNop())
}

func Test_IsIgnorableAddColumnErr(t *testing.T) {
	require.False(t, isIgnorableAddColumnErr(nil), "nil should not be ignorable")
	err := errors.New("some error")
	require.False(t, isIgnorableAddColumnErr(err), "random error should not be ignorable")
	err2 := errors.New("duplicate column name")
	require.True(t, isIgnorableAddColumnErr(err2), "duplicate column error should be ignorable")
}
