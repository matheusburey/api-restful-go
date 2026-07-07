package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/usecase/users"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) HandlerLoginUser(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[users.LoginUserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, http.StatusBadRequest, utils.Response{Message: problems, Error: "bad request"})
		return
	}
	user_id, err := api.UsersService.AuthenticateUser(r.Context(), data.Email, data.Password)

	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "invalid credentials"})
			return
		}
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "internal server error"})
		return
	}
	err = api.Sessions.RenewToken(r.Context())
	if err != nil {
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "internal server error"})
		return
	}
	api.Sessions.Put(r.Context(), "AuthenticatedUserID", user_id)
	utils.EncodeJSON(w, http.StatusOK, utils.Response{Message: "success"})
}

func (api *Api) HandlerLogoutUser(w http.ResponseWriter, r *http.Request) {
	err := api.Sessions.RenewToken(r.Context())
	if err != nil {
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "internal server error"})
		return
	}
	api.Sessions.Remove(r.Context(), "AuthenticatedUserID")
	utils.EncodeJSON(w, http.StatusOK, utils.Response{Message: "success"})
}

func (api *Api) HandlerSignupUser(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[users.CreateUserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, http.StatusBadRequest, utils.Response{Message: problems, Error: "bad request"})
		return
	}
	_, err = api.UsersService.CreateUser(r.Context(), data.Name, data.Email, data.Bio, data.Password)

	if err != nil {
		if errors.Is(err, services.ErrDuplicatedEmail) {
			utils.EncodeJSON(w, http.StatusUnprocessableEntity, utils.Response{Error: "email already registered"})
			return
		}
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "internal server error"})
		return
	}
	utils.EncodeJSON(w, http.StatusCreated, utils.Response{Message: "success"})
}

func (api *Api) HandlerDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserID").(uuid.UUID)
	if !ok {
		utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "unauthorized "})
		return
	}

	err := api.UsersService.DeleteUser(r.Context(), id)

	if err != nil {
		utils.EncodeJSON(w, http.StatusNotFound, utils.Response{Error: "user not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *Api) HandlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserID").(uuid.UUID)
	if !ok {
		utils.EncodeJSON(w, http.StatusUnauthorized, utils.Response{Error: "unauthorized "})
		return
	}
	data, problems, err := utils.DecodeValidJSON[users.UpdateUserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, http.StatusBadRequest, utils.Response{Message: problems, Error: "bad request"})
		return
	}

	u, err := api.UsersService.UpdateUser(r.Context(), id, data.Name, data.Email, data.Bio, data.Password)

	if err != nil {
		utils.EncodeJSON(w, http.StatusInternalServerError, utils.Response{Error: "something went wrong"})
		return
	}

	utils.EncodeJSON(w, http.StatusOK, utils.Response{Data: u})
}
