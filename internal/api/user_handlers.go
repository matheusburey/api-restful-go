package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matheusburey/api-restful-go/internal/services"
	"github.com/matheusburey/api-restful-go/internal/usecase/users"
	"github.com/matheusburey/api-restful-go/internal/utils"
)

func (api *Api) HandlerGetAllUser(w http.ResponseWriter, r *http.Request) {
	users, err := api.UsersService.GetAllUsers(r.Context())
	if err != nil {
		utils.EncodeJSON(w, r, http.StatusInternalServerError, err)
		return
	}
	utils.EncodeJSON(w, r, http.StatusOK, users)
}

func (api *Api) HandlerPostUser(w http.ResponseWriter, r *http.Request) {
	data, problems, err := utils.DecodeValidJSON[users.UserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, problems)
		return
	}

	if err := api.UsersService.CreateUser(r.Context(), data.Name, data.Email); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			utils.EncodeJSON(w, r, http.StatusUnprocessableEntity, map[string]string{"error": "email already registered"})
			return
		}
		utils.EncodeJSON(w, r, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (api *Api) HandlerGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ValidateAndParseUUID(chi.URLParam(r, "id"))

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	u, err := api.UsersService.GetUserById(r.Context(), id)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	utils.EncodeJSON(w, r, http.StatusOK, u)
}

func (api *Api) HandlerDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ValidateAndParseUUID(chi.URLParam(r, "id"))

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
	id, err := utils.ValidateAndParseUUID(chi.URLParam(r, "id"))

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	data, problems, err := utils.DecodeValidJSON[users.UserReqBody](r)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusBadRequest, problems)
		return
	}

	u, err := api.UsersService.UpdateUser(r.Context(), id, data.Name, data.Email)

	if err != nil {
		utils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "something went wrong"})
		return
	}

	utils.EncodeJSON(w, r, http.StatusOK, u)
}
