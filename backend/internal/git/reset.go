package git

import (
	"fmt"
	"strings"
)

func (r *Repo) Unstage(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("%w: paths is required", ErrValidation)
	}
	idx, err := r.loadIndex()
	if err != nil {
		return err
	}
	headMap := map[string]FlatEntry{}
	if head, err := r.ResolveHEAD(); err == nil {
		c, err := r.ReadCommit(head)
		if err != nil {
			return err
		}
		headMap, err = r.FlattenMap(c.Tree)
		if err != nil {
			return err
		}
	} else if !isUnborn(err) {
		return err
	}

	changed := false
	for _, p := range paths {
		rel, err := normalizeRel(p)
		if err != nil {
			return err
		}
		if rel == "" {
			return fmt.Errorf("%w: path is required", ErrInvalidPath)
		}
		if he, ok := headMap[rel]; ok {
			blob, err := r.ReadBlob(he.OID)
			if err != nil {
				return err
			}
			idx.Upsert(IndexEntry{Path: rel, Mode: he.Mode, OID: he.OID, Size: int64(len(blob))})
			changed = true
			continue
		}
		if _, ok := idx.Map()[rel]; ok {
			idx.Remove(rel)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return SaveIndex(r.indexPath(), idx)
}

func (r *Repo) RestoreWorktree(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("%w: paths is required", ErrValidation)
	}
	head, err := r.ResolveHEAD()
	if err != nil {
		return err
	}
	c, err := r.ReadCommit(head)
	if err != nil {
		return err
	}
	headMap, err := r.FlattenMap(c.Tree)
	if err != nil {
		return err
	}
	start := 0
	if len(paths) > 1 {
		start = 1
	}
	for _, p := range paths[start:] {
		rel, err := normalizeRel(p)
		if err != nil {
			return err
		}
		he, ok := headMap[rel]
		if !ok {
			return fmt.Errorf("%w: %s is not in HEAD", ErrNotFound, rel)
		}
		blob, err := r.ReadBlob(he.OID)
		if err != nil {
			return err
		}
		if err := r.writeWorktreeFile(rel, blob, he.Mode); err != nil {
			return err
		}
	}
	return nil
}

func NormalizeResetMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "mixed"
	}
	switch mode {
	case "mixed", "worktree":
		return mode, nil
	default:
		return "", fmt.Errorf("%w: reset mode must be mixed or worktree", ErrValidation)
	}
}

func (r *Repo) ResetPaths(paths []string, mode string) error {
	mode, err := NormalizeResetMode(mode)
	if err != nil {
		return err
	}
	if err := r.Unstage(paths); err != nil {
		return err
	}
	if mode == "worktree" {
		return r.RestoreWorktree(paths)
	}
	return nil
}
