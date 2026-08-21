package api

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"gogit/internal/git"
	"gogit/internal/logger"
)

func TestCheckoutPreservesUnstagedChanges(t *testing.T) {
	log := logger.New(logger.Error, io.Discard)
	repo, err := git.Init(t.TempDir(), git.SHA1, log)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("settings.json", "{\"channel\":\"main\"}\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"settings.json"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Commit("main settings", "Dev <dev@example.com>"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateBranch("release"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Checkout("release"); err != nil {
		t.Fatal(err)
	}
	if err := repo.WriteWorktree("settings.json", "{\"channel\":\"release\"}\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add([]string{"settings.json"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Commit("release settings", "Dev <dev@example.com>"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	const local = "{\"channel\":\"local-draft\"}\n"
	if err := repo.WriteWorktree("settings.json", local); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(`{"name":"release"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	New(repo, "", log).Handler().ServeHTTP(rec, req)
	if rec.Code != 409 {
		t.Fatalf("checkout status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
	got, err := repo.ReadWorktree("settings.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != local {
		t.Fatalf("worktree content = %q, want local draft %q", got, local)
	}
	branch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Fatalf("current branch = %q, want main", branch)
	}
}
