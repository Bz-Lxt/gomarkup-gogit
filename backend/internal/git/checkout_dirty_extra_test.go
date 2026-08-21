package git

import (
	"errors"
	"testing"
)

// Checkout with unstaged changes to a file that only exists on current
// branch (will be deleted) must also be refused.
func TestCheckoutRefusesUnstagedOnDeletableFile(t *testing.T) {
	r := initRepo(t, SHA1)
	if err := r.WriteWorktree("keep.txt", "keep\n"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("only-main.txt", "only-main\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"keep.txt", "only-main.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("base", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateBranch("release"); err != nil {
		t.Fatal(err)
	}
	if err := r.Checkout("release"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("release-only.txt", "release\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"release-only.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("release commit", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	// back to main, edit only-main.txt without staging
	if err := r.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("only-main.txt", "only-main-edited\n"); err != nil {
		t.Fatal(err)
	}
	err := r.Checkout("release")
	if !errors.Is(err, ErrDirtyWorktree) {
		t.Fatalf("expected ErrDirtyWorktree for file-to-be-deleted, got %v", err)
	}
	// content preserved
	got, _ := r.ReadWorktree("only-main.txt")
	if string(got) != "only-main-edited\n" {
		t.Fatalf("worktree overwritten: %q", got)
	}
}

// Checkout should succeed when worktree already matches target even if
// the index has staged changes (no-op for that file).
func TestCheckoutAllowsWhenWorktreeMatchesTarget(t *testing.T) {
	r := initRepo(t, SHA1)
	if err := r.WriteWorktree("f.txt", "v1\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"f.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("c1", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateBranch("release"); err != nil {
		t.Fatal(err)
	}
	if err := r.Checkout("release"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("f.txt", "release-v1\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"f.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("release c1", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	// back to main, make f.txt match release exactly (unstaged change)
	if err := r.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("f.txt", "release-v1\n"); err != nil {
		t.Fatal(err)
	}
	// switching to release should succeed since worktree already matches target
	if err := r.Checkout("release"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	got, _ := r.ReadWorktree("f.txt")
	if string(got) != "release-v1\n" {
		t.Fatalf("content mismatch: %q", got)
	}
}
