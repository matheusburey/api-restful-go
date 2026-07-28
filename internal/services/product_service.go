package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
)

var (
	ErrProductNotFound = errors.New("product not found")
)

// productStore is the slice of pgstore.Queries that ProductService actually
// needs, narrow enough that tests can fake it directly without a real DB.
type productStore interface {
	CreateProduct(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (pgstore.Product, error)
}

type ProductService struct {
	q productStore
}

func NewProductService(q productStore) ProductService {
	return ProductService{q: q}
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

func (s ProductService) GetProductByID(ctx context.Context, product_id uuid.UUID) (pgstore.Product, error) {
	product, err := s.q.GetProductByID(ctx, product_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgstore.Product{}, ErrProductNotFound
		}
		return pgstore.Product{}, err
	}

	return product, nil
}
