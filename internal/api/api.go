package api

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/matheusburey/api-restful-go/internal/services"
)

type Api struct {
	Router         *chi.Mux
	UsersService   services.UsersService
	ProductService services.ProductService
	BidsService    services.BidsService
	Sessions       *scs.SessionManager
	WsUpgrade      websocket.Upgrader
	AuctionLobby   services.AuctionLobby
}
