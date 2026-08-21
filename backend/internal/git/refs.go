package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func (r *Repo) HEADPath() string { return filepath.Join(r.gitDir, "HEAD") }
func (r *Repo) headsDir() string { return filepath.Join(r.gitDir, "refs", "heads") }
func (r *Repo) branchPath(name string) string {
	return filepath.Join(r.headsDir(), filepath.FromSlash(name))
}

func validateBranchName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: branch name is required", ErrValidation)
	}
	if len(name) > 64 {
		return fmt.Errorf("%w: branch name too long", ErrValidation)
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("%w: invalid branch name", ErrValidation)
	}
	if strings.Contains(name, "//") {
		return fmt.Errorf("%w: invalid branch name", ErrValidation)
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("%w: branch name has illegal character", ErrValidation)
	}
	return nil
}

func (r *Repo) CurrentBranch() (string, error) {
	b, err := os.ReadFile(r.HEADPath())
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	const prefix = "ref: refs/heads/"
	if !strings.HasPrefix(s, prefix) {
		return "", fmt.Errorf("%w: detached HEAD not supported", ErrValidation)
	}
	name := strings.TrimPrefix(s, prefix)
	if err := validateBranchName(name); err != nil {
		return "", err
	}
	return name, nil
}

func (r *Repo) ResolveHEAD() (string, error) {
	name, err := r.CurrentBranch()
	if err != nil {
		return "", err
	}
	return r.ResolveBranch(name)
}

func (r *Repo) ResolveBranch(name string) (string, error) {
	if err := validateBranchName(name); err != nil {
		return "", err
	}
	b, err := os.ReadFile(r.branchPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: branch %s", ErrUnbornHEAD, name)
		}
		return "", err
	}
	oid := strings.TrimSpace(string(b))
	if !r.algo.ValidHex(oid) {
		return "", fmt.Errorf("%w: corrupt ref %s", ErrValidation, name)
	}
	return oid, nil
}

func (r *Repo) WriteHEAD(branch string) error {
	if err := validateBranchName(branch); err != nil {
		return err
	}
	content := "ref: refs/heads/" + branch + "\n"
	return os.WriteFile(r.HEADPath(), []byte(content), 0o644)
}

func (r *Repo) WriteBranch(name, oid string) error {
	if err := validateBranchName(name); err != nil {
		return err
	}
	if !r.algo.ValidHex(oid) {
		return fmt.Errorf("%w: invalid oid for ref", ErrValidation)
	}
	path := r.branchPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(oid+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (r *Repo) ListBranches() ([]BranchInfo, error) {
	current, err := r.CurrentBranch()
	if err != nil {
		return nil, err
	}
	var out []BranchInfo
	err = filepath.WalkDir(r.headsDir(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(r.headsDir(), path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		oid, err := r.ResolveBranch(name)
		if err != nil {
			return err
		}
		out = append(out, BranchInfo{Name: name, Hash: oid, Current: name == current})
		return nil
	})
	if os.IsNotExist(err) {
		return []BranchInfo{}, nil
	}
	return out, err
}

type BranchInfo struct {
	Name    string `json:"name"`
	Hash    string `json:"hash"`
	Current bool   `json:"current"`
}

func (r *Repo) mergeHeadPath() string { return filepath.Join(r.gitDir, "MERGE_HEAD") }
func (r *Repo) mergeMsgPath() string  { return filepath.Join(r.gitDir, "MERGE_MSG") }

func (r *Repo) MergeInProgress() bool {
	_, err := os.Stat(r.mergeHeadPath())
	return err == nil
}

func (r *Repo) ReadMergeHEAD() (string, error) {
	b, err := os.ReadFile(r.mergeHeadPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	oid := strings.TrimSpace(string(b))
	if !r.algo.ValidHex(oid) {
		return "", fmt.Errorf("%w: corrupt MERGE_HEAD", ErrValidation)
	}
	return oid, nil
}

func (r *Repo) writeMergeState(theirsOID, msg string) error {
	if err := os.WriteFile(r.mergeHeadPath(), []byte(theirsOID+"\n"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(r.mergeMsgPath(), []byte(msg), 0o644)
}

func (r *Repo) clearMergeState() {
	_ = os.Remove(r.mergeHeadPath())
	_ = os.Remove(r.mergeMsgPath())
}
