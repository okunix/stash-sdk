package stash

import (
	"testing"
)

var (
	username = "yusuf"
	password = "1234567890"
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
	st, err := client.GetStashByID(ctx, "bac5ff60-4f9b-42af-9584-f77804f18357")
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

func TestCreateStash(t *testing.T) {
	ctx := t.Context()

	client := initClient(t)
	req := CreateStashRequest{Name: "test_stash", Password: "supersecretpass"}
	if err := client.CreateStash(ctx, req); err != nil {
		t.Fatal(err.Error())
		return
	}
}
