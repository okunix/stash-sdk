package stash

import (
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	ctx := t.Context()
	client, err := NewClient(WithUser("httpie", "httpieclient"))
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	resp, err := client.get(ctx, "/api/v1/stashes")
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	fmt.Printf("statusCode: %v\n", resp.StatusCode)
}
