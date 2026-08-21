package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gogit/internal/api"
	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestMergeConflictReturnsConflictResponse(t *testing.T) {
	log := logger.New(logger.Error, io.Discard)
	repo, err := git.Init(t.TempDir(), git.SHA1, log)
	if err != nil {
		t.Fatal(err)
	}
	commitFile := func(content, message string) {
		t.Helper()
		if err := repo.WriteWorktree("notes.txt", content); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Add([]string{"notes.txt"}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Commit(message, "Developer <dev@example.com>"); err != nil {
			t.Fatal(err)
		}
	}

	commitFile("heading\nbase text\n", "base")
	if _, err := repo.CreateBranch("release"); err != nil {
		t.Fatal(err)
	}
	commitFile("heading\nmain text\n", "edit on main")
	if err := repo.Checkout("release"); err != nil {
		t.Fatal(err)
	}
	commitFile("heading\nrelease text\n", "edit on release")
	if err := repo.Checkout("main"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/merge", strings.NewReader(`{"branch":"release"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.New(repo, "", log).Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("merge status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	var body struct {
		Error *struct {
			Code    string   `json:"code"`
			Details []string `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rec.Body.String())
	}
	if body.Error == nil || body.Error.Code != "conflict" {
		t.Fatalf("response error = %+v, want conflict", body.Error)
	}
	if len(body.Error.Details) != 1 || body.Error.Details[0] != "notes.txt" {
		t.Fatalf("conflict details = %v, want [notes.txt]", body.Error.Details)
	}
}
