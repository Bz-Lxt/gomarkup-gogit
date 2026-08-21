package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gogit/internal/api"
	gitrepo "gogit/internal/git"
	"gogit/internal/logger"
)

func TestFailedCommitPreservesMergeState(t *testing.T) {
	repo, err := gitrepo.Init(t.TempDir(), gitrepo.SHA1, nil)
	if err != nil {
		t.Fatal(err)
	}
	writeCommit := func(content, message string) {
		t.Helper()
		if err := repo.WriteWorktree("conflict.txt", content); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Add([]string{"conflict.txt"}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Commit(message, "Dev <dev@example.com>"); err != nil {
			t.Fatal(err)
		}
	}

	writeCommit("base\n", "base")
	if _, err := repo.CreateBranch("release"); err != nil {
		t.Fatal(err)
	}
	writeCommit("main change\n", "main change")
	if err := repo.Checkout("release"); err != nil {
		t.Fatal(err)
	}
	writeCommit("release change\n", "release change")
	if err := repo.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Merge("release"); !errors.Is(err, gitrepo.ErrMergeConflict) {
		t.Fatalf("merge error = %v, want conflict", err)
	}

	handler := api.New(repo, "", logger.New(logger.Error, io.Discard)).Handler()
	badCommit := httptest.NewRequest(http.MethodPost, "/api/v1/commits", bytes.NewBufferString(`{"message":"","author":"Dev <dev@example.com>"}`))
	badCommit.Header.Set("Content-Type", "application/json")
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, badCommit)
	if badResponse.Code != http.StatusBadRequest {
		t.Fatalf("empty-message commit status = %d, want %d", badResponse.Code, http.StatusBadRequest)
	}

	infoResponse := httptest.NewRecorder()
	handler.ServeHTTP(infoResponse, httptest.NewRequest(http.MethodGet, "/api/v1/repo", nil))
	if infoResponse.Code != http.StatusOK {
		t.Fatalf("repo status = %d, want %d", infoResponse.Code, http.StatusOK)
	}
	var response struct {
		Data struct {
			MergeInProgress bool `json:"merge_in_progress"`
		} `json:"data"`
	}
	if err := json.NewDecoder(infoResponse.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if !response.Data.MergeInProgress {
		t.Fatal("failed commit discarded the in-progress merge")
	}
}
