package session

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	"log/slog"
	"net/http"
)

func Delete(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, token string, sd SessionDeleter) {
	op := pkg + "Delete"

	log.With(slog.String("op", op))

	err := sd.Logout(ctx, token)
	if err != nil && !errors.Is(err, models.ErrSessionNotFound) {
		log.Error("failed to delete session", slog.String("error", err.Error()))
	}

	response := map[string]any{
		"response": map[string]any{
			token: true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to write response", slog.String("error", err.Error()))
	}
}
