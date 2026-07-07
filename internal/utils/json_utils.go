package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Error   string `json:"error,omitempty"`
	Message any    `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func EncodeJSON(w http.ResponseWriter, status int, data Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return err
	}

	return nil
}

func DecodeValidJSON[T Validator](r *http.Request) (T, map[string]string, error) {
	var data T
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return data, nil, err
	}

	if problems := data.Valid(r.Context()); len(problems) > 0 {
		return data, problems, fmt.Errorf("valid %T: %d problems", data, len(problems))
	}

	return data, nil, nil
}

func DecodeJSON[T any](r *http.Request) (T, error) {
	var data T
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return data, err
	}
	return data, nil
}
