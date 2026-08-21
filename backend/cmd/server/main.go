package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gogit/internal/api"
	"gogit/internal/git"
	"gogit/internal/logger"
)

func main() {
	log := logger.New(logger.ParseLevel(env("LOG_LEVEL", "info")), os.Stdout)
	repoDir := env("GOGIT_REPO", "/data/repo")
	algo, err := git.ParseAlgo(env("GOGIT_HASH_ALGO", "sha1"))
	if err != nil {
		log.Error("invalid hash algo", "err", err)
		os.Exit(1)
	}
	webDir := env("GOGIT_WEB", "/app/web")
	if st, err := os.Stat(webDir); err != nil || !st.IsDir() {
		webDir = ""
	}
	repo, err := git.OpenOrInit(repoDir, algo, log, true)
	if err != nil {
		log.Error("open repo failed", "err", err)
		os.Exit(1)
	}
	srv := api.New(repo, webDir, log)
	addr := ":" + env("PORT", "8080")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Info("gogit listening", "addr", addr, "repo", repoDir)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownServer(httpSrv, log, shutdownTimeout())
	log.Info("gogit stopped")
}

// shutdownTimeout returns the graceful-shutdown grace window. It can be
// overridden with GOGIT_SHUTDOWN_TIMEOUT (e.g. "15s") and defaults to
// 8 seconds, which fits the typical container graceful-stop window.
func shutdownTimeout() time.Duration {
	d := 8 * time.Second
	if v := os.Getenv("GOGIT_SHUTDOWN_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			d = parsed
		}
	}
	return d
}

// shutdownServer performs a bounded graceful shutdown of httpSrv.
//
// The grace window is enforced even when there are connections stuck in
// the active state (e.g. a client that started an upload and then
// stalled, never completing the request body). http.Server.Shutdown only
// closes idle connections and waits indefinitely for active ones to
// return to idle; if the deadline were stripped — as it was previously
// via context.WithoutCancel — Shutdown would poll forever and the
// process would only ever be killed by SIGKILL after the container's
// stop grace period elapsed.
//
// Here Shutdown is given the real (deadline-bearing) context. When the
// grace window expires, Shutdown returns context.DeadlineExceeded and
// httpSrv.Close force-closes any still-active connections so the
// process can exit promptly.
func shutdownServer(httpSrv *http.Server, log *logger.Logger, grace time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	start := time.Now()
	err := httpSrv.Shutdown(ctx)
	if err != nil {
		log.Warn("graceful shutdown timed out, forcing close", "err", err, "elapsed", time.Since(start).String(), "grace", grace.String())
		_ = httpSrv.Close()
		return
	}
	log.Info("graceful shutdown complete", "elapsed", time.Since(start).String())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
