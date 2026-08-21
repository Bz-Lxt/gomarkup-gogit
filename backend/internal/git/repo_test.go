package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddDirectoryWithBrokenSymlinkFails(t *testing.T) {
	r := initRepo(t, SHA1)

	dir := filepath.Join(r.WorkDir(), "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// broken symlink: target does not exist
	broken := filepath.Join(dir, "missing-link")
	if err := os.Symlink(filepath.Join(dir, "no-such-target"), broken); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	// snapshot the index before staging
	before, err := r.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	beforePaths := make(map[string]bool, len(before.Entries))
	for _, e := range before.Entries {
		beforePaths[e.Path] = true
	}

	_, err = r.Add([]string{"docs"})
	if err == nil {
		t.Fatal("expected error when staging a directory with a broken symlink")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for broken link, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing-link") {
		t.Fatalf("error should mention the broken entry, got %q", err.Error())
	}

	// index on disk must be unchanged from before staging
	after, err := r.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Entries) != len(before.Entries) {
		t.Fatalf("index changed on failure: before=%d after=%d", len(before.Entries), len(after.Entries))
	}
	for _, e := range after.Entries {
		if !beforePaths[e.Path] {
			t.Fatalf("index gained entry %s after a failed Add", e.Path)
		}
	}
	// specifically, the good file must NOT have been staged
	for _, e := range after.Entries {
		if e.Path == "docs/good.txt" {
			t.Fatal("docs/good.txt was staged despite the Add failing")
		}
	}
}

func TestAddDirectoryHappyPath(t *testing.T) {
	r := initRepo(t, SHA1)

	dir := filepath.Join(r.WorkDir(), "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	added, err := r.Add([]string{"pkg"})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range added {
		seen[e.Path] = true
	}
	if !seen["pkg/a.txt"] || !seen["pkg/sub/b.txt"] {
		t.Fatalf("missing entries, got %v", seen)
	}
}
