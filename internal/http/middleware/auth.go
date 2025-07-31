package middleware

import (
	"context"
	"fileserver/internal/models"
	utils "fileserver/internal/utils/http_errors"
	"log/slog"
	"net/http"
)

func Auth(log *slog.Logger, storer SessionStorer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			op := pkg + "Auth"

			log = log.With(slog.String("op", op))

			token := r.URL.Query().Get("token")

			requester, err := storer.UserByToken(r.Context(), token)
			if err != nil {
				log.Error("failed get user by token", slog.String("error", err.Error()))
				utils.WriteJSONError(w, http.StatusForbidden, "token is invalid")
				return
			}

			ctx := context.WithValue(r.Context(), models.UserContextKey, requester)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
