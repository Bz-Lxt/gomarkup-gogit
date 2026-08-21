package git

import (
	"errors"
	"testing"
)

// Reproduces the user's scenario: editing a tracked file without staging
// then switching branches should be refused, not silently overwrite.
func TestCheckoutRefusesUnstagedChanges(t *testing.T) {
	r := initRepo(t, SHA1)
	// main: settings.json = "main-v1"
	if err := r.WriteWorktree("settings.json", "main-v1\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"settings.json"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("main settings", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	// create release branch from main, then modify settings on release
	if _, err := r.CreateBranch("release"); err != nil {
		t.Fatal(err)
	}
	if err := r.Checkout("release"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("settings.json", "release-v1\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"settings.json"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("release settings", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	// back to main
	if err := r.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	// hand-edit settings.json on main WITHOUT staging
	if err := r.WriteWorktree("settings.json", "main-v1-edited\n"); err != nil {
		t.Fatal(err)
	}
	// switching to release must be refused
	err := r.Checkout("release")
	if !errors.Is(err, ErrDirtyWorktree) {
		t.Fatalf("expected ErrDirtyWorktree, got %v", err)
	}
	// content must be preserved
	got, _ := r.ReadWorktree("settings.json")
	if string(got) != "main-v1-edited\n" {
		t.Fatalf("worktree overwritten: %q", got)
	}
}
