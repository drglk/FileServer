package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/dto"
	"fileserver/internal/models"
	errutils "fileserver/internal/utils/http_errors"
	parseutil "fileserver/internal/utils/parseLimit"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func Get(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, dp DocumentProvider, up UserIDProvider) {
	op := pkg + "Get"

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
		errutils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}

	dtoDocks := make([]dto.DocumentResponse, 0)

	for _, doc := range rawDocs {
		dtoDocks = append(dtoDocks, dto.DocumentResponse{
			ID:        doc.ID,
			Name:      doc.Name,
			Mime:      doc.Mime,
			IsFile:    doc.IsFile,
			IsPublic:  doc.IsPublic,
			CreatedAt: doc.CreatedAt,
			Grants:    doc.Grants,
		})
	}

	response := map[string]any{
		"data": map[string]any{
			"docs": dtoDocks,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to write response", slog.String("error", err.Error()))
	}
}

func GetByID(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, docID string, dp DocumentProvider, up UserIDProvider) {
	op := pkg + "GetByID"

	log = log.With(slog.String("op", op))

	requester, ok := r.Context().Value(models.UserContextKey).(*models.User)
	if !ok {
		log.Error("failed to get user from context")
		errutils.WriteJSONError(w, http.StatusForbidden, models.ErrForbidden.Error())
		return
	}

	doc, file, err := dp.DocumentByID(ctx, docID, requester)
	if err != nil {
		if errors.Is(err, models.ErrForbidden) {
			log.Error("failed to get document by id, permission denied", slog.String("error", err.Error()))
			errutils.WriteJSONError(w, http.StatusForbidden, models.ErrForbidden.Error())
			return
		}
		log.Warn("failed to get document by id", slog.String("error", err.Error()))
		errutils.WriteJSONError(w, http.StatusBadRequest, models.ErrInvalidParams.Error())
		return
	}

	if file != nil {
		defer file.Close()
	}

	if doc.IsFile {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", doc.Name))
		w.Header().Set("Content-Type", doc.Mime)
		if _, err := io.Copy(w, file); err != nil {
			log.Error("failed to write file response", slog.String("error", err.Error()))
		}
		return
	}

	var parsed map[string]any

	if err := json.Unmarshal(doc.JSONData, &parsed); err != nil {
		log.Error("invalid JSON in JSONData", slog.String("error", err.Error()))
		errutils.WriteJSONError(w, http.StatusInternalServerError, "invalid json data")
		return
	}

	response := map[string]any{
		"data": parsed,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to write response", slog.String("error", err.Error()))
	}
}
