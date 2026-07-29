package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

var ErrBidIsTooLow = errors.New("bid is too low")

// bidsStore is the slice of pgstore.Queries that BidsService actually needs,
// narrow enough that tests can fake it directly without a real DB.
type bidsStore interface {
	GetProductByID(ctx context.Context, id uuid.UUID) (pgstore.Product, error)
	GetHighestBidByProductID(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error)
	CreateBid(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error)
}

type BidsService struct {
	q bidsStore
}

func NewBidsService(q bidsStore) BidsService {
	return BidsService{q: q}
}

func (bs BidsService) PlaceBid(
	ctx context.Context, product_id, user_id uuid.UUID, amount_cents int64,
) (pgstore.Bid, error) {
	product, err := bs.q.GetProductByID(ctx, product_id)
	if err != nil {
		return pgstore.Bid{}, err
	}

	highestBid, err := bs.q.GetHighestBidByProductID(ctx, product_id)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return pgstore.Bid{}, err
		}
	}

	if product.BasePriceCents >= amount_cents || highestBid.AmountCents >= amount_cents {
		return pgstore.Bid{}, ErrBidIsTooLow
	}

	highestBid, err = bs.q.CreateBid(context.Background(), pgstore.CreateBidParams{
		ProductID:   product_id,
		UserID:      user_id,
		AmountCents: amount_cents,
	})

	if err != nil {
		return pgstore.Bid{}, err
	}

	return highestBid, nil
}
