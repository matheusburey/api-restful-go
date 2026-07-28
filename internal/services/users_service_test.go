package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore/pgstoretest"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		wantID := uuid.New()
		store := pgstoretest.FakeUsersStore{
			CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
				return wantID, nil
			},
		}
		us := NewUsersService(store)

		id, err := us.CreateUser(context.Background(), "Teste", "teste@email.com", "bio teste", "Abc123@@")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != wantID {
			t.Fatalf("got id %v, want %v", id, wantID)
		}
	})

	t.Run("duplicated email", func(t *testing.T) {
		store := pgstoretest.FakeUsersStore{
			CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
				return uuid.UUID{}, &pgconn.PgError{Code: "23505"}
			},
		}
		us := NewUsersService(store)

		_, err := us.CreateUser(context.Background(), "Teste", "teste@email.com", "bio teste", "Abc123@@")
		if !errors.Is(err, ErrDuplicatedEmail) {
			t.Fatalf("got err %v, want ErrDuplicatedEmail", err)
		}
	})

	t.Run("unexpected db error is propagated", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		store := pgstoretest.FakeUsersStore{
			CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
				return uuid.UUID{}, wantErr
			},
		}
		us := NewUsersService(store)

		_, err := us.CreateUser(context.Background(), "Teste", "teste@email.com", "bio teste", "Abc123@@")
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})
}

func TestAuthenticateUser(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Abc123@@"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	storedUser := pgstore.User{ID: uuid.New(), Email: "teste@email.com", PasswordHash: hash}

	t.Run("success", func(t *testing.T) {
		store := pgstoretest.FakeUsersStore{
			GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
				return storedUser, nil
			},
		}
		us := NewUsersService(store)

		id, err := us.AuthenticateUser(context.Background(), storedUser.Email, "Abc123@@")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != storedUser.ID {
			t.Fatalf("got id %v, want %v", id, storedUser.ID)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		store := pgstoretest.FakeUsersStore{
			GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
				return storedUser, nil
			},
		}
		us := NewUsersService(store)

		_, err := us.AuthenticateUser(context.Background(), storedUser.Email, "wrong-password")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("got err %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := pgstoretest.FakeUsersStore{
			GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
				return pgstore.User{}, pgx.ErrNoRows
			},
		}
		us := NewUsersService(store)

		_, err := us.AuthenticateUser(context.Background(), "nobody@email.com", "Abc123@@")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("got err %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unexpected db error maps to internal error", func(t *testing.T) {
		store := pgstoretest.FakeUsersStore{
			GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
				return pgstore.User{}, errors.New("connection reset")
			},
		}
		us := NewUsersService(store)

		_, err := us.AuthenticateUser(context.Background(), storedUser.Email, "Abc123@@")
		if !errors.Is(err, ErrInternal) {
			t.Fatalf("got err %v, want ErrInternal", err)
		}
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("success strips password hash from response", func(t *testing.T) {
		updated := pgstore.User{ID: uuid.New(), Name: "Novo Nome", Email: "novo@email.com", Bio: "nova bio", PasswordHash: []byte("hash")}
		store := pgstoretest.FakeUsersStore{
			UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
				return updated, nil
			},
		}
		us := NewUsersService(store)

		u, err := us.UpdateUser(context.Background(), updated.ID, updated.Name, updated.Email, updated.Bio, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Name != updated.Name || u.Email != updated.Email {
			t.Fatalf("got user %+v, want name/email %s/%s", u, updated.Name, updated.Email)
		}
		if u.PasswordHash != nil {
			t.Fatalf("expected password hash to be stripped from response, got %v", u.PasswordHash)
		}
	})

	t.Run("password change hashes the new password before saving", func(t *testing.T) {
		newPassword := "N3wPassw0rd!"
		var gotArg pgstore.UpdateUserParams
		store := pgstoretest.FakeUsersStore{
			UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
				gotArg = arg
				return pgstore.User{ID: arg.ID, Name: arg.Name, Email: arg.Email, Bio: arg.Bio}, nil
			},
		}
		us := NewUsersService(store)

		_, err := us.UpdateUser(context.Background(), uuid.New(), "n", "e", "b", &newPassword)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := bcrypt.CompareHashAndPassword(gotArg.PasswordHash, []byte(newPassword)); err != nil {
			t.Fatalf("stored hash does not match new password: %v", err)
		}
	})

	t.Run("db error is propagated", func(t *testing.T) {
		wantErr := errors.New("write failed")
		store := pgstoretest.FakeUsersStore{
			UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
				return pgstore.User{}, wantErr
			},
		}
		us := NewUsersService(store)

		_, err := us.UpdateUser(context.Background(), uuid.New(), "n", "e", "b", nil)
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		called := false
		store := pgstoretest.FakeUsersStore{
			DeleteUserFn: func(ctx context.Context, id uuid.UUID) error {
				called = true
				return nil
			},
		}
		us := NewUsersService(store)

		if err := us.DeleteUser(context.Background(), uuid.New()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("expected DeleteUser to be called")
		}
	})

	t.Run("db error is propagated", func(t *testing.T) {
		wantErr := errors.New("delete failed")
		store := pgstoretest.FakeUsersStore{
			DeleteUserFn: func(ctx context.Context, id uuid.UUID) error {
				return wantErr
			},
		}
		us := NewUsersService(store)

		err := us.DeleteUser(context.Background(), uuid.New())
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})
}
