package users

import (
	"context"

	"github.com/matheusburey/api-restful-go/internal/utils"
)

type LoginUserReqBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req LoginUserReqBody) Valid(ctx context.Context) utils.Evaluator {
	var eval utils.Evaluator

	eval.CheckField(utils.IsEmail(req.Email), "email", "valid email is required")
	eval.CheckField(utils.NotBlank(req.Password), "password", "password is required")

	return eval
}
