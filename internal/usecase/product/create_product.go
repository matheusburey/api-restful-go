package product

import (
	"context"
	"time"

	"github.com/matheusburey/api-restful-go/internal/utils"
)

const MIN_AUCTION_DURATION = 2 * time.Hour

type CreateProductReqBody struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	BasePriceCents int64     `json:"base_price_cents"`
	AuctionEnd     time.Time `json:"auction_end"`
}

func (req CreateProductReqBody) Valid(ctx context.Context) utils.Evaluator {
	var eval utils.Evaluator

	eval.CheckField(utils.NotBlank(req.Name), "name", "name is required")
	eval.CheckField(utils.NotBlank(req.Description), "description", "description is required")
	eval.CheckField(utils.MinLength(req.Description, 10) && utils.MaxLength(req.Name, 255), "description", "min length is 10 and max length is 255")
	eval.CheckField(req.BasePriceCents > 0, "base_price_cents", "base price must be greater than 0")
	eval.CheckField(time.Until(req.AuctionEnd) >= MIN_AUCTION_DURATION, "auction_end", "auction end must be in the future")

	return eval
}
