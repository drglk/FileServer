package user

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/dto"
	"fileserver/internal/models"
	utils "fileserver/internal/utils/http_errors"
	"io"
	"log/slog"
	"net/http"
)

func Add(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, ua UserAdder) {
	op := pkg + "Add"

	log = log.With(slog.String("op", op))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read body", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}
	defer r.Body.Close()

	var userRequest dto.UserRequest

	err = json.Unmarshal(body, &userRequest)
	if err != nil {
		log.Error("unmarshal body", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}

	login, err := ua.Register(ctx, userRequest.Login, userRequest.Password, userRequest.AdminToken)
	if err != nil {
		if errors.Is(err, models.ErrUserExists) {
			log.Warn("failed to register user", slog.String("error", models.ErrUserExists.Error()))
			utils.WriteJSONError(w, http.StatusConflict, models.ErrUserExists.Error())
			return
		}
		if errors.Is(err, models.ErrInvalidParams) {
			log.Warn("failed to register user", slog.String("error", models.ErrInvalidParams.Error()))
			utils.WriteJSONError(w, http.StatusBadRequest, models.ErrInvalidParams.Error())
			return
		}
		if errors.Is(err, models.ErrForbidden) {
			log.Warn("failed to register user", slog.String("error", models.ErrForbidden.Error()))
			utils.WriteJSONError(w, http.StatusForbidden, models.ErrForbidden.Error())
			return
		}
		log.Error("failed to register user", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]any{
		"response": map[string]any{
			"login": login,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to write response", slog.String("error", err.Error()))
	}
}
