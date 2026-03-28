package stash

import (
	"context"
	"errors"

	"github.com/okunix/stash-sdk/jsonutil"
)

type StashClient interface {
	GetStashByID(ctx context.Context, id string) (*StashResponse, error)
}

var _ StashClient = (*Client)(nil)

func (c *Client) GetStashByID(ctx context.Context, id string) (*StashResponse, error) {
	resp, err := c.get(ctx, "/api/v1/stashes/"+id)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("error status code")
	}

	stashResponse, err := jsonutil.Read[StashResponse](resp.Body)
	if err != nil {
		return nil, err
	}

	return &stashResponse, nil
}
