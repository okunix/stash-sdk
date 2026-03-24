package stash

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/okunix/stash-sdk/jsonutil"
)

type Client struct {
	client http.Client
	token  string
	addr   string
}

type clientOption func(*Client) error

func WithUser(username, password string) clientOption {
	type request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	type response struct {
		Token string `json:"token"`
	}
	type errorMessage struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	return func(c *Client) error {
		reqBody := request{
			Username: username,
			Password: password,
		}
		bodyJson, _ := json.Marshal(reqBody)
		reader := bytes.NewReader(bodyJson)
		resp, err := c.post(context.Background(), "/api/v1/login", reader)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			errResp, err := jsonutil.Read[errorMessage](resp.Body)
			if err != nil {
				return err
			}
			return errors.New(errResp.Detail)
		}
		var respBody response
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			return err
		}
		c.token = respBody.Token
		return nil
	}
}

func WithToken(token string) clientOption {
	return func(c *Client) error {
		c.token = token
		return nil
	}
}

func WithTimeout(t time.Duration) clientOption {
	return func(c *Client) error {
		c.client.Timeout = t
		return nil
	}
}

func WithAddr(addr string) clientOption {
	return func(c *Client) error {
		c.addr = addr
		return nil
	}
}

func newDefaultClient() Client {
	return Client{
		client: http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		},
		addr: "http://localhost:7878",
	}
}

func NewClient(opts ...clientOption) (*Client, error) {
	client := newDefaultClient()
	for _, opt := range opts {
		if err := opt(&client); err != nil {
			return nil, err
		}
	}
	return &client, nil
}

func (c *Client) getURL(path string) string {
	return fmt.Sprintf("%s%s", c.addr, path)
}

func (c *Client) do(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

func (c *Client) newRequest(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.getURL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, err
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, "GET", path, nil)
}

func (c *Client) post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.do(ctx, "POST", path, body)
}

func (c *Client) put(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.do(ctx, "PUT", path, body)
}

func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, "DELETE", path, nil)
}

func (c *Client) patch(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.do(ctx, "PATCH", path, body)
}
