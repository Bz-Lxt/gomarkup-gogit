package git

import "testing"

func TestIgnorePatterns(t *testing.T) {
	ig := &IgnoreSet{rules: []ignoreRule{
		{pattern: "*.tmp"},
		{pattern: "build", dirOnly: true},
		{pattern: "keep.tmp", neg: true},
		{pattern: "docs/*.bak"},
	}}
	if !ig.Match("foo.tmp", false) {
		t.Fatal("*.tmp should match foo.tmp")
	}
	if ig.Match("keep.tmp", false) {
		t.Fatal("negation should un-ignore keep.tmp")
	}
	if !ig.Match("build/out.o", false) {
		t.Fatal("dir rule should match children")
	}
	if !ig.Match("docs/old.bak", false) {
		t.Fatal("docs/*.bak should match")
	}
	if ig.Match("src/hello.txt", false) {
		t.Fatal("hello.txt should not be ignored")
	}
}

func TestIgnoreFileLoaded(t *testing.T) {
	r := initRepo(t, SHA1)
	if err := r.WriteWorktree(".gogitignore", "*.tmp\n"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("a.tmp", "x"); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteWorktree("a.txt", "y"); err != nil {
		t.Fatal(err)
	}
	if !r.IsIgnored("a.tmp") {
		t.Fatal("expected a.tmp ignored")
	}
	added, err := r.Add([]string{"."})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range added {
		if e.Path == "a.tmp" {
			t.Fatal("ignored file was added")
		}
	}
}
