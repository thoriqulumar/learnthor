package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/learnthor/backend/models"
)

func TestRegister_CreatesStudentWithToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	db := newTestDB(t)
	svc := NewAuthService(db)

	resp, err := svc.Register(RegisterRequest{
		Username: "newlearner",
		Email:    "new@example.com",
		Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected a JWT token")
	}
	if resp.User.Role != models.UserRoleStudent {
		t.Fatalf("role = %s, want student", resp.User.Role)
	}
	if resp.User.PreferredGloss != "both" {
		t.Fatalf("preferred_gloss = %q, want 'both'", resp.User.PreferredGloss)
	}
	if resp.User.PasswordHash == "supersecret" {
		t.Fatal("password must be hashed, not stored in plaintext")
	}
}

func TestRegister_RejectsDuplicateEmailOrUsername(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	db := newTestDB(t)
	svc := NewAuthService(db)

	if _, err := svc.Register(RegisterRequest{
		Username: "taken", Email: "taken@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("first register: %v", err)
	}

	for _, req := range []RegisterRequest{
		{Username: "taken", Email: "other@example.com", Password: "supersecret"}, // dup username
		{Username: "other", Email: "taken@example.com", Password: "supersecret"}, // dup email
	} {
		if _, err := svc.Register(req); !errors.Is(err, ErrBadRequest) {
			t.Fatalf("%+v: err = %v, want ErrBadRequest", req, err)
		}
	}
}

func TestLogin_SuccessAndTokenRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	db := newTestDB(t)
	svc := NewAuthService(db)

	if _, err := svc.Register(RegisterRequest{
		Username: "loginuser", Email: "login@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	resp, err := svc.Login(LoginRequest{Email: "login@example.com", Password: "supersecret"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	claims, err := ParseToken(resp.Token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != resp.User.ID {
		t.Fatalf("token user = %s, want %s", claims.UserID, resp.User.ID)
	}
}

func TestLogin_RejectsBadCredentials(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	db := newTestDB(t)
	svc := NewAuthService(db)

	if _, err := svc.Register(RegisterRequest{
		Username: "loginuser2", Email: "login2@example.com", Password: "supersecret",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	for _, req := range []LoginRequest{
		{Email: "ghost@example.com", Password: "supersecret"},    // unknown email
		{Email: "login2@example.com", Password: "wrongpassword"}, // wrong password
	} {
		if _, err := svc.Login(req); !errors.Is(err, ErrBadRequest) {
			t.Fatalf("%+v: err = %v, want ErrBadRequest", req, err)
		}
	}
}

func TestParseToken_RejectsGarbage(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	if _, err := ParseToken("not.a.token"); err == nil {
		t.Fatal("garbage token should be rejected")
	}
}

func TestPreferences_UpdateAndValidation(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	db := newTestDB(t)
	svc := NewAuthService(db)
	user := createTestUser(t, db, models.UserRoleStudent)

	prefs, err := svc.GetPreferences(user.ID)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	if prefs.PreferredGloss != "both" {
		t.Fatalf("default gloss = %q, want 'both'", prefs.PreferredGloss)
	}

	updated, err := svc.UpdatePreferences(user.ID, UpdatePreferencesRequest{PreferredGloss: "id"})
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	if updated.PreferredGloss != "id" {
		t.Fatalf("gloss = %q, want 'id'", updated.PreferredGloss)
	}

	// Persisted across reads.
	again, _ := svc.GetPreferences(user.ID)
	if again.PreferredGloss != "id" {
		t.Fatalf("re-read gloss = %q, want 'id'", again.PreferredGloss)
	}

	// Invalid value rejected.
	if _, err := svc.UpdatePreferences(user.ID, UpdatePreferencesRequest{PreferredGloss: "xx"}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("invalid gloss: err = %v, want ErrBadRequest", err)
	}

	// Nonexistent user.
	if _, err := svc.GetPreferences(uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ghost user: err = %v, want ErrNotFound", err)
	}
}
