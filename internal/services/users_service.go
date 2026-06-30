package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicatedEmail = errors.New("email already registered")
)

type UsersService struct {
	p *pgxpool.Pool
	q *pgstore.Queries
}

func NewUsersService(p *pgxpool.Pool) *UsersService {
	return &UsersService{p: p, q: pgstore.New(p)}
}

func (us *UsersService) CreateUser(ctx context.Context, name, email string) error {
	u := pgstore.CreateUserParams{Name: name, Email: email}

	if _, err := us.q.CreateUser(ctx, u); err != nil {
		return err
	}
	return nil
}

func (us *UsersService) GetUserById(ctx context.Context, id uuid.UUID) (pgstore.User, error) {
	u, err := us.q.GetUserById(ctx, id)
	if err != nil {
		return pgstore.User{}, err
	}
	return pgstore.User{ID: u.ID, Name: u.Name, Email: u.Email}, nil
}

func (us *UsersService) GetAllUsers(ctx context.Context) ([]pgstore.User, error) {
	u, err := us.q.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (us *UsersService) UpdateUser(ctx context.Context, id uuid.UUID, name, email string) (pgstore.User, error) {
	u, err := us.q.UpdateUser(ctx, pgstore.UpdateUserParams{ID: id, Name: name, Email: email})
	if err != nil {
		return pgstore.User{}, err
	}
	return pgstore.User{ID: u.ID, Name: u.Name, Email: u.Email}, nil
}

func (us *UsersService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := us.q.DeleteUser(ctx, id); err != nil {
		return err
	}
	return nil
}
