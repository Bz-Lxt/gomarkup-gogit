package git

import (
	"fmt"
	"strings"
)

type FileDiff struct {
	Path   string `json:"path"`
	Side   string `json:"side"`
	Binary bool   `json:"binary"`
	Patch  string `json:"patch"`
	OldOID string `json:"old_oid"`
	NewOID string `json:"new_oid"`
}

func Unified(oldName, newName string, oldB, newB []byte) string {
	if IsBinary(oldB) || IsBinary(newB) {
		return fmt.Sprintf("Binary files %s and %s differ\n", oldName, newName)
	}
	oldL := splitKeep(string(oldB))
	newL := splitKeep(string(newB))
	hs := hunksOf(oldL, newL)
	if len(hs) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", oldName, newName)
	for _, h := range hs {
		oldStart := h.a0 + 1
		if h.a0 == h.a1 && h.a0 == 0 {
			oldStart = 0
		}
		newStart := h.b0 + 1
		if h.b0 == h.b1 && h.b0 == 0 {
			newStart = 0
		}
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, h.a1-h.a0, newStart, h.b1-h.b0)
		ctx0 := h.a0 - 1
		if ctx0 >= 0 && ctx0 < len(oldL) {
			b.WriteString(" ")
			b.WriteString(ensureNL(oldL[ctx0]))
		}
		for i := h.a0; i < h.a1 && i < len(oldL); i++ {
			b.WriteString("-")
			b.WriteString(ensureNL(oldL[i]))
		}
		for i := h.b0; i < h.b1 && i < len(newL); i++ {
			b.WriteString("+")
			b.WriteString(ensureNL(newL[i]))
		}
	}
	return b.String()
}

func ensureNL(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func (r *Repo) DiffPath(rel, side string) (FileDiff, error) {
	rel, err := normalizeRel(rel)
	if err != nil {
		return FileDiff{}, err
	}
	if rel == "" {
		return FileDiff{}, fmt.Errorf("%w: path is required", ErrInvalidPath)
	}
	side = strings.ToLower(strings.TrimSpace(side))
	if side == "" {
		side = "unstaged"
	}
	if side != "unstaged" && side != "staged" {
		return FileDiff{}, fmt.Errorf("%w: side must be staged or unstaged", ErrValidation)
	}

	idx, err := r.loadIndex()
	if err != nil {
		return FileDiff{}, err
	}
	im := idx.Map()
	headMap := map[string]FlatEntry{}
	if head, err := r.ResolveHEAD(); err == nil {
		c, err := r.ReadCommit(head)
		if err != nil {
			return FileDiff{}, err
		}
		headMap, err = r.FlattenMap(c.Tree)
		if err != nil {
			return FileDiff{}, err
		}
	} else if !isUnborn(err) {
		return FileDiff{}, err
	}

	var oldB, newB []byte
	var oldOID, newOID string
	if side == "unstaged" {
		if ie, ok := im[rel]; ok {
			oldOID = ie.OID
			oldB, err = r.ReadBlob(ie.OID)
			if err != nil {
				return FileDiff{}, err
			}
		}
		if b, err := r.ReadWorktree(rel); err == nil {
			newB = b
			e, herr := r.hashWorktreeFile(rel)
			if herr == nil {
				newOID = e.OID
			}
		} else if !isNotFound(err) {
			return FileDiff{}, err
		}
	} else {
		if he, ok := headMap[rel]; ok {
			oldOID = he.OID
			oldB, err = r.ReadBlob(he.OID)
			if err != nil {
				return FileDiff{}, err
			}
		}
		if ie, ok := im[rel]; ok {
			newOID = ie.OID
			newB, err = r.ReadBlob(ie.OID)
			if err != nil {
				return FileDiff{}, err
			}
		}
	}

	d := FileDiff{
		Path:   rel,
		Side:   side,
		OldOID: oldOID,
		NewOID: newOID,
		Binary: IsBinary(oldB) || IsBinary(newB),
	}
	d.Patch = Unified(rel, rel, oldB, newB)
	return d, nil
}

func isNotFound(err error) bool {
	return err != nil && (err == ErrNotFound || strings.Contains(err.Error(), "not found"))
}
