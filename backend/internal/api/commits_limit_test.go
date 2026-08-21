package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestCommitHistoryIncludesLimitBoundary(t *testing.T) {
	repo, err := git.Init(t.TempDir(), git.SHA1, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		if err := repo.WriteWorktree("counter.txt", string(rune(i))); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Add([]string{"counter.txt"}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Commit("increment", "Test <test@example.com>"); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/commits", nil)
	resp := httptest.NewRecorder()
	New(repo, "", logger.New(logger.Error, io.Discard)).Handler().ServeHTTP(resp, req)
	if resp.Code != 200 {
		t.Fatalf("history status = %d, want 200", resp.Code)
	}
	var body struct {
		Data []git.Commit `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 100 {
		t.Fatalf("history returned %d commits, want 100", len(body.Data))
	}
}
