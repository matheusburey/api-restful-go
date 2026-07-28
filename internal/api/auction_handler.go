package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) HandlerSubscribeUserToAuction(w http.ResponseWriter, r *http.Request) {
	rawProductID := chi.URLParam(r, "product_id")
	product_id, err := uuid.Parse(rawProductID)
	if err != nil {
		utils.EncodeJSON(w, http.StatusBadRequest, utils.Response{Error: "invalid product id"})
		return
	}

	_, err = api.ProductService.GetProductByID(r.Context(), product_id)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			utils.EncodeJSON(w, http.StatusNotFound, utils.Response{Error: "product not found"})
			return
		}
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "internal server error"})
		return
	}

	id, ok := AuthenticatedUserID(r.Context())
	if !ok {
		utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "unauthorized "})
		return
	}

	api.AuctionLobby.Lock()
	room, ok := api.AuctionLobby.Rooms[product_id]
	api.AuctionLobby.Unlock()

	if !ok {
		utils.EncodeJSON(w, http.StatusNotFound, utils.Response{Error: "the auction has finished"})
		return
	}

	conn, err := api.WsUpgrade.Upgrade(w, r, nil)
	if err != nil {
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "internal server error"})
		return
	}

	client := services.NewClient(conn, room, id)

	room.Register <- client
	go client.ReadEventLoop()
	go client.WriteEventLoop()
}
