package git

import (
	"errors"
	"testing"
)

func TestValidateBranchName(t *testing.T) {
	cases := []struct {
		name string
		err  bool
	}{
		{"feature/login", false},
		{"feature/docs", false},
		{"feature/login/v2", false},
		{"main", false},
		{"feature-login", false},
		{"release_1", false},
		{"v1.2.3", false},

		{"", true},
		{"/leading", true},
		{"trailing/", true},
		{"feature//double", true},
		{"feature/../login", true},
		{"feature/./login", true},
		{"feature/login/.", true},
		{"feature~login", true},
		{"feature login", true},
		{"feature:login", true},
	}
	for _, c := range cases {
		err := validateBranchName(c.name)
		switch {
		case c.err && err == nil:
			t.Errorf("validateBranchName(%q): expected error, got nil", c.name)
		case !c.err && err != nil:
			t.Errorf("validateBranchName(%q): unexpected error %v", c.name, err)
		case c.err && err != nil && !errors.Is(err, ErrValidation):
			t.Errorf("validateBranchName(%q): error not ErrValidation: %v", c.name, err)
		}
	}
}

func TestCreateBranchHierarchical(t *testing.T) {
	r := initRepo(t, SHA1)
	_ = r.WriteWorktree("f.txt", "x\n")
	_, _ = r.Add([]string{"f.txt"})
	if _, err := r.Commit("init", "T <t@t>"); err != nil {
		t.Fatal(err)
	}
	info, err := r.CreateBranch("feature/login")
	if err != nil {
		t.Fatalf("CreateBranch(feature/login) failed: %v", err)
	}
	if info.Name != "feature/login" {
		t.Fatalf("CreateBranch name = %q want feature/login", info.Name)
	}

	// Duplicate should fail with ErrAlreadyExists.
	if _, err := r.CreateBranch("feature/login"); err == nil {
		t.Fatal("expected already-exists error, got nil")
	}

	list, err := r.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	var names []string
	for _, b := range list {
		names = append(names, b.Name)
	}
	found := false
	for _, n := range names {
		if n == "feature/login" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListBranches did not contain feature/login, got %v", names)
	}

	// Single-dot path segment must be rejected even though '.' is allowed.
	if _, err := r.CreateBranch("feature/./login"); err == nil {
		t.Fatal("expected validation error for feature/./login, got nil")
	}
}
