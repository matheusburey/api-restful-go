package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore/pgstoretest"
)

func TestCreateProduct(t *testing.T) {
	sellerID := uuid.New()
	auctionEnd := time.Now().Add(3 * time.Hour)

	t.Run("success", func(t *testing.T) {
		wantID := uuid.New()
		store := pgstoretest.FakeProductStore{
			CreateProductFn: func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
				return wantID, nil
			},
		}
		s := NewProductService(store)

		id, err := s.CreateProduct(context.Background(), sellerID, "Relógio", "descrição do produto", 5000, auctionEnd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != wantID {
			t.Fatalf("got id %v, want %v", id, wantID)
		}
	})

	t.Run("db error is propagated", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		store := pgstoretest.FakeProductStore{
			CreateProductFn: func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
				return uuid.UUID{}, wantErr
			},
		}
		s := NewProductService(store)

		_, err := s.CreateProduct(context.Background(), sellerID, "Relógio", "descrição do produto", 5000, auctionEnd)
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})
}

func TestGetProductByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := pgstore.Product{ID: uuid.New(), Name: "Relógio", BasePriceCents: 5000}
		store := pgstoretest.FakeProductStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return want, nil
			},
		}
		s := NewProductService(store)

		got, err := s.GetProductByID(context.Background(), want.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != want.ID || got.Name != want.Name {
			t.Fatalf("got product %+v, want %+v", got, want)
		}
	})

	t.Run("not found maps to ErrProductNotFound", func(t *testing.T) {
		store := pgstoretest.FakeProductStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return pgstore.Product{}, pgx.ErrNoRows
			},
		}
		s := NewProductService(store)

		_, err := s.GetProductByID(context.Background(), uuid.New())
		if !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("got err %v, want ErrProductNotFound", err)
		}
	})

	t.Run("unexpected db error is propagated", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		store := pgstoretest.FakeProductStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return pgstore.Product{}, wantErr
			},
		}
		s := NewProductService(store)

		_, err := s.GetProductByID(context.Background(), uuid.New())
		if !errors.Is(err, wantErr) {
			t.Fatalf("got err %v, want %v", err, wantErr)
		}
	})
}
