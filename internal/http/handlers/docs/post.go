package docs

import (
	"context"
	"encoding/json"
	"fileserver/internal/dto"
	"fileserver/internal/models"
	errutils "fileserver/internal/utils/http_errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
)

func Upload(ctx context.Context, log *slog.Logger, w http.ResponseWriter, r *http.Request, auth AuthService, du DocumentUploader, up UserIDProvider) {
	op := pkg + "Upload"

	log = log.With(slog.String("op", op))

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Error("failed to parse multipart form", slog.String("error", err.Error()))
		errutils.WriteJSONError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	metaPart := r.FormValue("meta")

	var meta dto.UploadMeta

	if err := json.Unmarshal([]byte(metaPart), &meta); err != nil {
		log.Error("failed to unmarshal meta", slog.String("error", err.Error()))
		errutils.WriteJSONError(w, http.StatusBadRequest, "invalid meta json")
		return
	}

	requester, err := auth.UserByToken(ctx, meta.Token)
	if err != nil {
		log.Warn("failed get user by token", slog.String("error", err.Error()))
		errutils.WriteJSONError(w, http.StatusForbidden, "token is invalid")
		return
	}

	var jsonData []byte

	if jsonFile, _, err := r.FormFile("json"); err == nil {
		defer jsonFile.Close()

		jsonData, _ = io.ReadAll(jsonFile)
	}

	if len(jsonData) > 0 && !json.Valid(jsonData) {
		log.Warn("invalid json body")
		errutils.WriteJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var file multipart.File

	if meta.IsFile {
		var err error
		file, _, err = r.FormFile("file")
		if err != nil {
			errutils.WriteJSONError(w, http.StatusBadRequest, "failed upload error")
			return
		}

		defer file.Close()
	}

	grantsID := make([]string, 0)

	for _, grantName := range meta.Grants {
		grantID, err := up.UserIDByLogin(ctx, grantName)
		if err != nil {
			log.Warn("failed to get grant ID by grant name", slog.String("grant_name", grantName))
			continue
		}

		grantsID = append(grantsID, grantID)
	}

	doc := models.Document{
		OwnerID:  requester.ID,
		Name:     meta.Name,
		Mime:     meta.Mime,
		IsFile:   meta.IsFile,
		IsPublic: meta.IsPublic,
		JSONData: jsonData,
		Grants:   grantsID,
	}

	_, err = du.UploadDocument(ctx, requester, &doc, file)
	if err != nil {
		log.Error("failed to upload document", slog.String("error", err.Error()))
		errutils.WriteJSONError(w, http.StatusInternalServerError, models.ErrInternal.Error())
		return
	}

	response := map[string]any{
		"data": map[string]any{
			"json": json.RawMessage(jsonData),
			"file": doc.Name,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to write response", slog.String("error", err.Error()))
	}
}
