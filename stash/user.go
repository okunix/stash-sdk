package stash

import "context"

type UserClient interface {
	GetToken(ctx context.Context)
}

var _ UserClient = (*Client)(nil)

func (c *Client) GetToken(ctx context.Context) {
}
