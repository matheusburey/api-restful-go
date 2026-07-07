package api

import (
	"net/http"

	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !api.Sessions.Exists(r.Context(), "AuthenticatedUserID") {
			utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
