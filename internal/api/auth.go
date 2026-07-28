package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

// sessionAuthenticatedUserIDKey is the single session key used to store the
// authenticated user's id. Login (Put), logout (Remove), and AuthMiddleware
// (Get) all share this constant instead of repeating the string literal.
const sessionAuthenticatedUserIDKey = "AuthenticatedUserID"

type contextKey int

const authenticatedUserIDContextKey contextKey = iota

// AuthenticatedUserID returns the user id AuthMiddleware attached to ctx, and
// whether one was present. Handlers behind AuthMiddleware read it from here
// instead of querying the session store directly.
func AuthenticatedUserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(authenticatedUserIDContextKey).(uuid.UUID)
	return id, ok
}

func (api *Api) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := api.Sessions.Get(r.Context(), sessionAuthenticatedUserIDKey).(uuid.UUID)
		if !ok {
			utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authenticatedUserIDContextKey, id)))
	})
}
