package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gogit/internal/git"
	"gogit/internal/logger"
)

type Server struct {
	repo   *git.Repo
	webDir string
	log    *logger.Logger
	mu     sync.Mutex
	loc    *time.Location
}

func New(repo *git.Repo, webDir string, log *logger.Logger) *Server {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("GMT+8", 8*3600)
	}
	return &Server{repo: repo, webDir: webDir, log: log, loc: loc}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/repo", s.getRepo)
	mux.HandleFunc("POST /api/v1/repo/init", s.initRepo)
	mux.HandleFunc("GET /api/v1/files", s.listFiles)
	mux.HandleFunc("GET /api/v1/files/content", s.fileContent)
	mux.HandleFunc("PUT /api/v1/files", s.putFile)
	mux.HandleFunc("DELETE /api/v1/files", s.deleteFile)
	mux.HandleFunc("POST /api/v1/index/add", s.add)
	mux.HandleFunc("GET /api/v1/index", s.getIndex)
	mux.HandleFunc("GET /api/v1/status", s.status)
	mux.HandleFunc("POST /api/v1/commits", s.commit)
	mux.HandleFunc("GET /api/v1/commits", s.listCommits)
	mux.HandleFunc("GET /api/v1/commits/{hash}/tree", s.commitTree)
	mux.HandleFunc("GET /api/v1/commits/{hash}", s.getCommit)
	mux.HandleFunc("GET /api/v1/branches", s.listBranches)
	mux.HandleFunc("POST /api/v1/branches", s.createBranch)
	mux.HandleFunc("POST /api/v1/checkout", s.checkout)
	mux.HandleFunc("POST /api/v1/merge", s.merge)
	mux.HandleFunc("GET /api/v1/objects/{hash}", s.inspect)
	mux.HandleFunc("GET /api/v1/diff", s.diff)
	mux.HandleFunc("POST /api/v1/index/unstage", s.unstage)
	mux.HandleFunc("POST /api/v1/index/reset", s.resetPaths)
	mux.HandleFunc("POST /api/v1/files/restore", s.restore)
	mux.HandleFunc("GET /api/v1/rev-parse", s.revParse)
	mux.HandleFunc("GET /api/v1/fsck", s.fsck)
	mux.HandleFunc("GET /api/v1/config", s.getConfig)
	mux.HandleFunc("PUT /api/v1/config", s.putConfig)
	mux.HandleFunc("/", s.spa)
	return s.wrap(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	s.ok(w, map[string]any{
		"status": "ok",
		"time":   time.Now().In(s.loc).Format("2006-01-02 15:04:05"),
	})
}

func (s *Server) getRepo(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, err := s.repo.Info()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, info)
}

func (s *Server) initRepo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HashAlgo string `json:"hash_algo"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	algo, err := git.ParseAlgo(req.HashAlgo)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = git.Init(s.repo.WorkDir(), algo, s.log)
	if err != nil {
		s.fail(w, err)
		return
	}
	opened, err := git.Open(s.repo.WorkDir(), s.log)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.repo = opened
	info, err := s.repo.Info()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.created(w, info)
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	s.mu.Lock()
	defer s.mu.Unlock()
	ents, err := s.repo.ListWorktree(p)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"path": p, "entries": ents})
}

func (s *Server) fileContent(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if strings.TrimSpace(p) == "" {
		s.fail(w, fmt.Errorf("%w: path is required", git.ErrInvalidPath))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.repo.ReadWorktree(p)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{
		"path":    p,
		"content": string(b),
		"binary":  git.IsBinary(b),
		"size":    len(b),
	})
}

func (s *Server) putFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.WriteWorktree(req.Path, req.Content); err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"path": req.Path, "size": len(req.Content)})
}

func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.DeleteWorktree(p); err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"deleted": p})
}

func (s *Server) add(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ents, err := s.repo.Add(req.Paths)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"entries": ents})
}

func (s *Server) getIndex(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.repo.Index()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, idx)
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.repo.Status()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, st)
}

func (s *Server) commit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
		Author  string `json:"author"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.repo.Commit(req.Message, req.Author)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.created(w, c)
}

func (s *Server) listCommits(w http.ResponseWriter, r *http.Request) {
	branch := r.URL.Query().Get("branch")
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.repo.Log(branch, 100)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, list)
}

func (s *Server) getCommit(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.repo.ReadCommit(hash)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, c)
}

func (s *Server) commitTree(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.repo.ReadCommit(hash)
	if err != nil {
		s.fail(w, err)
		return
	}
	ents, err := s.repo.FlattenTree(c.Tree, "")
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"entries": ents})
}

func (s *Server) listBranches(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.repo.ListBranches()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, list)
}

func (s *Server) createBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.repo.CreateBranch(req.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.created(w, b)
}

func (s *Server) checkout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.Checkout(req.Name); err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, map[string]any{"branch": req.Name})
}

func (s *Server) merge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Branch string `json:"branch"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.fail(w, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.repo.Merge(req.Branch)
	if err != nil {
		if errors.Is(err, git.ErrMergeConflict) {
			s.log.Warn("merge requires resolution", "branch", req.Branch, "commit", res.Commit.Hash, "conflicts", len(res.Conflicts))
			writeJSON(w, http.StatusConflict, envelope{
				Error: &eBody{Code: "conflict", Message: "merge produced conflicts", Details: res.Conflicts},
			})
			return
		}
		s.fail(w, err)
		return
	}
	s.ok(w, res)
}

func (s *Server) inspect(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	s.mu.Lock()
	defer s.mu.Unlock()
	obj, err := s.repo.Inspect(hash)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.ok(w, obj)
}

func (s *Server) spa(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.fail(w, fmt.Errorf("%w: unknown endpoint", git.ErrNotFound))
		return
	}
	if s.webDir == "" {
		s.ok(w, map[string]string{"message": "GoGit API is running"})
		return
	}
	fs := http.FileServer(http.Dir(s.webDir))
	p := r.URL.Path
	if p != "/" {
		// try file, else index.html
		fs.ServeHTTP(&spaWriter{ResponseWriter: w, fallback: func() {
			http.ServeFile(w, r, s.webDir+"/index.html")
		}}, r)
		return
	}
	fs.ServeHTTP(w, r)
}

type spaWriter struct {
	http.ResponseWriter
	fallback func()
	wrote    bool
	code     int
}

func (w *spaWriter) WriteHeader(code int) {
	w.code = code
	if code == http.StatusNotFound {
		w.fallback()
		w.wrote = true
		return
	}
	w.ResponseWriter.WriteHeader(code)
	w.wrote = true
}

func (w *spaWriter) Write(b []byte) (int, error) {
	if w.code == http.StatusNotFound && w.wrote {
		return len(b), nil
	}
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
