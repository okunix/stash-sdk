package stash

import "context"

type StashClient interface {
	GetStashByID(ctx context.Context)
}

func (c *Client) GetStashByID(ctx context.Context) {
}
