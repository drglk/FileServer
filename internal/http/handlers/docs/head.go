package docs

import (
	"context"
	"errors"
	"fileserver/internal/models"
	errutils "fileserver/internal/utils/http_errors"
	parseutil "fileserver/internal/utils/parseLimit"
	"fmt"
	"log/slog"
	"net/http"
)

func Head(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, dp DocumentProvider, up UserIDProvider) {
	op := pkg + "Head"

	log = log.With(slog.String("op", op))

	login := r.URL.Query().Get("login")
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	limit := parseutil.ParseLimit(r.URL.Query().Get("limit"))

	requester, ok := r.Context().Value(models.UserContextKey).(*models.User)
	if !ok {
		log.Error("failed to get user from context")
		errutils.WriteJSONError(w, http.StatusForbidden, models.ErrForbidden.Error())
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
		errutils.WriteStatusError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Documents-Count", fmt.Sprint(len(rawDocs)))
	w.WriteHeader(http.StatusOK)
}

func HeadByID(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, docID string, dp DocumentProvider, up UserIDProvider) {
	op := pkg + "HeadByID"

	log = log.With(slog.String("op", op))

	requester, ok := r.Context().Value(models.UserContextKey).(*models.User)
	if !ok {
		log.Error("failed to get user from context")
		errutils.WriteJSONError(w, http.StatusForbidden, models.ErrForbidden.Error())
		return
	}

	doc, _, err := dp.DocumentByID(ctx, docID, requester)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			log.Error("failed to get document by id, permission denied", slog.String("error", err.Error()))
			errutils.WriteStatusError(w, http.StatusForbidden)
			return
		}
		log.Warn("failed to get document by id", slog.String("error", err.Error()))
		errutils.WriteStatusError(w, http.StatusBadRequest)
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
