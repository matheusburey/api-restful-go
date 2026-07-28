package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore/pgstoretest"
)

func TestHandlerCreateProduct(t *testing.T) {
	validPayload := map[string]any{
		"name":             "Relógio Vintage",
		"description":      "um relógio muito bonito e antigo",
		"base_price_cents": 5000,
		"auction_end":      time.Now().Add(3 * time.Hour),
	}

	t.Run("invalid payload returns bad request without touching the store", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				CreateProductFn: func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
					t.Fatal("CreateProduct should not be called when payload validation fails")
					return uuid.UUID{}, nil
				},
			}),
			AuctionLobby: services.AuctionLobby{Rooms: make(map[uuid.UUID]*services.AuctionRoom)},
		}

		invalidPayload := map[string]any{
			"name":             "",
			"description":      "",
			"base_price_cents": 0,
			"auction_end":      time.Now(),
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerCreateProduct)).
			ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/products", invalidPayload))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
		}
	})

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				CreateProductFn: func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
					t.Fatal("CreateProduct should not be called for an unauthenticated request")
					return uuid.UUID{}, nil
				},
			}),
			AuctionLobby: services.AuctionLobby{Rooms: make(map[uuid.UUID]*services.AuctionRoom)},
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerCreateProduct)).
			ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/products", validPayload))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body)
		}
	})

	t.Run("success starts an auction room for the product", func(t *testing.T) {
		sessions := scs.New()
		sellerID := uuid.New()
		wantID := uuid.New()
		a := Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				CreateProductFn: func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
					if arg.SellerID != sellerID {
						t.Fatalf("got seller id %v, want %v", arg.SellerID, sellerID)
					}
					return wantID, nil
				},
			}),
			AuctionLobby: services.AuctionLobby{Rooms: make(map[uuid.UUID]*services.AuctionRoom)},
		}

		req := newAuthenticatedRequest(t, sessions, sellerID, "POST", "/api/v1/products", validPayload)
		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerCreateProduct)).ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
		}

		a.AuctionLobby.Lock()
		_, ok := a.AuctionLobby.Rooms[wantID]
		a.AuctionLobby.Unlock()
		if !ok {
			t.Fatal("expected an auction room to be registered for the created product")
		}
	})

	t.Run("unexpected service error maps to internal server error", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				CreateProductFn: func(ctx context.Context, arg pgstore.CreateProductParams) (uuid.UUID, error) {
					return uuid.UUID{}, errors.New("connection reset")
				},
			}),
			AuctionLobby: services.AuctionLobby{Rooms: make(map[uuid.UUID]*services.AuctionRoom)},
		}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "POST", "/api/v1/products", validPayload)
		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerCreateProduct)).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body)
		}
	})
}
