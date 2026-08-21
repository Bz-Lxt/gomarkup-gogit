package git

import "testing"

func TestUnstageAndRestore(t *testing.T) {
	r := initRepo(t, SHA1)
	_ = r.WriteWorktree("f.txt", "v1\n")
	_, _ = r.Add([]string{"f.txt"})
	_, _ = r.Commit("c1", "T <t@t>")
	_ = r.WriteWorktree("f.txt", "v2\n")
	_, _ = r.Add([]string{"f.txt"})
	st, err := r.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Staged) == 0 {
		t.Fatal("expected staged change")
	}
	if err := r.Unstage([]string{"f.txt"}); err != nil {
		t.Fatal(err)
	}
	st, _ = r.Status()
	if len(st.Staged) != 0 {
		t.Fatalf("still staged: %+v", st.Staged)
	}
	if err := r.RestoreWorktree([]string{"f.txt"}); err != nil {
		t.Fatal(err)
	}
	got, _ := r.ReadWorktree("f.txt")
	if string(got) != "v1\n" {
		t.Fatalf("restored %q", got)
	}
}

func TestResetModeReject(t *testing.T) {
	if _, err := NormalizeResetMode("hard"); err == nil {
		t.Fatal("hard should be rejected")
	}
}
