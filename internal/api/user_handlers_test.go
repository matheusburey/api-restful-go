package api

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore"
	"github.com/matheusburey/api-restful-go/internal/store/pgstore/pgstoretest"
	"github.com/matheusburey/api-restful-go/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// scs's default GobCodec needs concrete types stored in the session
// registered up front, same as gob.Register(uuid.UUID{}) in cmd/api/main.go.
func init() {
	gob.Register(uuid.UUID{})
}

func jsonRequest(t *testing.T, method, target string, payload any) *http.Request {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// newAuthenticatedRequest logs userID into sessions via a throwaway request,
// then attaches the resulting session cookie to a fresh request for target,
// so handlers reading api.Sessions.Get(ctx, "AuthenticatedUserID") see it set.
// payload may be nil for requests without a body.
func newAuthenticatedRequest(t *testing.T, sessions *scs.SessionManager, userID uuid.UUID, method, target string, payload any) *http.Request {
	t.Helper()

	seedRec := httptest.NewRecorder()
	sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessions.Put(r.Context(), "AuthenticatedUserID", userID)
	})).ServeHTTP(seedRec, httptest.NewRequest(method, target, nil))

	var req *http.Request
	if payload != nil {
		req = jsonRequest(t, method, target, payload)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	for _, c := range seedRec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) utils.Response {
	t.Helper()
	var resp utils.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return resp
}

func TestHandlerSignupUser(t *testing.T) {
	validPayload := map[string]string{
		"name":     "Teste",
		"email":    "teste@email.com",
		"password": "Abc123@@",
		"bio":      "bio teste de usuário",
	}

	fakeCreateUser := pgstoretest.FakeUsersStore{
		CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}

	t.Run("success", func(t *testing.T) {
		a := Api{UsersService: services.NewUsersService(fakeCreateUser)}

		rec := httptest.NewRecorder()
		http.HandlerFunc(a.HandlerSignupUser).ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/signup", validPayload))

		if rec.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
		}
		if resp := decodeResponse(t, rec); resp.Message != "success" {
			t.Fatalf("got message %q, want %q", resp.Message, "success")
		}
	})

	t.Run("invalid payload returns bad request without touching the store", func(t *testing.T) {
		a := Api{UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
			CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
				t.Fatal("CreateUser should not be called when payload validation fails")
				return uuid.UUID{}, nil
			},
		})}

		invalidPayload := map[string]string{"name": "", "email": "not-an-email", "password": "123", "bio": ""}

		rec := httptest.NewRecorder()
		http.HandlerFunc(a.HandlerSignupUser).ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/signup", invalidPayload))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
		}
	})

	t.Run("duplicated email", func(t *testing.T) {
		a := Api{UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
			CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
				return uuid.UUID{}, services.ErrDuplicatedEmail
			},
		})}

		rec := httptest.NewRecorder()
		http.HandlerFunc(a.HandlerSignupUser).ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/signup", validPayload))

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusUnprocessableEntity, rec.Body)
		}
	})

	t.Run("unexpected service error maps to internal server error", func(t *testing.T) {
		a := Api{UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
			CreateUserFn: func(ctx context.Context, arg pgstore.CreateUserParams) (uuid.UUID, error) {
				return uuid.UUID{}, errors.New("connection reset")
			},
		})}

		rec := httptest.NewRecorder()
		http.HandlerFunc(a.HandlerSignupUser).ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/signup", validPayload))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body)
		}
	})
}

func TestHandlerLoginUser(t *testing.T) {
	validPayload := map[string]string{"email": "teste@email.com", "password": "Abc123@@"}

	hash, err := bcrypt.GenerateFromPassword([]byte("Abc123@@"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	storedUser := pgstore.User{ID: uuid.New(), Email: "teste@email.com", PasswordHash: hash}

	t.Run("success renews the session and stores the authenticated user id", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
					return storedUser, nil
				},
			}),
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerLoginUser)).
			ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/login", validPayload))

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
		}
		if resp := decodeResponse(t, rec); resp.Message != "success" {
			t.Fatalf("got message %q, want %q", resp.Message, "success")
		}
		if len(rec.Result().Cookies()) == 0 {
			t.Fatal("expected a session cookie to be set on successful login")
		}
	})

	t.Run("invalid payload returns bad request without touching the store", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
					t.Fatal("AuthenticateUser should not be called when payload validation fails")
					return pgstore.User{}, nil
				},
			}),
		}

		invalidPayload := map[string]string{"email": "not-an-email", "password": ""}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerLoginUser)).
			ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/login", invalidPayload))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
					return pgstore.User{}, pgx.ErrNoRows
				},
			}),
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerLoginUser)).
			ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/login", validPayload))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body)
		}
	})

	t.Run("unexpected service error maps to internal server error", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				GetUserByEmailFn: func(ctx context.Context, email string) (pgstore.User, error) {
					return pgstore.User{}, errors.New("connection reset")
				},
			}),
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerLoginUser)).
			ServeHTTP(rec, jsonRequest(t, "POST", "/api/v1/users/login", validPayload))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body)
		}
	})
}

func TestHandlerLogoutUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sessions := scs.New()
		a := Api{Sessions: sessions}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerLogoutUser)).
			ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/users/logout", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
		}
		if resp := decodeResponse(t, rec); resp.Message != "success" {
			t.Fatalf("got message %q, want %q", resp.Message, "success")
		}
	})
}

func TestHandlerDeleteUser(t *testing.T) {
	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				DeleteUserFn: func(ctx context.Context, id uuid.UUID) error {
					t.Fatal("DeleteUser should not be called for an unauthenticated request")
					return nil
				},
			}),
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerDeleteUser)).
			ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/users", nil))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body)
		}
	})

	t.Run("success", func(t *testing.T) {
		sessions := scs.New()
		userID := uuid.New()
		var gotID uuid.UUID
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				DeleteUserFn: func(ctx context.Context, id uuid.UUID) error {
					gotID = id
					return nil
				},
			}),
		}

		req := newAuthenticatedRequest(t, sessions, userID, "DELETE", "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerDeleteUser)).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusNoContent, rec.Body)
		}
		if gotID != userID {
			t.Fatalf("DeleteUser called with id %v, want %v", gotID, userID)
		}
	})

	t.Run("service error maps to not found", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				DeleteUserFn: func(ctx context.Context, id uuid.UUID) error {
					return errors.New("connection reset")
				},
			}),
		}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "DELETE", "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerDeleteUser)).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body)
		}
	})
}

func TestHandlerUpdateUser(t *testing.T) {
	validPayload := map[string]string{"name": "Novo Nome", "email": "novo@email.com", "bio": "bio atualizada"}

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
					t.Fatal("UpdateUser should not be called for an unauthenticated request")
					return pgstore.User{}, nil
				},
			}),
		}

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerUpdateUser)).
			ServeHTTP(rec, jsonRequest(t, "PUT", "/api/v1/users", validPayload))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body)
		}
	})

	t.Run("invalid payload returns bad request without touching the store", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
					t.Fatal("UpdateUser should not be called when payload validation fails")
					return pgstore.User{}, nil
				},
			}),
		}

		invalidPayload := map[string]string{"name": "", "email": "not-an-email", "bio": ""}
		req := newAuthenticatedRequest(t, sessions, uuid.New(), "PUT", "/api/v1/users", invalidPayload)

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerUpdateUser)).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
		}
	})

	t.Run("success", func(t *testing.T) {
		sessions := scs.New()
		userID := uuid.New()
		updated := pgstore.User{ID: userID, Name: "Novo Nome", Email: "novo@email.com", Bio: "bio atualizada"}
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
					return updated, nil
				},
			}),
		}

		req := newAuthenticatedRequest(t, sessions, userID, "PUT", "/api/v1/users", validPayload)

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerUpdateUser)).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
		}
		resp := decodeResponse(t, rec)
		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("got data %#v, want a user object", resp.Data)
		}
		if data["email"] != updated.Email {
			t.Fatalf("got email %v, want %v", data["email"], updated.Email)
		}
	})

	t.Run("unexpected service error maps to internal server error", func(t *testing.T) {
		sessions := scs.New()
		a := Api{
			Sessions: sessions,
			UsersService: services.NewUsersService(pgstoretest.FakeUsersStore{
				UpdateUserFn: func(ctx context.Context, arg pgstore.UpdateUserParams) (pgstore.User, error) {
					return pgstore.User{}, errors.New("connection reset")
				},
			}),
		}

		req := newAuthenticatedRequest(t, sessions, uuid.New(), "PUT", "/api/v1/users", validPayload)

		rec := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(a.HandlerUpdateUser)).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body)
		}
	})
}
