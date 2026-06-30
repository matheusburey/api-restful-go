package api

import (
	"github.com/go-chi/chi/v5"
)

func (api *Api) BindRoutes() {
	api.Router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Post("/signup", api.HandlerSignupUser)
				r.Post("/login", api.HandlerLoginUser)
				r.Post("/logout", api.HandlerLogoutUser)
				r.Put("/", api.HandlerUpdateUser)
				r.Delete("/", api.HandlerDeleteUser)
			})
		})
	})
}
