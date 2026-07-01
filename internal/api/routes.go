package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (api *Api) BindRoutes() {
	api.Router.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger, api.Sessions.LoadAndSave)

	api.Router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Post("/signup", api.HandlerSignupUser)
				r.Post("/login", api.HandlerLoginUser)
				r.With(api.AuthMiddleware).Post("/logout", api.HandlerLogoutUser)
				r.With(api.AuthMiddleware).Put("/", api.HandlerUpdateUser)
				r.With(api.AuthMiddleware).Delete("/", api.HandlerDeleteUser)
			})
		})
	})
}
