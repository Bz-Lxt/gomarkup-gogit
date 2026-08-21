package api

import (
	"net/http"
	"strings"
)

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	side := r.URL.Query().Get("side")
	s.mu.Lock()
	defer s.mu.Unlock()
	d, err := s.repo.DiffPath(path, side)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, d)
}

func (s *Server) unstage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.Unstage(req.Paths); err != nil {
		s.fail(w, err)
		return
	}
	st, err := s.repo.Status()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, st)
}

func (s *Server) restore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.RestoreWorktree(req.Paths); err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"restored": req.Paths})
}

func (s *Server) resetPaths(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
		Mode  string   `json:"mode"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.ResetPaths(req.Paths, req.Mode); err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"reset": req.Paths, "mode": req.Mode})
}

func (s *Server) revParse(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	s.mu.Lock()
	defer s.mu.Unlock()
	oid, kind, err := s.repo.RevParse(q)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"q": q, "oid": oid, "kind": kind})
}

func (s *Server) fsck(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rep, err := s.repo.Fsck()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, rep)
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := s.repo.Config()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, cfg)
}

func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserName  string `json:"user_name"`
		UserEmail string `json:"user_email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, err := s.repo.Config()
	if err != nil {
		s.fail(w, err)
		return
	}
	if strings.TrimSpace(req.UserName) != "" {
		cfg.UserName = strings.TrimSpace(req.UserName)
	}
	if strings.TrimSpace(req.UserEmail) != "" {
		cfg.UserEmail = strings.TrimSpace(req.UserEmail)
	}
	if err := s.repo.SetConfig(cfg); err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, cfg)
}
