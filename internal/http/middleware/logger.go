package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(lrw, r)

			log.Debug("HTTP Request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", lrw.statusCode),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if lrw.wroteHeader {
		return
	}
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
	lrw.wroteHeader = true
}
