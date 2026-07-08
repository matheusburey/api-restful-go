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
				r.Group(func(r chi.Router) {
					r.Use(api.AuthMiddleware)

					r.Post("/logout", api.HandlerLogoutUser)
					r.Put("/", api.HandlerUpdateUser)
					r.Delete("/", api.HandlerDeleteUser)
				})
			})

			r.Route("/products", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(api.AuthMiddleware)

					r.Post("/", api.HandlerCreateProduct)

					r.Get("/ws/subscribe/{product_id}", api.HandlerSubscribeUserToAuction)
				})

			})
		})
	})
}
