package handler

import (
	"minibili/internal/model"
	"minibili/internal/pkg/useravatar"
)

func avatarURLForAPI(u *model.User) string {
	return useravatar.PublicURL(u)
}
