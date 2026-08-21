package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestRevParseExplicitCorruptRefDoesNotFallBackToObject(t *testing.T) {
	workDir := t.TempDir()
	log := logger.New(logger.Error, &bytes.Buffer{})
	repo, err := git.Init(workDir, git.SHA1, log)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("tracked.txt", "content\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"tracked.txt"}); err != nil {
		t.Fatal(err)
	}
	commit, err := repo.Commit("initial", "Test <test@example.com>")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateBranch(commit.Hash); err != nil {
		t.Fatal(err)
	}
	refPath := filepath.Join(workDir, ".gogit", "refs", "heads", commit.Hash)
	if err := os.WriteFile(refPath, []byte("truncated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rev-parse?q=refs/heads/"+commit.Hash, nil)
	res := httptest.NewRecorder()
	New(repo, "", log).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("explicit corrupt ref returned status %d, body %s", res.Code, res.Body.String())
	}
	var body struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error == nil || body.Error.Code != "validation_error" {
		t.Fatalf("explicit corrupt ref returned %#v", body.Error)
	}
}
