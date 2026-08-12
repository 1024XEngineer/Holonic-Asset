package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const issuer = "holonic-asset"

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
)

const (
	minimumJWTSecretLength = 32
	jwtSecretPlaceholder   = "replace-with-a-long-random-secret"
)

type User struct {
	ID           uint
	Username     string
	PasswordHash string
	Email        string
}

type Store interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginResult struct {
	AccessToken string
	ExpiresIn   int64
	User        User
}

type Manager interface {
	Login(ctx context.Context, username string, password string) (*LoginResult, error)
	VerifyToken(token string) (*Claims, error)
}

type Service struct {
	store       Store
	secret      []byte
	tokenExpiry time.Duration
	now         func() time.Time
}

func NewService(store Store, jwtSecret string, tokenExpiry time.Duration) (*Service, error) {
	if store == nil {
		return nil, errors.New("auth: user store is required")
	}
	jwtSecret = strings.TrimSpace(jwtSecret)
	if jwtSecret == "" {
		return nil, errors.New("auth: JWT secret is required")
	}
	if jwtSecret == jwtSecretPlaceholder {
		return nil, errors.New("auth: JWT secret must be replaced")
	}
	if len([]byte(jwtSecret)) < minimumJWTSecretLength {
		return nil, fmt.Errorf("auth: JWT secret must be at least %d bytes", minimumJWTSecretLength)
	}
	if tokenExpiry <= 0 {
		return nil, errors.New("auth: token expiry must be positive")
	}
	return &Service{
		store:       store,
		secret:      []byte(jwtSecret),
		tokenExpiry: tokenExpiry,
		now:         time.Now,
	}, nil
}

func (s *Service) Login(ctx context.Context, username string, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.store.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: find user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	expiresAt := now.Add(s.tokenExpiry)
	claims := Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("auth: sign token: %w", err)
	}
	return &LoginResult{
		AccessToken: token,
		ExpiresIn:   int64(s.tokenExpiry / time.Second),
		User:        *user,
	}, nil
}

func (s *Service) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("auth: unexpected signing method %q", token.Method.Alg())
			}
			return s.secret, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid || claims.Subject == "" {
		return nil, ErrInvalidCredentials
	}
	return claims, nil
}

var _ Manager = (*Service)(nil)
