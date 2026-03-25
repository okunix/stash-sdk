package stash

import "context"

type StashClient interface {
	GetStashByID(ctx context.Context, id string) error
}

var _ StashClient = (*Client)(nil)

func (c *Client) GetStashByID(ctx context.Context, id string) error {
	resp, err := c.get(ctx, "/api/v1/stashes/"+id)

}
