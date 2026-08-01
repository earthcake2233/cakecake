package handler

import (
	"cakecake/internal/model/user"
	"cakecake/internal/pkg/useravatar"
)

func avatarURLForAPI(u *user.User) string {
	return useravatar.PublicURL(u)
}
