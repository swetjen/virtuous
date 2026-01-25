package db

import (
	"errors"
	"sync"
)

type User struct {
	ID    int64
	Email string
	Name  string
	Role  string
}

type Store struct {
	mu     sync.RWMutex
	users  []User
	nextID int64
}

func New() *Store {
	return &Store{
		users: []User{
			{ID: 1, Email: "admin@example.com", Name: "Admin User", Role: "admin"},
			{ID: 2, Email: "support@example.com", Name: "Support User", Role: "support"},
		},
		nextID: 3,
	}
}

func (s *Store) ListUsers() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, len(s.users))
	copy(out, s.users)
	return out
}

func (s *Store) GetUser(id int64) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.users {
		if user.ID == id {
			return user, true
		}
	}
	return User{}, false
}

func (s *Store) CreateUser(user User) (User, error) {
	if user.Email == "" || user.Name == "" || user.Role == "" {
		return User{}, errors.New("email, name, and role are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user.ID = s.nextID
	s.nextID++
	s.users = append(s.users, user)
	return user, nil
}
