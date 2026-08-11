package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const testJWTSecret = "test-jwt-secret-0123456789abcdef"

type memoryStore struct {
	users map[string]*User
}

func (s *memoryStore) FindByUsername(_ context.Context, username string) (*User, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	copy := *user
	return &copy, nil
}

func TestServiceLoginIssuesVerifiableToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("login-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	store := &memoryStore{users: map[string]*User{
		"login-test-user": {ID: 1, Username: "login-test-user", PasswordHash: string(hash)},
	}}
	service, err := NewService(store, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	service.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	result, err := service.Login(context.Background(), "login-test-user", "login-test-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.AccessToken == "" || result.ExpiresIn != 3600 || result.User.ID != 1 {
		t.Fatalf("unexpected login result: %+v", result)
	}

	service.now = time.Now
	_, err = service.VerifyToken(result.AccessToken)
	if err == nil {
		t.Fatal("expected historical token to be expired")
	}

	result, err = service.Login(context.Background(), "login-test-user", "login-test-password")
	if err != nil {
		t.Fatalf("login with current clock: %v", err)
	}
	claims, err := service.VerifyToken(result.AccessToken)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Subject != "1" || claims.Username != "login-test-user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestServiceLoginRejectsInvalidCredentials(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("login-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	service, err := NewService(&memoryStore{users: map[string]*User{
		"login-test-user": {ID: 1, Username: "login-test-user", PasswordHash: string(hash)},
	}}, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	for _, test := range []struct {
		username string
		password string
	}{
		{username: "missing", password: "login-test-password"},
		{username: "login-test-user", password: "wrong"},
	} {
		if _, err := service.Login(context.Background(), test.username, test.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials for %+v, got %v", test, err)
		}
	}
}

func TestNewServiceRejectsWeakJWTSecrets(t *testing.T) {
	store := &memoryStore{users: make(map[string]*User)}
	for _, secret := range []string{
		"short-secret",
		"replace-with-a-long-random-secret",
	} {
		if _, err := NewService(store, secret, time.Hour); err == nil {
			t.Fatalf("expected JWT secret %q to be rejected", secret)
		}
	}
}
