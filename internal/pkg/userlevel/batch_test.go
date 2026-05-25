package userlevel

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func TestBatchCurrentLevels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	u1 := model.User{Username: "a", PasswordHash: "x", Experience: 0}
	u2 := model.User{Username: "b", PasswordHash: "x", Experience: 2880}
	if err := db.Create(&u1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&u2).Error; err != nil {
		t.Fatal(err)
	}
	got := BatchCurrentLevels(db, []uint64{u1.ID, u2.ID, u2.ID})
	if got[u1.ID] != 1 || got[u2.ID] != 6 {
		t.Fatalf("got %+v, want u1=1 u2=6", got)
	}
}
