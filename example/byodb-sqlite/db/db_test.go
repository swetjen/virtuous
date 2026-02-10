package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestQueriesUsers(t *testing.T) {
	queries, pool, err := NewTest()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer pool.Close()

	created, err := queries.CreateUser(context.Background(), CreateUserParams{
		Email: "dev@example.com",
		Name:  "Dev",
		Role:  "admin",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected user id")
	}

	got, err := queries.GetUser(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Email != created.Email {
		t.Fatalf("expected email %q got %q", created.Email, got.Email)
	}

	users, err := queries.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
}

func TestQueriesStates(t *testing.T) {
	queries, pool, err := NewTest()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer pool.Close()

	created, err := queries.CreateState(context.Background(), CreateStateParams{
		Code: "ZZ",
		Name: "Zeta State",
	})
	if err != nil {
		t.Fatalf("create state: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected state id")
	}

	got, err := queries.GetStateByCode(context.Background(), "ZZ")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.Name != created.Name {
		t.Fatalf("expected name %q got %q", created.Name, got.Name)
	}

	states, err := queries.ListStates(context.Background())
	if err != nil {
		t.Fatalf("list states: %v", err)
	}
	if len(states) < 51 {
		t.Fatalf("expected at least 51 states, got %d", len(states))
	}
	found := false
	for _, state := range states {
		if state.Code == created.Code {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find created state %q in list", created.Code)
	}
}

func TestQueriesGetMissing(t *testing.T) {
	queries, pool, err := NewTest()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer pool.Close()

	_, err = queries.GetUser(context.Background(), 999)
	if err == nil {
		t.Fatalf("expected error for missing user")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
