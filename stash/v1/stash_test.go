package stash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// errorBody builds a JSON error payload matching jsonutil.Message.
func errorBody(message, detail string) string {
	return fmt.Sprintf(`{"code":400,"message":%q,"detail":%q}`, message, detail)
}

// newTestClient spins up an httptest.Server and returns a Client pointed at it.
func newTestClient(t *testing.T, mux *http.ServeMux) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c, err := NewClient(WithAddr(srv.URL), WithToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

// mustJSON marshals v to JSON or fatals.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(b)
}

func jsonReply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// ---------------------------------------------------------------------------
// WithUser option
// ---------------------------------------------------------------------------

func TestWithUser_Success(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req GetTokenRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Username != "alice" || req.Password != "secret" {
			jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "bad credentials"))
			return
		}
		jsonReply(w, http.StatusOK, `{"token":"tok-abc"}`)
	})

	c, err := NewClient(WithAddr(srv.URL), WithUser("alice", "secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.token != "tok-abc" {
		t.Errorf("token = %q, want tok-abc", c.token)
	}
}

func TestWithUser_WrongCredentials(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "invalid credentials"))
	})

	_, err := NewClient(WithAddr(srv.URL), WithUser("alice", "wrong"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWithToken(t *testing.T) {
	c, err := NewClient(WithToken("my-static-token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.token != "my-static-token" {
		t.Errorf("token = %q, want my-static-token", c.token)
	}
}

// ---------------------------------------------------------------------------
// Auth / User endpoints
// ---------------------------------------------------------------------------

func TestGetToken_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, `{"token":"tok-xyz"}`)
	})
	c, _ := newTestClient(t, mux)
	resp, err := c.GetToken(context.Background(), GetTokenRequest{Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "tok-xyz" {
		t.Errorf("token = %q, want tok-xyz", resp.Token)
	}
}

func TestGetToken_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(
			w,
			http.StatusInternalServerError,
			errorBody("internal error", "something went wrong"),
		)
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetToken(context.Background(), GetTokenRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWhoami_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	user := UserResponse{ID: "u-1", Username: "alice", Locked: false, CreatedAt: now}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, user))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u-1" || got.Username != "alice" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestWhoami_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "token expired"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.Whoami(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token expired") {
		t.Errorf("error = %q, want it to mention 'token expired'", err.Error())
	}
}

func TestChangePassword_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var req ChangePasswordRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.OldPassword == "" || req.NewPassword == "" {
			jsonReply(w, http.StatusBadRequest, errorBody("bad request", "missing fields"))
			return
		}
		jsonReply(w, http.StatusOK, `{"message":"ok"}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.ChangePassword(context.Background(), ChangePasswordRequest{
		OldPassword: "old",
		NewPassword: "new",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "wrong old password"))
	})
	c, _ := newTestClient(t, mux)
	err := c.ChangePassword(context.Background(), ChangePasswordRequest{
		OldPassword: "bad",
		NewPassword: "new",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListUsers_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	body := mustJSON(t, ListUsersResponse{
		Page: &Page{Limit: 10, Offset: 0, Total: 1},
		Result: []*UserResponse{
			{ID: "u-1", Username: "alice", CreatedAt: now},
		},
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")
		if limit == "" || offset == "" {
			t.Error("missing pagination query params")
		}
		jsonReply(w, http.StatusOK, body)
	})
	c, _ := newTestClient(t, mux)
	resp, err := c.ListUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Result) != 1 {
		t.Errorf("expected 1 user, got %d", len(resp.Result))
	}
}

func TestListUsers_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "not authenticated"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.ListUsers(context.Background(), 10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserByID_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/u-1", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(
			w,
			http.StatusOK,
			mustJSON(t, UserResponse{ID: "u-1", Username: "alice", CreatedAt: now}),
		)
	})
	c, _ := newTestClient(t, mux)
	got, err := c.GetUserByID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u-1" {
		t.Errorf("id = %q, want u-1", got.ID)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "user not found"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetUserByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateUser_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req CreateUserRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Username == "" || req.Password == "" {
			jsonReply(w, http.StatusBadRequest, errorBody("bad request", "missing fields"))
			return
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateUser(context.Background(), CreateUserRequest{Username: "bob", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateUser_Conflict(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusConflict, errorBody("conflict", "username already taken"))
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateUser(
		context.Background(),
		CreateUserRequest{Username: "alice", Password: "pass"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "username already taken") {
		t.Errorf("error = %q, want it to mention 'username already taken'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Stash CRUD
// ---------------------------------------------------------------------------

func TestGetStashByID_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	stash := StashResponse{
		ID:           "s-1",
		Name:         "my-stash",
		MaintainerID: "u-1",
		CreatedAt:    now,
		Locked:       false,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, stash))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.GetStashByID(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "s-1" || got.Name != "my-stash" {
		t.Errorf("unexpected stash: %+v", got)
	}
}

func TestGetStashByID_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetStashByID(context.Background(), "s-missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListStashes_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	resp := ListStashesResponse{
		Member: []StashResponse{
			{ID: "s-1", Name: "stash-a", MaintainerID: "u-2", CreatedAt: now},
			{ID: "s-2", Name: "stash-b", MaintainerID: "u-2", CreatedAt: now},
		},
		Maintainer: []StashResponse{
			{ID: "s-3", Name: "stash-c", MaintainerID: "u-1", CreatedAt: now},
			{ID: "s-4", Name: "stash-d", MaintainerID: "u-1", CreatedAt: now},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, resp))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.ListStashes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Maintainer) != 2 {
		t.Errorf("expected 2 stashes, got %d", len(got.Maintainer))
	}
}

func TestListStashes_EmptyResult(t *testing.T) {
	resp := ListStashesResponse{
		Member:     []StashResponse{},
		Maintainer: []StashResponse{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, resp))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.ListStashes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Maintainer) != 0 {
		t.Errorf("expected 0 stashes, got %d", len(got.Maintainer))
	}
}

func TestListStashes_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "not authenticated"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.ListStashes(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateStash_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req CreateStashRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name == "" || req.Password == "" {
			jsonReply(w, http.StatusBadRequest, errorBody("bad request", "missing fields"))
			return
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateStash(context.Background(), CreateStashRequest{Name: "vault", Password: "pass"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateStash_Conflict(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusConflict, errorBody("conflict", "stash name already exists"))
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateStash(context.Background(), CreateStashRequest{Name: "vault", Password: "pass"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateStash_Success(t *testing.T) {
	newName := "renamed"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var req UpdateStashRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name == nil || *req.Name != "renamed" {
			t.Errorf("unexpected body: %+v", req)
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.UpdateStash(context.Background(), UpdateStashRequest{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStash_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	name := "x"
	err := c.UpdateStash(context.Background(), UpdateStashRequest{Name: &name})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteStash_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.DeleteStash(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStash_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	err := c.DeleteStash(context.Background(), "s-ghost")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteStash_Forbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "not the maintainer"))
	})
	c, _ := newTestClient(t, mux)
	err := c.DeleteStash(context.Background(), "s-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not the maintainer") {
		t.Errorf("error = %q, want it to mention 'not the maintainer'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Lock / Unlock
// ---------------------------------------------------------------------------

func TestLock_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.Lock(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLock_AlreadyLocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/lock", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusConflict, errorBody("conflict", "stash already locked"))
	})
	c, _ := newTestClient(t, mux)
	err := c.Lock(context.Background(), "s-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLock_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-ghost/lock", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	err := c.Lock(context.Background(), "s-ghost")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUnlock_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/unlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req UnlockStashRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Password != "correct" {
			jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "wrong password"))
			return
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.Unlock(context.Background(), "s-1", "correct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnlock_WrongPassword(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/unlock", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "wrong password"))
	})
	c, _ := newTestClient(t, mux)
	err := c.Unlock(context.Background(), "s-1", "wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "wrong password") {
		t.Errorf("error = %q, want it to mention 'wrong password'", err.Error())
	}
}

func TestUnlock_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-ghost/unlock", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	err := c.Unlock(context.Background(), "s-ghost", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Secrets
// ---------------------------------------------------------------------------

func TestGetSecrets_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	resp := SecretResponse{Keys: []string{"db_pass", "api_key"}, UnlockedAt: now}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, resp))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.GetSecrets(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(got.Keys))
	}
}

func TestGetSecrets_Locked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "stash is locked"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetSecrets(context.Background(), "s-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stash is locked") {
		t.Errorf("error = %q, want it to mention 'stash is locked'", err.Error())
	}
}

func TestGetSecrets_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-ghost/secrets", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetSecrets(context.Background(), "s-ghost")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSecretsEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/api/v1/stashes/s-1/secrets/db_pass",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("supersecretvalue"))
		},
	)
	c, _ := newTestClient(t, mux)
	val, err := c.GetSecretsEntry(context.Background(), "s-1", "db_pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "supersecretvalue" {
		t.Errorf("value = %q, want supersecretvalue", val)
	}
}

func TestGetSecretsEntry_KeyNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "secret key not found"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetSecretsEntry(context.Background(), "s-1", "missing_key")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSecretsEntry_StashLocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/api/v1/stashes/s-1/secrets/db_pass",
		func(w http.ResponseWriter, r *http.Request) {
			jsonReply(w, http.StatusForbidden, errorBody("forbidden", "stash is locked"))
		},
	)
	c, _ := newTestClient(t, mux)
	_, err := c.GetSecretsEntry(context.Background(), "s-1", "db_pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddSecretsEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var req AddSecretRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name == "" || req.Value == "" {
			jsonReply(w, http.StatusBadRequest, errorBody("bad request", "missing fields"))
			return
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.AddSecretsEntry(
		context.Background(),
		"s-1",
		AddSecretRequest{Name: "db_pass", Value: "s3cr3t"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddSecretsEntry_StashLocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "stash is locked"))
	})
	c, _ := newTestClient(t, mux)
	err := c.AddSecretsEntry(context.Background(), "s-1", AddSecretRequest{Name: "k", Value: "v"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddSecretsEntry_Conflict(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusConflict, errorBody("conflict", "key already exists"))
	})
	c, _ := newTestClient(t, mux)
	err := c.AddSecretsEntry(
		context.Background(),
		"s-1",
		AddSecretRequest{Name: "existing_key", Value: "v"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key already exists") {
		t.Errorf("error = %q, want it to mention 'key already exists'", err.Error())
	}
}

func TestRemoveSecretsEntry_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/api/v1/stashes/s-1/secrets/db_pass",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			jsonReply(w, http.StatusOK, `{}`)
		},
	)
	c, _ := newTestClient(t, mux)
	err := c.RemoveSecretsEntry(context.Background(), "s-1", "db_pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveSecretsEntry_KeyNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/secrets/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "secret key not found"))
	})
	c, _ := newTestClient(t, mux)
	err := c.RemoveSecretsEntry(context.Background(), "s-1", "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveSecretsEntry_StashLocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/api/v1/stashes/s-1/secrets/db_pass",
		func(w http.ResponseWriter, r *http.Request) {
			jsonReply(w, http.StatusForbidden, errorBody("forbidden", "stash is locked"))
		},
	)
	c, _ := newTestClient(t, mux)
	err := c.RemoveSecretsEntry(context.Background(), "s-1", "db_pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Stash Members
// ---------------------------------------------------------------------------

func TestGetStashMembers_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	resp := ListStashMemberResponse{
		Maintainer: StashMaintainerResponse{UserID: "u-1", Username: "alice"},
		Members: []StashMemberResponse{
			{UserID: "u-2", Username: "bob", Since: now},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/members", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, resp))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.GetStashMembers(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Maintainer.UserID != "u-1" {
		t.Errorf("maintainer = %q, want u-1", got.Maintainer.UserID)
	}
	if len(got.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(got.Members))
	}
}

func TestGetStashMembers_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-ghost/members", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusNotFound, errorBody("not found", "stash not found"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetStashMembers(context.Background(), "s-ghost")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddStashMember_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req AddStashMemberRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.UserID == "" {
			jsonReply(w, http.StatusBadRequest, errorBody("bad request", "missing user_id"))
			return
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.AddStashMember(context.Background(), "s-1", "u-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddStashMember_AlreadyMember(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/members", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusConflict, errorBody("conflict", "user is already a member"))
	})
	c, _ := newTestClient(t, mux)
	err := c.AddStashMember(context.Background(), "s-1", "u-2")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "user is already a member") {
		t.Errorf("error = %q, want it to mention 'user is already a member'", err.Error())
	}
}

func TestAddStashMember_Forbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/members", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(
			w,
			http.StatusForbidden,
			errorBody("forbidden", "only maintainer can add members"),
		)
	})
	c, _ := newTestClient(t, mux)
	err := c.AddStashMember(context.Background(), "s-1", "u-3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveStashMember_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/members/u-2", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.RemoveStashMember(context.Background(), "s-1", "u-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveStashMember_NotMember(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/api/v1/stashes/s-1/members/u-999",
		func(w http.ResponseWriter, r *http.Request) {
			jsonReply(w, http.StatusNotFound, errorBody("not found", "user is not a member"))
		},
	)
	c, _ := newTestClient(t, mux)
	err := c.RemoveStashMember(context.Background(), "s-1", "u-999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveStashMember_Forbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1/members/u-2", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(
			w,
			http.StatusForbidden,
			errorBody("forbidden", "only maintainer can remove members"),
		)
	})
	c, _ := newTestClient(t, mux)
	err := c.RemoveStashMember(context.Background(), "s-1", "u-2")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Request headers
// ---------------------------------------------------------------------------

func TestAuthorizationHeader_SetWhenTokenPresent(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-token" {
			t.Errorf("Authorization = %q, want 'Bearer my-token'", auth)
		}
		jsonReply(
			w,
			http.StatusOK,
			mustJSON(t, UserResponse{ID: "u-1", Username: "alice", CreatedAt: time.Now()}),
		)
	})
	c, _ := newTestClient(t, mux)
	c.token = "my-token"
	_, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthorizationHeader_AbsentWhenNoToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "no token"))
	})
	c, _ := newTestClient(t, mux)
	c.token = ""
	_, _ = c.Whoami(context.Background()) // we only care about the header check above
}

func TestContentTypeHeader(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes", func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	_ = c.CreateStash(context.Background(), CreateStashRequest{Name: "vault", Password: "pass"})
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestContextCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stashes/s-1", func(w http.ResponseWriter, r *http.Request) {
		// handler that blocks — should never reach the response because the
		// context is already cancelled before the request is made
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.GetStashByID(ctx, "s-1")
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
}
