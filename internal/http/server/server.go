package server

import (
	"context"
	"errors"
	"fileserver/internal/config"
	"fileserver/internal/http/handlers/docs"
	"fileserver/internal/http/handlers/session"
	"fileserver/internal/http/handlers/user"
	"fileserver/internal/http/middleware"
	"fileserver/internal/models"
	utils "fileserver/internal/utils/http_errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func StartServer(
	ctx context.Context,
	cfg *config.HTTPServer,
	log *slog.Logger,
	documentService DocumentService,
	authService AuthService,
	userService UserService,
	sessionStorer SessionStorer,
) error {
	r := mux.NewRouter()

	r.Use(middleware.Logger(log))

	setupRoutes(r, log, authService, documentService, userService, sessionStorer)

	srv := &http.Server{
		Addr:         cfg.Address,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
		Handler:      r,
	}

	errChan := make(chan error, 1)

	go func() {
		log.Info("server started", slog.String("address", cfg.Address))
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("server closed gracefully")
			} else {
				log.Error("could not start server:", "error", err)
				errChan <- err
			}
		}
	}()
	select {
	case <-ctx.Done():
		log.Info("shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("error shutting down server", "error", err)
			return err
		}
		log.Info("server exited gracefully")
		return nil
	case err := <-errChan:
		return err
	}
}

func setupRoutes(r *mux.Router, log *slog.Logger, auth AuthService, doc DocumentService, us UserService, sessionStorer SessionStorer) {
	// POST user
	r.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user.Add(ctx, log, w, r, auth)
	}).Methods(http.MethodPost)

	// POST session
	r.HandleFunc("/api/auth", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session.Add(ctx, log, w, r, auth)
	}).Methods(http.MethodPost)

	// DELETE session
	r.HandleFunc("/api/auth/{token}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		token := vars["token"]
		session.Delete(ctx, log, w, r, token, auth)
	}).Methods(http.MethodDelete)

	// POST doc
	r.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		docs.Upload(ctx, log, w, r, sessionStorer, doc, us)
	}).Methods(http.MethodPost)

	protected := r.NewRoute().Subrouter()

	protected.Use(middleware.Auth(log, sessionStorer))

	// GET docs
	protected.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		docs.Get(ctx, log, w, r, doc, us)
	}).Methods(http.MethodGet)

	// GET doc by id
	protected.HandleFunc("/api/docs/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		docID := vars["id"]
		docs.GetByID(ctx, log, w, r, docID, doc, us)
	}).Methods(http.MethodGet)

	// HEAD docs
	protected.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		docs.Head(ctx, log, w, r, doc, us)
	}).Methods(http.MethodHead)

	// HEAD doc by ID
	protected.HandleFunc("/api/docs/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		docID := vars["id"]
		docs.HeadByID(ctx, log, w, r, docID, doc, us)
	}).Methods(http.MethodHead)

	// DELETE doc by id
	protected.HandleFunc("/api/docs/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		docID := vars["id"]
		docs.Delete(ctx, log, w, r, docID, doc, us)
	}).Methods(http.MethodDelete)

	// Not allowed
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, models.ErrMethodNotAllowed.Error())
	})
}
