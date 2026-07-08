package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

var ErrBidIsTooLow = errors.New("bid is too low")

type BidsService struct {
	p *pgxpool.Pool
	q *pgstore.Queries
}

func NewBidsService(p *pgxpool.Pool) BidsService {
	return BidsService{p: p, q: pgstore.New(p)}
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
