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
	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-runCtx.Done()
	ctx, cancel := context.WithTimeout(runCtx, 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Info("gogit stopped")
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
