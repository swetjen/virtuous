package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const DefaultTokenTTL = 24 * time.Hour

type claimsContextKey struct{}

type Claims struct {
	UserID int64  `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", errors.New("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	if strings.TrimSpace(hash) == "" || strings.TrimSpace(password) == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func SignToken(secret string, userID int64, role string, ttl time.Duration) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errors.New("token secret is required")
	}
	if userID <= 0 {
		return "", errors.New("user id is required")
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return "", errors.New("role is required")
	}
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}

	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret string, tokenString string) (Claims, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return Claims{}, errors.New("token secret is required")
	}
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return Claims{}, errors.New("token is required")
	}

	parsed, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithLeeway(15*time.Second),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return Claims{}, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Claims{}, errors.New("invalid token claims")
	}
	if claims.UserID <= 0 {
		return Claims{}, errors.New("invalid token user")
	}
	if strings.TrimSpace(claims.Role) == "" {
		return Claims{}, errors.New("invalid token role")
	}
	return *claims, nil
}

func ContextWithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(Claims)
	if !ok {
		return Claims{}, false
	}
	return claims, true
}
