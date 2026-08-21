package git

import "testing"

func TestRevParseHEADAndPrefix(t *testing.T) {
	r := initRepo(t, SHA1)
	_ = r.WriteWorktree("f.txt", "x\n")
	_, _ = r.Add([]string{"f.txt"})
	c, err := r.Commit("c", "T <t@t>")
	if err != nil {
		t.Fatal(err)
	}
	oid, kind, err := r.RevParse("HEAD")
	if err != nil || oid != c.Hash || kind != "commit" {
		t.Fatalf("HEAD %s %s %v", oid, kind, err)
	}
	oid, _, err = r.RevParse("main")
	if err != nil || oid != c.Hash {
		t.Fatalf("branch %s %v", oid, err)
	}
	oid, _, err = r.RevParse(c.Hash[:8])
	if err != nil || oid != c.Hash {
		t.Fatalf("prefix %s %v", oid, err)
	}
	if _, _, err := r.RevParse("deadbeef"); err == nil {
		t.Fatal("expected missing")
	}
}
