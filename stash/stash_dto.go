package stash

import (
	"time"

	"github.com/google/uuid"
)

type StashResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	MaintainerID uuid.UUID `json:"maintainer_id"`
	CreatedAt    time.Time `json:"created_at"`
	Locked       bool      `json:"locked"`
}

type ListStashResponse struct {
	Page   Page            `json:"page"`
	Result []StashResponse `json:"result"`
}

type StashMemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Since    time.Time `json:"since"`
}

type StashMaintainerResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
}

type ListStashMemberResponse struct {
	Maintainer StashMaintainerResponse `json:"maintainer"`
	Members    []StashMemberResponse   `json:"members"`
}

type AddStashMemberRequest struct {
	StashID uuid.UUID `json:"stash_id"`
	UserID  uuid.UUID `json:"user_id"`
}

type RemoveStashMemberRequest struct {
	StashID uuid.UUID `json:"stash_id"`
	UserID  uuid.UUID `json:"user_id"`
}

type UpdateStashRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type CreateStashRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Password    string  `json:"password"`
}
