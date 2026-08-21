package git

import (
	"bytes"
	"compress/zlib"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlobHashMatchesGitVectors(t *testing.T) {
	algo := SHA1
	raw := EncodeObject(TypeBlob, []byte("hello\n"))
	got := algo.Sum(raw)
	want := "ce013625030ba8dba906f756967f9e9ca394464a"
	if got != want {
		t.Fatalf("hello blob sha1 = %s want %s", got, want)
	}
	empty := algo.Sum(EncodeObject(TypeBlob, nil))
	if empty != "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391" {
		t.Fatalf("empty blob sha1 = %s", empty)
	}
}

func TestSHA256BlobRoundtrip(t *testing.T) {
	raw := EncodeObject(TypeBlob, []byte("hello\n"))
	sum := SHA256.Sum(raw)
	if len(sum) != 64 {
		t.Fatalf("sha256 hex len %d", len(sum))
	}
}

func TestZlibObjectRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, SHA1)
	oid, err := s.Write(TypeBlob, []byte("payload-中文"))
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := s.ReadRawCompressed(oid)
	if err != nil {
		t.Fatal(err)
	}
	if !IsZlib(compressed) {
		t.Fatal("disk object is not zlib")
	}
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatal(err)
	}
	_ = zr.Close()
	typ, content, err := DecodeObject(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if typ != TypeBlob || string(content) != "payload-中文" {
		t.Fatalf("decoded %s %q", typ, content)
	}
}

func TestAddCommitCheckoutRestore(t *testing.T) {
	r := initRepo(t, SHA1)
	if err := r.WriteWorktree("a.txt", "alpha\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"a.txt"}); err != nil {
		t.Fatal(err)
	}
	c1, err := r.Commit("first", "T <t@t>")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateBranch("other"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("a.txt", "beta\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"a.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("second", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	if err := r.Checkout("other"); err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadWorktree("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "alpha\n" {
		t.Fatalf("restored %q", got)
	}
	if err := r.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	got, err = r.ReadWorktree("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "beta\n" {
		t.Fatalf("main restored %q", got)
	}
	obj, err := r.Inspect(c1.Hash)
	if err != nil || obj["type"] != TypeCommit {
		t.Fatalf("inspect: %v %v", obj, err)
	}
}

func TestThreeWayMergeNoConflictAndConflict(t *testing.T) {
	r := initRepo(t, SHA1)
	if err := r.WriteWorktree("keep.txt", "base-keep\n"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("both.txt", "base-both\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"keep.txt", "both.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("base", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateBranch("side"); err != nil {
		t.Fatal(err)
	}

	if err := r.WriteWorktree("keep.txt", "ours-keep\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"keep.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("ours keep", "T <t@t>"); err != nil {
		t.Fatal(err)
	}

	if err := r.Checkout("side"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("extra.txt", "theirs-only\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"extra.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit("theirs extra", "T <t@t>"); err != nil {
		t.Fatal(err)
	}

	if err := r.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	res, err := r.Merge("side")
	if err != nil {
		t.Fatal(err)
	}
	if res.Commit == nil || len(res.Commit.Parents) != 2 {
		t.Fatalf("expected merge commit, got %+v", res.Commit)
	}
	extra, err := r.ReadWorktree("extra.txt")
	if err != nil || string(extra) != "theirs-only\n" {
		t.Fatalf("merged extra: %s %v", extra, err)
	}
	keep, _ := r.ReadWorktree("keep.txt")
	if string(keep) != "ours-keep\n" {
		t.Fatalf("keep overwritten: %s", keep)
	}

	// conflict: both edit both.txt
	r2 := initRepo(t, SHA1)
	_ = r2.WriteWorktree("f.txt", "line1\nline2\n")
	_, _ = r2.Add([]string{"f.txt"})
	_, _ = r2.Commit("b", "T <t@t>")
	_, _ = r2.CreateBranch("x")
	_ = r2.WriteWorktree("f.txt", "line1\nOURS\n")
	_, _ = r2.Add([]string{"f.txt"})
	_, _ = r2.Commit("ours", "T <t@t>")
	_ = r2.Checkout("x")
	_ = r2.WriteWorktree("f.txt", "line1\nTHEIRS\n")
	_, _ = r2.Add([]string{"f.txt"})
	_, _ = r2.Commit("theirs", "T <t@t>")
	_ = r2.Checkout("main")
	res, err = r2.Merge("x")
	if !errors.Is(err, ErrMergeConflict) {
		t.Fatalf("want merge conflict, got %v", err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "f.txt" {
		t.Fatalf("conflicts %+v", res.Conflicts)
	}
	body, _ := r2.ReadWorktree("f.txt")
	s := string(body)
	if !strings.Contains(s, "<<<<<<<") || !strings.Contains(s, "=======") || !strings.Contains(s, ">>>>>>>") {
		t.Fatalf("missing markers:\n%s", s)
	}
}

func TestSHA256FullLoop(t *testing.T) {
	r := initRepo(t, SHA256)
	if err := r.WriteWorktree("n.txt", "sha256-ok\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add([]string{"n.txt"}); err != nil {
		t.Fatal(err)
	}
	c, err := r.Commit("s", "T <t@t>")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Hash) != 64 {
		t.Fatalf("hash len %d", len(c.Hash))
	}
}

func TestPathTraversalRejected(t *testing.T) {
	r := initRepo(t, SHA1)
	if err := r.WriteWorktree("../escape.txt", "x"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := r.Add([]string{".gogit/config"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorruptObjectRejected(t *testing.T) {
	_, _, err := DecodeObject([]byte("not-an-object"))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func initRepo(t *testing.T, algo Algo) *Repo {
	t.Helper()
	dir := t.TempDir()
	r, err := Init(dir, algo, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestIndexRoundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "index")
	idx := &Index{Entries: []IndexEntry{{Path: "a.txt", Mode: "100644", OID: strings.Repeat("a", 40), Size: 1, Mtime: 1}}}
	if err := SaveIndex(p, idx); err != nil {
		t.Fatal(err)
	}
	got, err := LoadIndex(p, SHA1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Path != "a.txt" {
		t.Fatal(got.Entries)
	}
	_ = os.WriteFile(p, []byte("garbage"), 0o644)
	if _, err := LoadIndex(p, SHA1); err == nil {
		t.Fatal("corrupt index accepted")
	}
}
