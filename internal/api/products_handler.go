package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/usecase/product"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) HandlerCreateProduct(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[product.CreateProductReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, problems)
		return
	}

	id, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserID").(uuid.UUID)
	if !ok {
		utils.EncodeJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "unauthorized "})
		return
	}

	product_id, err := api.ProductService.CreateProduct(r.Context(), id, data.Name, data.Description, data.BasePriceCents, data.AuctionEnd)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "error creating product"})
		return
	}

	utils.EncodeJSON(w, r, http.StatusCreated, map[string]any{"id": product_id})
}
