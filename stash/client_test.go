package stash

import (
	"testing"
)

var (
	username = "test"
	password = "testpassword"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(WithUser(username, password))
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	if client == nil {
		t.Fatal("client is nil")
		return
	}
}

func initClient(t *testing.T) *Client {
	client, err := NewClient(WithUser(username, password))
	if err != nil {
		t.Fatal(err.Error())
		return nil
	}
	return client
}

func TestGetStashByID(t *testing.T) {
	ctx := t.Context()
	client := initClient(t)
	st, err := client.GetStashByID(ctx, "1def6ac0-0c2d-4ad0-9db4-da0b04b5a214")
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	t.Logf("stash: %+v\n", st)
}

func TestListStashes(t *testing.T) {
	ctx := t.Context()
	client := initClient(t)
	listStashesResponse, err := client.ListStashes(ctx, 30, 0)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	t.Logf("stash: %+v\n", listStashesResponse)
}
