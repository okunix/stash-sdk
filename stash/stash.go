package stash

import (
	"context"
	"errors"
	"fmt"

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
	return &stashResponse, err
}

func (c *Client) ListStashes(ctx context.Context, limit, offset uint) (*ListStashResponse, error) {
	path := "/api/v1/stashes"
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("limit", fmt.Sprintf("%d", limit))
	q.Add("offset", fmt.Sprintf("%d", offset))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("error status code")
	}

	listStashResponse, err := jsonutil.Read[ListStashResponse](resp.Body)
	return &listStashResponse, err
}
