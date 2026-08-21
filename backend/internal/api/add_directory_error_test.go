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

func TestAddDirectoryReadFailureIsAtomic(t *testing.T) {
	workDir := t.TempDir()
	repo, err := git.Init(workDir, git.SHA1, nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(workDir, "docs")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-target", filepath.Join(dir, "z-broken")); err != nil {
		t.Fatal(err)
	}

	log := logger.New(logger.Error, &bytes.Buffer{})
	handler := New(repo, "", log).Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/index/add", bytes.NewBufferString(`{"paths":["docs"]}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code < 400 {
		t.Errorf("adding an unreadable directory returned %d, want an error; body=%s", res.Code, res.Body.String())
	}

	indexReq := httptest.NewRequest(http.MethodGet, "/api/v1/index", nil)
	indexRes := httptest.NewRecorder()
	handler.ServeHTTP(indexRes, indexReq)
	if indexRes.Code != http.StatusOK {
		t.Fatalf("reading index returned %d: %s", indexRes.Code, indexRes.Body.String())
	}
	var body struct {
		Data struct {
			Entries []json.RawMessage `json:"entries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(indexRes.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Entries) != 0 {
		t.Fatalf("failed directory add partially changed the index: %s", indexRes.Body.String())
	}
}
