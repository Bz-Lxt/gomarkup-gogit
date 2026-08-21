package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestUnstagePreservesIndexWhenHeadBlobIsMissing(t *testing.T) {
	workDir := t.TempDir()
	log := logger.New(logger.Error, io.Discard)
	repo, err := git.Init(workDir, git.SHA1, log)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("report.txt", "published\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"report.txt"}); err != nil {
		t.Fatal(err)
	}
	commit, err := repo.Commit("publish report", "Tester <test@example.com>")
	if err != nil {
		t.Fatal(err)
	}
	headTree, err := repo.FlattenMap(commit.Tree)
	if err != nil {
		t.Fatal(err)
	}
	headBlob := headTree["report.txt"].OID

	if err := repo.WriteWorktree("report.txt", "revised\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"report.txt"}); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Index()
	if err != nil {
		t.Fatal(err)
	}
	beforeEntry := before.Map()["report.txt"]

	objectPath := filepath.Join(git.GitDir(workDir), "objects", headBlob[:2], headBlob[2:])
	if err := os.Remove(objectPath); err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{"paths": []string{"report.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/index/unstage", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	New(repo, "", log).Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("unstage unexpectedly succeeded after a referenced object disappeared: %s", rec.Body.String())
	}
	after, err := repo.Index()
	if err != nil {
		t.Fatal(err)
	}
	afterEntry, ok := after.Map()["report.txt"]
	if !ok || afterEntry.OID != beforeEntry.OID {
		t.Fatalf("failed unstage changed staged entry: before=%+v after=%+v present=%v", beforeEntry, afterEntry, ok)
	}
}
