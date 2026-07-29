package pgstoretest

import (
	"context"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

// FakeBidsStore fakes the subset of pgstore.Queries' product/bid methods
// that services.BidsService depends on. Set only the *Fn fields exercised by
// a given test case; leaving the others nil means a call to them panics,
// surfacing an unexpected call.
type FakeBidsStore struct {
	GetProductByIDFn           func(ctx context.Context, id uuid.UUID) (pgstore.Product, error)
	GetHighestBidByProductIDFn func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error)
	CreateBidFn                func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error)
}

func (f FakeBidsStore) GetProductByID(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
	return f.GetProductByIDFn(ctx, id)
}

func (f FakeBidsStore) GetHighestBidByProductID(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
	return f.GetHighestBidByProductIDFn(ctx, productID)
}

func (f FakeBidsStore) CreateBid(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
	return f.CreateBidFn(ctx, arg)
}
