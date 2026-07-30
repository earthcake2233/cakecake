package handler

import (
	"minibili/internal/model/user"
	"minibili/internal/pkg/useravatar"
)

func avatarURLForAPI(u *user.User) string {
	return useravatar.PublicURL(u)
}
