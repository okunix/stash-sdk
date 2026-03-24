package stash

import "context"

type UserClient interface {
	GetToken(ctx context.Context)
}

func (c *Client) GetToken(ctx context.Context) {
}
