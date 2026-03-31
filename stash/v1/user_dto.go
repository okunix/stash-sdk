package stash

import (
	"time"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Locked    bool       `json:"locked"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiredAt *time.Time `json:"expired_at,omitempty"`
}

type ChangePasswordRequest struct {
	UserID      *string `json:"user_id,omitempty"`
	OldPassword string  `json:"old_password"`
	NewPassword string  `json:"new_password"`
}

type ListUsersResponse struct {
	Page   *Page           `json:"page,omitempty"`
	Result []*UserResponse `json:"result"`
}
