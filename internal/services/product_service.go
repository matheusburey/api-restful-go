package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

type ProductService struct {
	p *pgxpool.Pool
	q *pgstore.Queries
}

func NewProductService(p *pgxpool.Pool) ProductService {
	return ProductService{p: p, q: pgstore.New(p)}
}

func (s ProductService) CreateProduct(
	ctx context.Context, seller_id uuid.UUID, name, description string, basePriceCents int64, auctionEnd time.Time,
) (uuid.UUID, error) {
	product_id, err := s.q.CreateProduct(context.Background(), pgstore.CreateProductParams{
		SellerID:       seller_id,
		Name:           name,
		Description:    description,
		BasePriceCents: basePriceCents,
		AuctionEnd:     auctionEnd,
	})

	if err != nil {
		return uuid.UUID{}, err
	}

	return product_id, nil
}
