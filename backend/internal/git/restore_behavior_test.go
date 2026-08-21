package git

import "testing"

func TestRestoreWorktreeRestoresEverySelectedPath(t *testing.T) {
	r := initRepo(t, SHA1)
	for _, path := range []string{"README.md", "CHANGELOG.md"} {
		if err := r.WriteWorktree(path, "committed\n"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Add([]string{"README.md", "CHANGELOG.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("initial", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"README.md", "CHANGELOG.md"} {
		if err := r.WriteWorktree(path, "local edit\n"); err != nil {
			t.Fatal(err)
		}
	}

	if err := r.RestoreWorktree([]string{"README.md", "CHANGELOG.md"}); err != nil {
		t.Fatalf("restore returned an error: %v", err)
	}
	for _, path := range []string{"README.md", "CHANGELOG.md"} {
		got, err := r.ReadWorktree(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "committed\n" {
			t.Fatalf("%s content = %q after restore, want committed content", path, got)
		}
	}
}
