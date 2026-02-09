package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Queries struct {
	db *sql.DB
}

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, email, name, role FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Role); err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Queries) GetUser(ctx context.Context, id int64) (User, error) {
	var user User
	row := q.db.QueryRowContext(ctx, `SELECT id, email, name, role FROM users WHERE id = ?`, id)
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Role); err != nil {
		return User{}, err
	}
	return user, nil
}

func (q *Queries) CreateUser(ctx context.Context, email, name, role string) (User, error) {
	var user User
	row := q.db.QueryRowContext(
		ctx,
		`INSERT INTO users (email, name, role) VALUES (?, ?, ?) RETURNING id, email, name, role`,
		email,
		name,
		role,
	)
	if err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Role); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (q *Queries) ListStates(ctx context.Context) ([]State, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, code, name FROM states ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]State, 0)
	for rows.Next() {
		var state State
		if err := rows.Scan(&state.ID, &state.Code, &state.Name); err != nil {
			return nil, err
		}
		out = append(out, state)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Queries) GetStateByCode(ctx context.Context, code string) (State, error) {
	var state State
	row := q.db.QueryRowContext(ctx, `SELECT id, code, name FROM states WHERE code = ?`, code)
	if err := row.Scan(&state.ID, &state.Code, &state.Name); err != nil {
		return State{}, err
	}
	return state, nil
}

func (q *Queries) CreateState(ctx context.Context, code, name string) (State, error) {
	var state State
	row := q.db.QueryRowContext(
		ctx,
		`INSERT INTO states (code, name) VALUES (?, ?) RETURNING id, code, name`,
		code,
		name,
	)
	if err := row.Scan(&state.ID, &state.Code, &state.Name); err != nil {
		return State{}, fmt.Errorf("create state: %w", err)
	}
	return state, nil
}
