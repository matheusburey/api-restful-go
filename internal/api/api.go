package api

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/matheusburey/api-restful-go/internal/services"
)

type Api struct {
	Router       *chi.Mux
	UsersService services.UsersService
	Sessions     *scs.SessionManager
}
