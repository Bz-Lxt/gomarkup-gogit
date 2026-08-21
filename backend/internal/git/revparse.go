package git

import (
	"fmt"
	"strings"
)

func (r *Repo) RevParse(q string) (string, string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", "", fmt.Errorf("%w: revision is required", ErrValidation)
	}
	if q == "HEAD" {
		oid, err := r.ResolveHEAD()
		if err != nil {
			return "", "", err
		}
		return oid, "commit", nil
	}
	if strings.HasPrefix(q, "refs/heads/") {
		name := strings.TrimPrefix(q, "refs/heads/")
		if oid, err := r.ResolveBranch(name); err == nil {
			return oid, "commit", nil
		}
		q = name
	}
	if oid, err := r.ResolveBranch(q); err == nil {
		return oid, "commit", nil
	}
	if r.algo.ValidHex(q) {
		if _, _, err := r.store.Read(q); err == nil {
			return q, "object", nil
		}
		return "", "", fmt.Errorf("%w: object %s", ErrNotFound, q)
	}
	if len(q) < 4 || len(q) >= r.algo.HexLen() {
		return "", "", fmt.Errorf("%w: unknown revision %s", ErrNotFound, q)
	}
	for _, c := range q {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", "", fmt.Errorf("%w: unknown revision %s", ErrNotFound, q)
		}
	}
	q = strings.ToLower(q)
	matches, err := r.store.ListPrefix(q)
	if err != nil {
		return "", "", err
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("%w: object prefix %s", ErrNotFound, q)
	}
	if len(matches) > 1 {
		return "", "", fmt.Errorf("%w: ambiguous prefix %s (%d matches)", ErrValidation, q, len(matches))
	}
	return matches[0], "object", nil
}
