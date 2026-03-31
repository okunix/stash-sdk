package stash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/okunix/stash-sdk/jsonutil"
)

type UserClient interface {
	GetToken(ctx context.Context, request GetTokenRequest) (*GetTokenResponse, error)
	Whoami(ctx context.Context) (*UserResponse, error)
	ChangePassword(ctx context.Context, request ChangePasswordRequest) error

	ListUsers(ctx context.Context, limit, offset uint) (*ListUsersResponse, error)
	GetUserByID(ctx context.Context, userID string) (*UserResponse, error)
	CreateUser(ctx context.Context, request CreateUserRequest) error
}

var _ UserClient = (*Client)(nil)

func (c *Client) GetToken(ctx context.Context, request GetTokenRequest) (*GetTokenResponse, error) {
	resp, err := c.get(ctx, "/api/v1/auth/login")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	tokenResponse, err := jsonutil.Read[GetTokenResponse](resp.Body)
	return &tokenResponse, err
}

func (c *Client) Whoami(ctx context.Context) (*UserResponse, error) {
	resp, err := c.get(ctx, "/api/v1/auth/whoami")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	userResponse, err := jsonutil.Read[UserResponse](resp.Body)
	return &userResponse, err
}

func (c *Client) ChangePassword(ctx context.Context, request ChangePasswordRequest) error {
	path := "/api/v1/auth/change-password"
	jsonbody, _ := json.Marshal(request)
	resp, err := c.patch(ctx, path, bytes.NewReader(jsonbody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) ListUsers(ctx context.Context, limit, offset uint) (*ListUsersResponse, error) {
	req, _ := c.newRequest(ctx, "GET", "/api/v1/users", nil)

	params := req.URL.Query()
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("offset", fmt.Sprintf("%d", offset))
	req.URL.RawQuery = params.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	userResponse, err := jsonutil.Read[ListUsersResponse](resp.Body)
	return &userResponse, err
}

func (c *Client) GetUserByID(ctx context.Context, userID string) (*UserResponse, error) {
	path := fmt.Sprintf("/api/v1/users/%s", userID)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	userResponse, err := jsonutil.Read[UserResponse](resp.Body)
	return &userResponse, err
}

func (c *Client) CreateUser(ctx context.Context, request CreateUserRequest) error {
	path := "/api/v1/users"
	jsonbody, _ := json.Marshal(request)
	resp, err := c.post(ctx, path, bytes.NewReader(jsonbody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}
