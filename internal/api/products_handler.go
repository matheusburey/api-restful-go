package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/usecase/product"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) HandlerCreateProduct(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[product.CreateProductReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, http.StatusBadRequest, utils.Response{Message: problems, Error: "bad request"})
		return
	}

	id, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserID").(uuid.UUID)
	if !ok {
		utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "unauthorized "})
		return
	}

	product_id, err := api.ProductService.CreateProduct(r.Context(), id, data.Name, data.Description, data.BasePriceCents, data.AuctionEnd)

	if err != nil {
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "error creating product"})
		return
	}

	ctx, _ := context.WithDeadline(context.Background(), data.AuctionEnd)
	auctionRoom := services.NewAuctionRoom(ctx, product_id, api.BidsService)
	go auctionRoom.Run()
	api.AuctionLobby.Lock()
	api.AuctionLobby.Rooms[product_id] = auctionRoom
	api.AuctionLobby.Unlock()

	utils.EncodeJSON(w, http.StatusCreated, utils.Response{Message: "Auction has started with success"})
}
