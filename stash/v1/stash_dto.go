package stash

import (
	"time"
)

type StashResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	MaintainerID string    `json:"maintainer_id"`
	CreatedAt    time.Time `json:"created_at"`
	Locked       bool      `json:"locked"`
}

type StashMemberResponse struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Since    time.Time `json:"since"`
}

type StashMaintainerResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type ListStashMemberResponse struct {
	Maintainer StashMaintainerResponse `json:"maintainer"`
	Members    []StashMemberResponse   `json:"members"`
}

type AddStashMemberRequest struct {
	UserID string `json:"user_id"`
}

type UpdateStashRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type CreateStashRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Password    string  `json:"password"`
}

type SecretResponse struct {
	Keys       []string  `json:"keys"`
	UnlockedAt time.Time `json:"unlocked_at"`
}

type AddSecretRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UnlockStashRequest struct {
	Password string `json:"password"`
}

type ListStashesResponse struct {
	Maintainer []StashResponse `json:"maintainer"`
	Member     []StashResponse `json:"member"`
}
