package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestCreateNestedBranchThroughAPI(t *testing.T) {
	repo, err := git.Init(t.TempDir(), git.SHA1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("README.md", "initial\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"README.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Commit("initial", "Test <test@example.com>"); err != nil {
		t.Fatal(err)
	}

	handler := New(repo, "", logger.New(logger.Error, io.Discard)).Handler()

	body := bytes.NewBufferString(`{"name":"feature/login"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/branches", body)
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create nested branch status = %d, want %d; body=%s", createResp.Code, http.StatusCreated, createResp.Body.Bytes())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/branches", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	var result struct {
		Data []git.BranchInfo `json:"data"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	for _, branch := range result.Data {
		if branch.Name == "feature/login" {
			return
		}
	}
	t.Fatalf("created branch missing from branch list: %+v", result.Data)
}
