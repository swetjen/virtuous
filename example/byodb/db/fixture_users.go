package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/swetjen/virtuous/example/byodb/auth"
)

type FixtureUserCredential struct {
	Email    string
	Role     string
	Password string
}

func SeedFixtureUsers(ctx context.Context, queries *Queries) ([]FixtureUserCredential, error) {
	if queries == nil {
		return nil, errors.New("queries is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	existing, err := queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, nil
	}

	fixtures := []struct {
		Email string
		Name  string
		Role  string
	}{
		{Email: "admin@virtuous.dev", Name: "Virtuous Admin", Role: "admin"},
		{Email: "user@virtuous.dev", Name: "Virtuous User", Role: "user"},
	}

	created := make([]FixtureUserCredential, 0, len(fixtures))
	for _, fixture := range fixtures {
		password, genErr := randomCredential(12)
		if genErr != nil {
			return created, genErr
		}
		passwordHash, hashErr := auth.HashPassword(password)
		if hashErr != nil {
			return created, hashErr
		}

		_, createErr := queries.CreateUserWithPassword(ctx, CreateUserWithPasswordParams{
			Email:        fixture.Email,
			Name:         fixture.Name,
			Role:         fixture.Role,
			PasswordHash: passwordHash,
			Confirmed:    true,
			ConfirmCode:  "",
		})
		if createErr != nil {
			var pgErr *pgconn.PgError
			if errors.As(createErr, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			return created, createErr
		}

		created = append(created, FixtureUserCredential{
			Email:    fixture.Email,
			Role:     fixture.Role,
			Password: password,
		})
	}

	return created, nil
}
