// Package pgstoretest provides reusable test doubles for pgstore's generated
// queries, so any caller (services, handlers, ...) can fake them without
// duplicating the same struct in every _test.go file.
package pgstoretest

import (
	"context"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

// FakeUsersStore fakes the subset of pgstore.Queries' user methods that
// services.UsersService depends on. Set only the *Fn fields exercised by a
// given test case; leaving the others nil means a call to them panics,
// surfacing an unexpected call.
type FakeUsersStore struct {
	CreateUserFn     func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error)
	GetUserByEmailFn func(ctx context.Context, email string) (pgstore.User, error)
	UpdateUserFn     func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error)
	DeleteUserFn     func(ctx context.Context, id uuid.UUID) error
}

func (f FakeUsersStore) CreateUser(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
	return f.CreateUserFn(ctx, arg)
}

func (f FakeUsersStore) GetUserByEmail(ctx context.Context, email string) (pgstore.User, error) {
	return f.GetUserByEmailFn(ctx, email)
}

func (f FakeUsersStore) UpdateUser(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
	return f.UpdateUserFn(ctx, arg)
}

func (f FakeUsersStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return f.DeleteUserFn(ctx, id)
}
