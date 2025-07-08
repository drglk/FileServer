package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	utils "fileserver/internal/utils/http_errors"
	"log/slog"
	"net/http"
)

func Delete(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, docID string, auth AuthService, dd DocumentDeleter, up UserIDProvider) {
	op := pkg + "Delete"

	log = log.With(slog.String("op", op))

	token := r.URL.Query().Get("token")

	requester, err := auth.UserByToken(ctx, token)
	if err != nil {
		log.Error("failed get user by token", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusForbidden, "token is invalid")
		return
	}

	err = dd.DeleteDocument(ctx, docID, requester)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			log.Error("failed to get document by id, permission denied", slog.String("error", err.Error()))
			utils.WriteJSONError(w, http.StatusForbidden, models.ErrForbidden.Error())
			return
		}
		log.Warn("failed to get document by id", slog.String("error", err.Error()))
		utils.WriteJSONError(w, http.StatusBadRequest, models.ErrInvalidParams.Error())
		return
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
