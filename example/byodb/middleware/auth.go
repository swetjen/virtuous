package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	sessionauth "github.com/swetjen/virtuous/example/byodb/auth"
	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/rpc"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

type Auth interface {
	HasSignedIn() rpc.Guard
	HasSignedInAdmin() rpc.Guard

	SignToken(userID int64, role string) (string, error)
	ParseToken(token string) (sessionauth.Claims, error)
	GetClaims(ctx context.Context) (sessionauth.Claims, bool)

	HashPassword(password string) (string, error)
	VerifyPassword(hash, password string) bool
}

type authService struct {
	secret   string
	queries  *db.Queries
	tokenTTL time.Duration
}

func NewAuthService(cfg config.Config, queries *db.Queries) Auth {
	tokenTTL := time.Duration(cfg.AuthTokenTTL) * time.Second
	if tokenTTL <= 0 {
		tokenTTL = sessionauth.DefaultTokenTTL
	}
	return &authService{
		secret:   strings.TrimSpace(cfg.AuthTokenSecret),
		queries:  queries,
		tokenTTL: tokenTTL,
	}
}

func (a *authService) HasSignedIn() rpc.Guard {
	return sessionRPCGuard{
		name: "UserSessionAuth",
		mw:   a.sessionMiddleware(""),
	}
}

func (a *authService) HasSignedInAdmin() rpc.Guard {
	return sessionRPCGuard{
		name: "AdminSessionAuth",
		mw:   a.sessionMiddleware("admin"),
	}
}

func (a *authService) SignToken(userID int64, role string) (string, error) {
	return sessionauth.SignToken(a.secret, userID, role, a.tokenTTL)
}

func (a *authService) ParseToken(token string) (sessionauth.Claims, error) {
	return sessionauth.ParseToken(a.secret, token)
}

func (a *authService) GetClaims(ctx context.Context) (sessionauth.Claims, bool) {
	return sessionauth.ClaimsFromContext(ctx)
}

func (a *authService) HashPassword(password string) (string, error) {
	return sessionauth.HashPassword(password)
}

func (a *authService) VerifyPassword(hash, password string) bool {
	return sessionauth.VerifyPassword(hash, password)
}

func (a *authService) sessionMiddleware(requiredRole string) func(http.Handler) http.Handler {
	requiredRole = strings.TrimSpace(requiredRole)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get(authorizationHeader))
			if header == "" {
				http.Error(w, "missing auth token", http.StatusUnauthorized)
				return
			}
			if !strings.HasPrefix(header, bearerPrefix) {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
			claims, err := a.ParseToken(token)
			if err != nil {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}

			user, err := a.ensureActiveUser(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}
			if requiredRole != "" && !strings.EqualFold(user.Role, requiredRole) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			// DB role is the source of truth; refresh claim role for downstream handlers.
			claims.Role = user.Role
			ctx := sessionauth.ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *authService) ensureActiveUser(ctx context.Context, userID int64) (db.UserByIDWithAuthRow, error) {
	if userID <= 0 {
		return db.UserByIDWithAuthRow{}, errors.New("invalid user id")
	}
	if a.queries == nil {
		return db.UserByIDWithAuthRow{}, errors.New("auth database unavailable")
	}
	user, err := a.queries.UserByIDWithAuth(ctx, userID)
	if err != nil {
		return db.UserByIDWithAuthRow{}, err
	}
	if !user.Confirmed {
		return db.UserByIDWithAuthRow{}, errors.New("user not confirmed")
	}
	if user.Disabled {
		return db.UserByIDWithAuthRow{}, errors.New("user is disabled")
	}
	return user, nil
}

type sessionRPCGuard struct {
	name string
	mw   func(http.Handler) http.Handler
}

func (g sessionRPCGuard) Spec() rpc.GuardSpec {
	return rpc.GuardSpec{
		Name:   g.name,
		In:     "header",
		Param:  authorizationHeader,
		Prefix: "Bearer",
	}
}

func (g sessionRPCGuard) Middleware() func(http.Handler) http.Handler {
	return g.mw
}
