package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const testJWTSecret = "test-jwt-secret-0123456789abcdef"

type memoryStore struct {
	users map[string]*User
	err   error
}

func (s *memoryStore) FindByUsername(_ context.Context, username string) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
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
		{username: " ", password: "login-test-password"},
		{username: "login-test-user", password: ""},
	} {
		if _, err := service.Login(context.Background(), test.username, test.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials for %+v, got %v", test, err)
		}
	}
}

func TestServiceLoginWrapsStoreErrors(t *testing.T) {
	storeErr := errors.New("database unavailable")
	service, err := NewService(&memoryStore{err: storeErr}, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	_, err = service.Login(context.Background(), "login-test-user", "login-test-password")
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected wrapped store error, got %v", err)
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

func TestNewServiceRejectsInvalidConfiguration(t *testing.T) {
	store := &memoryStore{users: make(map[string]*User)}
	for _, test := range []struct {
		name        string
		store       Store
		secret      string
		tokenExpiry time.Duration
	}{
		{name: "missing store", secret: testJWTSecret, tokenExpiry: time.Hour},
		{name: "missing secret", store: store, tokenExpiry: time.Hour},
		{name: "non-positive expiry", store: store, secret: testJWTSecret},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewService(test.store, test.secret, test.tokenExpiry); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestServiceVerifyTokenRejectsUnexpectedMethodAndMissingSubject(t *testing.T) {
	service, err := NewService(&memoryStore{users: make(map[string]*User)}, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	now := time.Now()

	for _, test := range []struct {
		name   string
		method jwt.SigningMethod
		claims Claims
	}{
		{
			name:   "unexpected signing method",
			method: jwt.SigningMethodHS384,
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "1", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}},
		},
		{
			name:   "missing subject",
			method: jwt.SigningMethodHS256,
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(test.method, test.claims).SignedString([]byte(testJWTSecret))
			if err != nil {
				t.Fatalf("sign token: %v", err)
			}
			if _, err := service.VerifyToken(token); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("expected invalid credentials, got %v", err)
			}
		})
	}
}
