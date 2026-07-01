package users

import (
	"context"

	"github.com/matheusburey/api-restful-go/internal/utils"
)

type UpdateUserReqBody struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Bio      string  `json:"bio"`
	Password *string `json:"password"`
}

func (req UpdateUserReqBody) Valid(ctx context.Context) utils.Evaluator {
	var eval utils.Evaluator

	eval.CheckField(utils.NotBlank(req.Name), "name", "name is required")
	eval.CheckField(utils.NotBlank(req.Email), "email", "email is required")
	eval.CheckField(utils.IsEmail(req.Email), "email", "email is invalid")
	eval.CheckField(utils.NotBlank(req.Bio), "bio", "bio is required")
	eval.CheckField(utils.MinLength(req.Bio, 10) && utils.MaxLength(req.Name, 255), "bio", "min length is 10 and max length is 255")
	eval.CheckField(utils.NullStringNotBlank(req.Password), "password", "password is required")
	eval.CheckField(utils.NullStringMinLength(req.Password, 8), "password", "min length is 8")

	return eval
}
