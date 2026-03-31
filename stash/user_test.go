package stash

// user_test.go covers edge cases for every method in user.go that are not
// already exercised in stash_test.go. The shared test helpers (newTestClient,
// errorBody, mustJSON, jsonReply) are defined in stash_test.go and available
// here because both files belong to the same test package.

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// GetToken
// ---------------------------------------------------------------------------

// GetToken sends a GET to /api/v1/auth/login — it does NOT forward the
// request body (the caller must call WithUser or set the token separately).
// We verify the malformed-JSON path here since the happy-path and 500 are
// already in stash_test.go.

func TestGetToken_MalformedResponseBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json{{{{"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetToken(context.Background(), GetTokenRequest{Username: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected error from malformed JSON, got nil")
	}
}

func TestGetToken_EmptyTokenInResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, `{"token":""}`)
	})
	c, _ := newTestClient(t, mux)
	resp, err := c.GetToken(context.Background(), GetTokenRequest{Username: "u", Password: "p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "" {
		t.Errorf("expected empty token, got %q", resp.Token)
	}
}

func TestGetToken_Forbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "account locked"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetToken(context.Background(), GetTokenRequest{Username: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "account locked") {
		t.Errorf("error = %q, want it to mention 'account locked'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Whoami
// ---------------------------------------------------------------------------

func TestWhoami_InternalServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(
			w,
			http.StatusInternalServerError,
			errorBody("internal error", "unexpected failure"),
		)
	})
	c, _ := newTestClient(t, mux)
	_, err := c.Whoami(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWhoami_MalformedResponseBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad json"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.Whoami(context.Background())
	if err == nil {
		t.Fatal("expected error from malformed JSON, got nil")
	}
}

func TestWhoami_LockedUser(t *testing.T) {
	// Verify that a locked user field is correctly deserialised.
	now := time.Now().UTC().Truncate(time.Second)
	user := UserResponse{ID: "u-locked", Username: "eve", Locked: true, CreatedAt: now}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, user))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Locked {
		t.Error("expected Locked=true")
	}
}

func TestWhoami_UsesGETMethod(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/whoami", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		jsonReply(
			w,
			http.StatusOK,
			mustJSON(t, UserResponse{ID: "u-1", Username: "alice", CreatedAt: time.Now()}),
		)
	})
	c, _ := newTestClient(t, mux)
	_, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ChangePassword
// ---------------------------------------------------------------------------

func TestChangePassword_AdminScope_WithUserID(t *testing.T) {
	// ChangePasswordRequest has an optional UserID field for admin use.
	targetID := "u-target"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		var req ChangePasswordRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.UserID == nil || *req.UserID != targetID {
			t.Errorf("expected UserID=%q in body, got %v", targetID, req.UserID)
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.ChangePassword(context.Background(), ChangePasswordRequest{
		UserID:      &targetID,
		OldPassword: "old",
		NewPassword: "new",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangePassword_UsesPATCHMethod(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		jsonReply(w, http.StatusOK, `{}`)
	})
	c, _ := newTestClient(t, mux)
	_ = c.ChangePassword(
		context.Background(),
		ChangePasswordRequest{OldPassword: "a", NewPassword: "b"},
	)
}

func TestChangePassword_InternalServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusInternalServerError, errorBody("internal error", "db unavailable"))
	})
	c, _ := newTestClient(t, mux)
	err := c.ChangePassword(
		context.Background(),
		ChangePasswordRequest{OldPassword: "a", NewPassword: "b"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChangePassword_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusUnauthorized, errorBody("unauthorized", "not authenticated"))
	})
	c, _ := newTestClient(t, mux)
	c.token = "" // no token set
	err := c.ChangePassword(
		context.Background(),
		ChangePasswordRequest{OldPassword: "a", NewPassword: "b"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListUsers
// ---------------------------------------------------------------------------

func TestListUsers_PaginationParamsOnURL(t *testing.T) {
	// Verify that limit/offset are forwarded as query parameters with correct
	// values, including a non-zero offset.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, err := strconv.Atoi(q.Get("limit"))
		if err != nil || limit != 5 {
			t.Errorf("limit = %q, want 5", q.Get("limit"))
		}
		offset, err := strconv.Atoi(q.Get("offset"))
		if err != nil || offset != 20 {
			t.Errorf("offset = %q, want 20", q.Get("offset"))
		}
		jsonReply(w, http.StatusOK, mustJSON(t, ListUsersResponse{
			Page:   &Page{Limit: 5, Offset: 20, Total: 25},
			Result: []*UserResponse{},
		}))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.ListUsers(context.Background(), 5, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListUsers_EmptyPage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, ListUsersResponse{
			Page:   &Page{Limit: 10, Offset: 0, Total: 0},
			Result: []*UserResponse{},
		}))
	})
	c, _ := newTestClient(t, mux)
	resp, err := c.ListUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Result) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp.Result))
	}
}

func TestListUsers_InternalServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusInternalServerError, errorBody("internal error", "db unavailable"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.ListUsers(context.Background(), 10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListUsers_MalformedResponseBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{{not json"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.ListUsers(context.Background(), 10, 0)
	if err == nil {
		t.Fatal("expected error from malformed JSON, got nil")
	}
}

func TestListUsers_UsesGETMethod(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		jsonReply(w, http.StatusOK, mustJSON(t, ListUsersResponse{Result: []*UserResponse{}}))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.ListUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListUsers_MultipleUsers(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	users := []*UserResponse{
		{ID: "u-1", Username: "alice", CreatedAt: now},
		{ID: "u-2", Username: "bob", CreatedAt: now},
		{ID: "u-3", Username: "carol", CreatedAt: now},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, ListUsersResponse{
			Page:   &Page{Limit: 10, Offset: 0, Total: 3},
			Result: users,
		}))
	})
	c, _ := newTestClient(t, mux)
	resp, err := c.ListUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Result) != 3 {
		t.Errorf("expected 3 users, got %d", len(resp.Result))
	}
	if resp.Result[1].Username != "bob" {
		t.Errorf("second user = %q, want bob", resp.Result[1].Username)
	}
}

// ---------------------------------------------------------------------------
// GetUserByID
// ---------------------------------------------------------------------------

func TestGetUserByID_Forbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/u-private", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "insufficient permissions"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetUserByID(context.Background(), "u-private")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient permissions") {
		t.Errorf("error = %q, want it to mention 'insufficient permissions'", err.Error())
	}
}

func TestGetUserByID_InternalServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusInternalServerError, errorBody("internal error", "db unavailable"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetUserByID(context.Background(), "u-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserByID_MalformedResponseBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/u-1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad json"))
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetUserByID(context.Background(), "u-1")
	if err == nil {
		t.Fatal("expected error from malformed JSON, got nil")
	}
}

func TestGetUserByID_LockedUserField(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/u-locked", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, UserResponse{
			ID:        "u-locked",
			Username:  "locked_user",
			Locked:    true,
			CreatedAt: now,
		}))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.GetUserByID(context.Background(), "u-locked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Locked {
		t.Error("expected Locked=true")
	}
}

func TestGetUserByID_WithExpiredAt(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expiry := now.Add(-24 * time.Hour)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/u-expired", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusOK, mustJSON(t, UserResponse{
			ID:        "u-expired",
			Username:  "expired_user",
			CreatedAt: now,
			ExpiredAt: &expiry,
		}))
	})
	c, _ := newTestClient(t, mux)
	got, err := c.GetUserByID(context.Background(), "u-expired")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ExpiredAt == nil {
		t.Fatal("expected ExpiredAt to be set")
	}
}

func TestGetUserByID_UsesGETMethod(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/u-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		jsonReply(
			w,
			http.StatusOK,
			mustJSON(t, UserResponse{ID: "u-1", Username: "alice", CreatedAt: now}),
		)
	})
	c, _ := newTestClient(t, mux)
	_, err := c.GetUserByID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateUser
// ---------------------------------------------------------------------------

func TestCreateUser_BadRequest_MissingFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Username == "" || req.Password == "" {
			jsonReply(
				w,
				http.StatusBadRequest,
				errorBody("bad request", "username and password are required"),
			)
			return
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateUser(context.Background(), CreateUserRequest{Username: "", Password: ""})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "username and password are required") {
		t.Errorf("error = %q, want it to mention required fields", err.Error())
	}
}

func TestCreateUser_Forbidden(t *testing.T) {
	// Creating users may be restricted to admins.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusForbidden, errorBody("forbidden", "admin only"))
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateUser(
		context.Background(),
		CreateUserRequest{Username: "newuser", Password: "pass"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "admin only") {
		t.Errorf("error = %q, want it to mention 'admin only'", err.Error())
	}
}

func TestCreateUser_InternalServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusInternalServerError, errorBody("internal error", "db unavailable"))
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateUser(context.Background(), CreateUserRequest{Username: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateUser_UsesPOSTMethod(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	_ = c.CreateUser(context.Background(), CreateUserRequest{Username: "u", Password: "p"})
}

func TestCreateUser_RequestBodyEncoding(t *testing.T) {
	// Verify that username and password are actually serialised into the body.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if req.Username != "testuser" {
			t.Errorf("username = %q, want testuser", req.Username)
		}
		if req.Password != "testpass" {
			t.Errorf("password = %q, want testpass", req.Password)
		}
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)
	err := c.CreateUser(
		context.Background(),
		CreateUserRequest{Username: "testuser", Password: "testpass"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateUser_ContextCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, http.StatusCreated, `{}`)
	})
	c, _ := newTestClient(t, mux)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.CreateUser(ctx, CreateUserRequest{Username: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
}
