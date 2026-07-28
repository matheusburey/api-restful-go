package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicatedEmail    = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInternal           = errors.New("internal server error")
)

// usersStore is the slice of pgstore.Queries that UsersService actually needs,
// narrow enough that tests can fake it directly without going through pgx.
type usersStore interface {
	CreateUser(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error)
	GetUserByEmail(ctx context.Context, email string) (pgstore.User, error)
	UpdateUser(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type UsersService struct {
	q usersStore
}

func NewUsersService(q usersStore) UsersService {
	return UsersService{q: q}
}

func (us *UsersService) CreateUser(ctx context.Context, name, email, bio, password string) (uuid.UUID, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.UUID{}, err
	}

	args := pgstore.CreateUserParams{Name: name, Email: email, PasswordHash: hash, Bio: bio}
	userId, err := us.q.CreateUser(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.UUID{}, ErrDuplicatedEmail
		}
		return uuid.UUID{}, err
	}
	return userId, nil
}

func (us *UsersService) AuthenticateUser(ctx context.Context, email, password string) (uuid.UUID, error) {
	u, err := us.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, ErrInvalidCredentials
		}
		return uuid.UUID{}, ErrInternal
	}

	err = bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return uuid.UUID{}, ErrInvalidCredentials
		}
		return uuid.UUID{}, ErrInternal
	}

	return u.ID, nil
}

func (us *UsersService) UpdateUser(ctx context.Context, id uuid.UUID, name, email, bio string, password *string) (pgstore.User, error) {
	new_u := pgstore.UpdateUserParams{ID: id, Name: name, Email: email, Bio: bio}
	if password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return pgstore.User{}, err
		}
		new_u.PasswordHash = hash
	}
	u, err := us.q.UpdateUser(ctx, new_u)
	if err != nil {
		return pgstore.User{}, err
	}
	u.PasswordHash = nil
	return u, nil
}

func (us *UsersService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := us.q.DeleteUser(ctx, id); err != nil {
		return err
	}
	return nil
}
