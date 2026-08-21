package git

import (
	"strings"
	"testing"
)

func TestUnifiedDiff(t *testing.T) {
	patch := Unified("a.txt", "a.txt", []byte("one\ntwo\n"), []byte("one\nTWO\n"))
	if !strings.Contains(patch, "-two\n") || !strings.Contains(patch, "+TWO\n") {
		t.Fatalf("patch:\n%s", patch)
	}
	if !strings.Contains(patch, "--- a/a.txt") {
		t.Fatal(patch)
	}
}

func TestDiffPathUnstaged(t *testing.T) {
	r := initRepo(t, SHA1)
	_ = r.WriteWorktree("f.txt", "base\n")
	_, _ = r.Add([]string{"f.txt"})
	_, _ = r.Commit("c", "T <t@t>")
	_ = r.WriteWorktree("f.txt", "base\nedit\n")
	d, err := r.DiffPath("f.txt", "unstaged")
	if err != nil {
		t.Fatal(err)
	}
	if d.Patch == "" || !strings.Contains(d.Patch, "+edit\n") {
		t.Fatalf("unexpected patch: %s", d.Patch)
	}
}
