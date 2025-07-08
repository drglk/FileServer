package session

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

func Add(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, sa SessionAdder) {
	op := pkg + "Add"

	log = log.With(slog.String("op", op))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read body", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}
	defer r.Body.Close()

	var sessionRequest dto.SessionRequest

	err = json.Unmarshal(body, &sessionRequest)
	if err != nil {
		log.Error("unmarshal body", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}

	token, err := sa.Login(ctx, sessionRequest.Login, sessionRequest.Password)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			log.Warn("failed to add session", slog.String("error", models.ErrUserNotFound.Error()))
			utils.WriteJSONError(w, http.StatusBadRequest, models.ErrInvalidCredentials.Error())
			return
		}
		if errors.Is(err, models.ErrInvalidCredentials) {
			log.Warn("failed to add session", slog.String("error", models.ErrInvalidParams.Error()))
			utils.WriteJSONError(w, http.StatusBadRequest, models.ErrInvalidCredentials.Error())
			return
		}
		log.Error("failed to add session", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]any{
		"response": map[string]any{
			"token": token,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to write response", slog.String("error", err.Error()))
	}
}
