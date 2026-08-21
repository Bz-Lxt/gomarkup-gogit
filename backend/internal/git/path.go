package git

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const maxFileBytes = 2 * 1024 * 1024

func validateRelPath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return fmt.Errorf("%w: path is required", ErrInvalidPath)
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if path.IsAbs(p) || strings.HasPrefix(p, "/") {
		return fmt.Errorf("%w: absolute path rejected", ErrInvalidPath)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "." {
			return fmt.Errorf("%w: path traversal rejected", ErrInvalidPath)
		}
		if seg == "" {
			return fmt.Errorf("%w: empty path segment", ErrInvalidPath)
		}
	}
	if p == ".gogit" || strings.HasPrefix(p, ".gogit/") {
		return fmt.Errorf("%w: .gogit is reserved", ErrInvalidPath)
	}
	if p == ".git" || strings.HasPrefix(p, ".git/") {
		return fmt.Errorf("%w: .git is reserved", ErrInvalidPath)
	}
	return nil
}

func normalizeRel(p string) (string, error) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	p = strings.TrimPrefix(p, "./")
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return "", nil
	}
	if err := validateRelPath(p); err != nil {
		return "", err
	}
	return path.Clean(p), nil
}

func (r *Repo) abs(rel string) (string, error) {
	rel, err := normalizeRel(rel)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return r.workDir, nil
	}
	full := filepath.Join(r.workDir, filepath.FromSlash(rel))
	// ensure still under workDir
	relBack, err := filepath.Rel(r.workDir, full)
	if err != nil || strings.HasPrefix(relBack, "..") {
		return "", fmt.Errorf("%w: escaped worktree", ErrInvalidPath)
	}
	return full, nil
}

func fileMode(info os.FileInfo) string {
	if info.Mode()&0o111 != 0 {
		return "100755"
	}
	return "100644"
}

func skipName(name string) bool {
	return name == ".gogit" || name == ".git" || name == ".DS_Store"
}
