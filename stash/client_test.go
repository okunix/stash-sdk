package stash

import (
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	ctx := t.Context()
	client, err := NewClient(WithUser("test", "testpassword"))
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	st, err := client.GetStashByID(ctx, "1def6ac0-0c2d-4ad0-9db4-da0b04b5a214")
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	fmt.Printf("st: %+v\n", st)
}
