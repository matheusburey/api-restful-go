package pgstoretest

import (
	"context"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

// FakeProductStore fakes the subset of pgstore.Queries' product methods that
// services.ProductService depends on. Set only the *Fn fields exercised by a
// given test case; leaving the others nil means a call to them panics,
// surfacing an unexpected call.
type FakeProductStore struct {
	CreateProductFn  func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error)
	GetProductByIDFn func(ctx context.Context, id uuid.UUID) (pgstore.Product, error)
}

func (f FakeProductStore) CreateProduct(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
	return f.CreateProductFn(ctx, arg)
}

func (f FakeProductStore) GetProductByID(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
	return f.GetProductByIDFn(ctx, id)
}
