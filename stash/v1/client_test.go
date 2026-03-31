package stash

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	ctx := t.Context()
	client, err := NewClient()
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	if client == nil {
		t.Fatal("client is nil")
		return
	}
	if err := client.Ping(ctx); err != nil {
		t.Fatal(err.Error())
		return
	}
}
