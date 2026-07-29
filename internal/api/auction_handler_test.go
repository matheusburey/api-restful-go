package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore/pgstoretest"
)

// subscribeRouter mounts HandlerSubscribeUserToAuction on a real chi router
// (needed so chi.URLParam(r, "product_id") resolves) behind the same
// session + auth middleware chain routes.go uses.
func subscribeRouter(a *Api) http.Handler {
	r := chi.NewRouter()
	r.With(a.AuthMiddleware).Get("/ws/subscribe/{product_id}", a.HandlerSubscribeUserToAuction)
	return a.Sessions.LoadAndSave(r)
}

func TestHandlerSubscribeUserToAuction(t *testing.T) {
	t.Run("invalid product id", func(t *testing.T) {
		sessions := scs.New()
		a := &Api{Sessions: sessions}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "GET", "/ws/subscribe/not-a-uuid", nil)
		rec := httptest.NewRecorder()
		subscribeRouter(a).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
		}
	})

	t.Run("product not found", func(t *testing.T) {
		sessions := scs.New()
		a := &Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
					return pgstore.Product{}, services.ErrProductNotFound
				},
			}),
		}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "GET", "/ws/subscribe/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		subscribeRouter(a).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body)
		}
	})

	t.Run("unexpected product service error maps to internal server error", func(t *testing.T) {
		sessions := scs.New()
		a := &Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
					return pgstore.Product{}, errors.New("connection reset")
				},
			}),
		}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "GET", "/ws/subscribe/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		subscribeRouter(a).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body)
		}
	})

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		sessions := scs.New()
		a := &Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
					return pgstore.Product{}, nil
				},
			}),
		}

		rec := httptest.NewRecorder()
		subscribeRouter(a).ServeHTTP(rec, httptest.NewRequest("GET", "/ws/subscribe/"+uuid.New().String(), nil))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body)
		}
	})

	t.Run("auction room not found once authenticated", func(t *testing.T) {
		sessions := scs.New()
		productID := uuid.New()
		a := &Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
					return pgstore.Product{}, nil
				},
			}),
			AuctionLobby: services.AuctionLobby{Rooms: make(map[uuid.UUID]*services.AuctionRoom)},
		}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "GET", "/ws/subscribe/"+productID.String(), nil)
		rec := httptest.NewRecorder()
		subscribeRouter(a).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body)
		}
	})

	t.Run("success: connects over websocket and places a bid", func(t *testing.T) {
		sessions := scs.New()
		productID := uuid.New()
		userID := uuid.New()
		product := pgstore.Product{ID: productID, BasePriceCents: 1000}

		bidsService := services.NewBidsService(pgstoretest.FakeBidsStore{
			GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
				return product, nil
			},
			GetHighestBidByProductIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Bid, error) {
				return pgstore.Bid{}, pgx.ErrNoRows
			},
			CreateBidFn: func(ctx context.Context, arg pgstore.CreateBidParams) (pgstore.Bid, error) {
				return pgstore.Bid{ID: uuid.New(), ProductID: productID, UserID: userID, AmountCents: arg.AmountCents}, nil
			},
		})

		// The room's context is intentionally never cancelled: Run() keeps a
		// single goroutine idling until the test binary exits, same tradeoff
		// already accepted for HandlerCreateProduct's success test.
		room := services.NewAuctionRoom(context.Background(), productID, bidsService)
		go room.Run()

		a := &Api{
			Sessions: sessions,
			ProductService: services.NewProductService(pgstoretest.FakeProductStore{
				GetProductByIDFn: func(ctx context.Context, id uuid.UUID) (pgstore.Product, error) {
					return product, nil
				},
			}),
			AuctionLobby: services.AuctionLobby{Rooms: map[uuid.UUID]*services.AuctionRoom{productID: room}},
			WsUpgrade:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		}

		ts := httptest.NewServer(subscribeRouter(a))
		t.Cleanup(ts.Close)

		target := "/ws/subscribe/" + productID.String()
		authReq := newAuthenticatedRequest(t, sessions, userID, "GET", target, nil)

		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + target
		header := http.Header{"Cookie": authReq.Header["Cookie"]}

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}
		t.Cleanup(func() { conn.Close() })

		if err := conn.WriteJSON(services.Message{Kind: services.PlaceBid, Amount: 2000}); err != nil {
			t.Fatalf("failed to write bid message: %v", err)
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var got services.Message
		if err := conn.ReadJSON(&got); err != nil {
			t.Fatalf("failed to read response message: %v", err)
		}
		if got.Kind != services.SuccessfulBid {
			t.Fatalf("got message kind %v, want SuccessfulBid (%v): %+v", got.Kind, services.SuccessfulBid, got)
		}
	})
}
