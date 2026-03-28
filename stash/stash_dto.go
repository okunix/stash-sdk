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
