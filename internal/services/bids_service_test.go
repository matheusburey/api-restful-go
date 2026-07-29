package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore/pgstoretest"
)

func TestPlaceBid(t *testing.T) {
	productID := uuid.New()
	userID := uuid.New()
	product := pgstore.Product{ID: productID, BasePriceCents: 1000}

	t.Run("success on the first bid for a product", func(t *testing.T) {
		wantBid := pgstore.Bid{ID: uuid.New(), ProductID: productID, UserID: userID, AmountCents: 1500}
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
				return pgstore.Bid{}, pgx.ErrNoRows
			},
			CreateBidFn: func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
				return wantBid, nil
			},
		}
		s := NewBidsService(store)

		bid, err := s.PlaceBid(context.Background(), productID, userID, 1500)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bid != wantBid {
			t.Fatalf("got bid %+v, want %+v", bid, wantBid)
		}
	})

	t.Run("success outbidding an existing highest bid", func(t *testing.T) {
		existingBid := pgstore.Bid{AmountCents: 1500}
		wantBid := pgstore.Bid{ID: uuid.New(), ProductID: productID, UserID: userID, AmountCents: 2000}
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
				return existingBid, nil
			},
			CreateBidFn: func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
				if arg.AmountCents != 2000 {
					t.Fatalf("got amount %d, want 2000", arg.AmountCents)
				}
				return wantBid, nil
			},
		}
		s := NewBidsService(store)

		bid, err := s.PlaceBid(context.Background(), productID, userID, 2000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bid != wantBid {
			t.Fatalf("got bid %+v, want %+v", bid, wantBid)
		}
	})

	t.Run("bid at or below the base price is rejected", func(t *testing.T) {
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
				return pgstore.Bid{}, pgx.ErrNoRows
			},
			CreateBidFn: func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
				t.Fatal("CreateBid should not be called when the bid is too low")
				return pgstore.Bid{}, nil
			},
		}
		s := NewBidsService(store)

		_, err := s.PlaceBid(context.Background(), productID, userID, product.BasePriceCents)
		if !errors.Is(err, ErrBidIsTooLow) {
			t.Fatalf("got err %v, want ErrBidIsTooLow", err)
		}
	})

	t.Run("bid at or below the current highest bid is rejected", func(t *testing.T) {
		existingBid := pgstore.Bid{AmountCents: 1500}
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
				return existingBid, nil
			},
			CreateBidFn: func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
				t.Fatal("CreateBid should not be called when the bid is too low")
				return pgstore.Bid{}, nil
			},
		}
		s := NewBidsService(store)

		_, err := s.PlaceBid(context.Background(), productID, userID, existingBid.AmountCents)
		if !errors.Is(err, ErrBidIsTooLow) {
			t.Fatalf("got err %v, want ErrBidIsTooLow", err)
		}
	})

	t.Run("product lookup error is propagated", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return pgstore.Product{}, wantErr
			},
		}
		s := NewBidsService(store)

		_, err := s.PlaceBid(context.Background(), productID, userID, 2000)
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})

	t.Run("unexpected error fetching the highest bid is propagated", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
				return pgstore.Bid{}, wantErr
			},
		}
		s := NewBidsService(store)

		_, err := s.PlaceBid(context.Background(), productID, userID, 2000)
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})

	t.Run("db error creating the bid is propagated", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		store := pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, productID uuid.UUID) (pgstore.Bid, error) {
				return pgstore.Bid{}, pgx.ErrNoRows
			},
			CreateBidFn: func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
				return pgstore.Bid{}, wantErr
			},
		}
		s := NewBidsService(store)

		_, err := s.PlaceBid(context.Background(), productID, userID, 2000)
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})
}
