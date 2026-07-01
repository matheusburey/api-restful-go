package api

import (
	"errors"
	"net/http"

	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/usecase/users"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) HandlerLoginUser(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[users.LoginUserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, problems)
		return
	}
	user_id, err := api.UsersService.AuthenticateUser(r.Context(), data.Email, data.Password)

	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.EncodeJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	err = api.Sessions.RenewToken(r.Context())
	if err != nil {
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	api.Sessions.Put(r.Context(), "AuthenticatedUserID", user_id.String())
	utils.EncodeJSON(w, r, http.StatusOK, map[string]string{"message": "success"})
}

func (api *Api) HandlerLogoutUser(w http.ResponseWriter, r *http.Request) {
	err := api.Sessions.RenewToken(r.Context())
	if err != nil {
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	api.Sessions.Remove(r.Context(), "AuthenticatedUserID")
	utils.EncodeJSON(w, r, http.StatusOK, map[string]string{"message": "success"})
}

func (api *Api) HandlerSignupUser(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[users.CreateUserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, problems)
		return
	}
	user_id, err := api.UsersService.CreateUser(r.Context(), data.Name, data.Email, data.Bio, data.Password)

	if err != nil {
		if errors.Is(err, services.ErrDuplicatedEmail) {
			utils.EncodeJSON(w, r, http.StatusUnprocessableEntity, map[string]string{"error": "email already registered"})
			return
		}
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	utils.EncodeJSON(w, r, http.StatusCreated, map[string]string{"id": user_id.String()})
}

func (api *Api) HandlerDeleteUser(w http.ResponseWriter, r *http.Request) {
	auth_user_id := api.Sessions.GetString(r.Context(), "AuthenticatedUserID")
	id, err := utils.ValidateAndParseUUID(auth_user_id)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	err = api.UsersService.DeleteUser(r.Context(), id)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *Api) HandlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	auth_user_id := api.Sessions.GetString(r.Context(), "AuthenticatedUserID")
	id, err := utils.ValidateAndParseUUID(auth_user_id)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	data, problems, err := utils.DecodeValidJSON[users.UpdateUserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, problems)
		return
	}

	u, err := api.UsersService.UpdateUser(r.Context(), id, data.Name, data.Email, data.Bio, data.Password)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "something went wrong"})
		return
	}

	utils.EncodeJSON(w, r, http.StatusOK, u)
}
