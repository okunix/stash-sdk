package stash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/okunix/stash-sdk/jsonutil"
)

type StashClient interface {
	GetStashByID(ctx context.Context, id string) (*StashResponse, error)
	GetStashByName(ctx context.Context, maintainerID, name string) (*StashResponse, error)
	ListStashes(ctx context.Context) (*ListStashesResponse, error)
	DeleteStash(ctx context.Context, stashID string) error
	CreateStash(ctx context.Context, request CreateStashRequest) error
	UpdateStash(ctx context.Context, stashID string, request UpdateStashRequest) error

	Lock(ctx context.Context, stashID string) error
	Unlock(ctx context.Context, stashID, password string) error

	GetSecrets(ctx context.Context, stashID string) (*SecretResponse, error)
	GetSecretsEntry(ctx context.Context, stashID, name string) (string, error)
	AddSecretsEntry(
		ctx context.Context,
		stashID string,
		request AddSecretRequest,
	) error
	RemoveSecretsEntry(ctx context.Context, stashID, name string) error

	GetStashMembers(ctx context.Context, stashID string) (*ListStashMemberResponse, error)
	AddStashMember(ctx context.Context, stashID, userID string) error
	RemoveStashMember(ctx context.Context, stashID, userID string) error
	GetStashMember(ctx context.Context, stashID, userID string) (*StashMemberResponse, error)
}

var _ StashClient = (*Client)(nil)

func (c *Client) GetStashByID(ctx context.Context, id string) (*StashResponse, error) {
	resp, err := c.get(ctx, "/api/v1/stashes/"+id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	stashResponse, err := jsonutil.Read[StashResponse](resp.Body)
	return &stashResponse, err
}

func (c *Client) ListStashes(ctx context.Context) (*ListStashesResponse, error) {
	path := "/api/v1/stashes"
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, jsonutil.ForgeError(resp.Body)
	}

	listStashResponse, err := jsonutil.Read[ListStashesResponse](resp.Body)
	return &listStashResponse, err
}

func (c *Client) DeleteStash(ctx context.Context, stashID string) error {
	path := fmt.Sprintf("/api/v1/stashes/%s", stashID)
	resp, err := c.delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) CreateStash(ctx context.Context, request CreateStashRequest) error {
	path := "/api/v1/stashes"
	body := bytes.NewBuffer([]byte{})
	json.NewEncoder(body).Encode(request)
	resp, err := c.post(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) UpdateStash(
	ctx context.Context,
	stashID string,
	request UpdateStashRequest,
) error {
	path := "/api/v1/stashes/" + stashID
	body := bytes.NewBuffer([]byte{})
	json.NewEncoder(body).Encode(request)
	resp, err := c.patch(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) Lock(ctx context.Context, stashID string) error {
	path := fmt.Sprintf("/api/v1/stashes/%s/lock", stashID)
	resp, err := c.post(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}
func (c *Client) Unlock(ctx context.Context, stashID, password string) error {
	path := fmt.Sprintf("/api/v1/stashes/%s/unlock", stashID)
	jsonbody, _ := json.Marshal(UnlockStashRequest{Password: password})
	resp, err := c.post(ctx, path, bytes.NewReader(jsonbody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) GetSecrets(ctx context.Context, stashID string) (*SecretResponse, error) {
	path := fmt.Sprintf("/api/v1/stashes/%s/secrets", stashID)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	secretResponse, err := jsonutil.Read[SecretResponse](resp.Body)
	return &secretResponse, err
}

func (c *Client) GetSecretsEntry(ctx context.Context, stashID, name string) (string, error) {
	path := fmt.Sprintf("/api/v1/stashes/%s/secrets/%s", stashID, name)
	resp, err := c.get(ctx, path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", jsonutil.ForgeError(resp.Body)
	}
	secretBytes, err := io.ReadAll(resp.Body)
	return string(secretBytes), err
}

func (c *Client) AddSecretsEntry(
	ctx context.Context,
	stashID string,
	request AddSecretRequest,
) error {
	path := fmt.Sprintf("/api/v1/stashes/%s/secrets", stashID)
	jsonbody, _ := json.Marshal(request)
	resp, err := c.put(ctx, path, bytes.NewReader(jsonbody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) RemoveSecretsEntry(ctx context.Context, stashID, name string) error {
	path := fmt.Sprintf("/api/v1/stashes/%s/secrets/%s", stashID, name)
	resp, err := c.delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) GetStashMembers(
	ctx context.Context,
	stashID string,
) (*ListStashMemberResponse, error) {
	path := fmt.Sprintf("/api/v1/stashes/%s/members", stashID)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	stashMembers, err := jsonutil.Read[ListStashMemberResponse](resp.Body)
	return &stashMembers, err
}

func (c *Client) AddStashMember(ctx context.Context, stashID, userID string) error {
	path := fmt.Sprintf("/api/v1/stashes/%s/members", stashID)
	jsonbody, _ := json.Marshal(AddStashMemberRequest{UserID: userID})
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

func (c *Client) RemoveStashMember(ctx context.Context, stashID, userID string) error {
	path := fmt.Sprintf("/api/v1/stashes/%s/members/%s", stashID, userID)
	resp, err := c.delete(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jsonutil.ForgeError(resp.Body)
	}
	return nil
}

func (c *Client) GetStashByName(
	ctx context.Context,
	maintainerID, name string,
) (*StashResponse, error) {
	path := fmt.Sprintf("/api/v1/stashes/by-name/%s/%s", maintainerID, name)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	stashResponse, err := jsonutil.Read[StashResponse](resp.Body)
	return &stashResponse, err
}

func (c *Client) GetStashMember(
	ctx context.Context,
	stashID, userID string,
) (*StashMemberResponse, error) {
	path := fmt.Sprintf("/api/v1/stashes/%s/members/%s", stashID, userID)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, jsonutil.ForgeError(resp.Body)
	}
	stashMemberResponse, err := jsonutil.Read[StashMemberResponse](resp.Body)
	return &stashMemberResponse, err
}
