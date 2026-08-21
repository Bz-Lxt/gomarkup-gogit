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

func TestFsckRejectsCorruptBranchRef(t *testing.T) {
	dir := t.TempDir()
	log := logger.New(logger.Error, &bytes.Buffer{})
	repo, err := git.Init(dir, git.SHA1, log)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("README.md", "healthy\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"README.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Commit("initial", "Test <test@example.com>"); err != nil {
		t.Fatal(err)
	}

	ref := filepath.Join(git.GitDir(dir), "refs", "heads", "main")
	if err := os.WriteFile(ref, []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fsck", nil)
	res := httptest.NewRecorder()
	New(repo, "", log).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("fsck status = %d, want %d; body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	var body struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error == nil || body.Error.Code != "validation_error" {
		t.Fatalf("fsck error = %+v, want validation_error", body.Error)
	}
}
