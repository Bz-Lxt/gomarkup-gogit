package api

import (
	"net/http"
	"runtime/debug"
	"time"
)

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		start := time.Now()
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic recovered", "err", rec, "stack", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, envelope{
					Error: &eBody{Code: "internal_error", Message: "internal error"},
				})
			}
		}()
		lw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(lw, r)
		s.log.Debug("http", "method", r.Method, "path", r.URL.Path, "status", lw.code, "ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
