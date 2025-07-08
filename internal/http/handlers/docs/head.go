package docs

import (
	"context"
	"errors"
	"fileserver/internal/models"
	errutil "fileserver/internal/utils/http_errors"
	parseutil "fileserver/internal/utils/parseLimit"
	"fmt"
	"log/slog"
	"net/http"
)

func Head(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, auth AuthService, dp DocumentProvider, up UserIDProvider) {
	op := pkg + "Head"

	log = log.With(slog.String("op", op))

	token := r.URL.Query().Get("token")
	login := r.URL.Query().Get("login")
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	limit := parseutil.ParseLimit(r.URL.Query().Get("limit"))

	requester, err := auth.UserByToken(ctx, token)
	if err != nil {
		log.Warn("failed get user by token", slog.String("error", err.Error()))
		errutil.WriteStatusError(w, http.StatusForbidden)
		return
	}

	filter := models.DocumentFilter{
		Key:   key,
		Value: value,
		Limit: limit,
	}

	rawDocs, err := dp.ListDocuments(ctx, requester, login, filter)
	if err != nil {
		log.Error("failed to list documents", slog.String("error", err.Error()))
		errutil.WriteStatusError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Documents-Count", fmt.Sprint(len(rawDocs)))
	w.WriteHeader(http.StatusOK)
}

func HeadByID(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, docID string, auth AuthService, dp DocumentProvider, up UserIDProvider) {
	op := pkg + "HeadByID"

	log = log.With(slog.String("op", op))

	token := r.URL.Query().Get("token")

	requester, err := auth.UserByToken(ctx, token)
	if err != nil {
		log.Error("failed get user by token", slog.String("error", err.Error()))
		errutil.WriteStatusError(w, http.StatusForbidden)
		return
	}

	doc, _, err := dp.DocumentByID(ctx, docID, requester)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			log.Error("failed to get document by id, permission denied", slog.String("error", err.Error()))
			errutil.WriteStatusError(w, http.StatusForbidden)
			return
		}
		log.Warn("failed to get document by id", slog.String("error", err.Error()))
		errutil.WriteStatusError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("X-Content-Mime", doc.Mime)

	if doc.IsFile {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", doc.Name))
		w.Header().Set("Content-Type", doc.Mime)

		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
