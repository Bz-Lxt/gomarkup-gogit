package git

import (
	"fmt"
	"strings"
)

type MergeResult struct {
	Commit      *Commit  `json:"commit,omitempty"`
	FastForward bool     `json:"fast_forward"`
	Conflicts   []string `json:"conflicts,omitempty"`
}

func (r *Repo) Merge(branch string) (MergeResult, error) {
	if err := validateBranchName(branch); err != nil {
		return MergeResult{}, err
	}
	if r.MergeInProgress() {
		return MergeResult{}, ErrMergeInProgress
	}
	oursBr, err := r.CurrentBranch()
	if err != nil {
		return MergeResult{}, err
	}
	if branch == oursBr {
		return MergeResult{}, fmt.Errorf("%w: cannot merge a branch into itself", ErrValidation)
	}
	oursOID, err := r.ResolveHEAD()
	if err != nil {
		return MergeResult{}, err
	}
	theirsOID, err := r.ResolveBranch(branch)
	if err != nil {
		return MergeResult{}, err
	}
	if oursOID == theirsOID {
		return MergeResult{}, ErrAlreadyUpToDate
	}
	oursAnc, err := r.ancestorSet(oursOID)
	if err != nil {
		return MergeResult{}, err
	}
	if oursAnc[theirsOID] {
		return MergeResult{}, ErrAlreadyUpToDate
	}
	theirsAnc, err := r.ancestorSet(theirsOID)
	if err != nil {
		return MergeResult{}, err
	}
	if theirsAnc[oursOID] {
		if err := r.Checkout(branch); err != nil {
			// fast-forward: move current branch pointer then restore HEAD name
			return MergeResult{}, err
		}
		// Checkout switched HEAD to `branch`. Move our original branch pointer and switch back.
		if err := r.WriteBranch(oursBr, theirsOID); err != nil {
			return MergeResult{}, err
		}
		if err := r.WriteHEAD(oursBr); err != nil {
			return MergeResult{}, err
		}
		c, err := r.ReadCommit(theirsOID)
		if err != nil {
			return MergeResult{}, err
		}
		return MergeResult{Commit: &c, FastForward: true}, nil
	}

	baseOID, err := r.mergeBase(oursOID, theirsOID)
	if err != nil {
		return MergeResult{}, err
	}
	oursC, err := r.ReadCommit(oursOID)
	if err != nil {
		return MergeResult{}, err
	}
	theirsC, err := r.ReadCommit(theirsOID)
	if err != nil {
		return MergeResult{}, err
	}
	oursMap, err := r.FlattenMap(oursC.Tree)
	if err != nil {
		return MergeResult{}, err
	}
	theirsMap, err := r.FlattenMap(theirsC.Tree)
	if err != nil {
		return MergeResult{}, err
	}
	baseMap := map[string]FlatEntry{}
	if baseOID != "" {
		bc, err := r.ReadCommit(baseOID)
		if err != nil {
			return MergeResult{}, err
		}
		baseMap, err = r.FlattenMap(bc.Tree)
		if err != nil {
			return MergeResult{}, err
		}
	}

	paths := map[string]struct{}{}
	for p := range oursMap {
		paths[p] = struct{}{}
	}
	for p := range theirsMap {
		paths[p] = struct{}{}
	}
	for p := range baseMap {
		paths[p] = struct{}{}
	}

	merged := map[string]IndexEntry{}
	var conflicts []string
	conflictBodies := map[string][]byte{}

	for p := range paths {
		b := baseMap[p]
		o := oursMap[p]
		t := theirsMap[p]
		res, body, conf, err := r.mergeFile(p, b, o, t, oursBr, branch)
		if err != nil {
			return MergeResult{}, err
		}
		if conf {
			conflicts = append(conflicts, p)
			conflictBodies[p] = body
			continue
		}
		if res != nil {
			merged[p] = *res
		}
	}

	if len(conflicts) > 0 {
		// materialize union of auto-merged + conflicted files
		for p, e := range merged {
			blob, err := r.ReadBlob(e.OID)
			if err != nil {
				return MergeResult{}, err
			}
			if err := r.writeWorktreeFile(p, blob, e.Mode); err != nil {
				return MergeResult{}, err
			}
		}
		for p, body := range conflictBodies {
			if err := r.writeWorktreeFile(p, body, "100644"); err != nil {
				return MergeResult{}, err
			}
		}
		idx := &Index{}
		for _, e := range merged {
			idx.Upsert(e)
		}
		for p, body := range conflictBodies {
			oid, err := r.store.Write(TypeBlob, body)
			if err != nil {
				return MergeResult{}, err
			}
			idx.Upsert(IndexEntry{Path: p, Mode: "100644", OID: oid, Size: int64(len(body))})
		}
		if err := SaveIndex(r.indexPath(), idx); err != nil {
			return MergeResult{}, err
		}
		msg := fmt.Sprintf("Merge branch '%s' into %s\n", branch, oursBr)
		if err := r.writeMergeState(theirsOID, msg); err != nil {
			return MergeResult{}, err
		}
		return MergeResult{Conflicts: conflicts}, ErrMergeConflict
	}

	idx := &Index{}
	for _, e := range merged {
		idx.Upsert(e)
	}
	if err := SaveIndex(r.indexPath(), idx); err != nil {
		return MergeResult{}, err
	}
	// write worktree
	old, _ := r.FlattenMap(oursC.Tree)
	newT := map[string]FlatEntry{}
	for p, e := range merged {
		newT[p] = FlatEntry{Path: p, Mode: e.Mode, OID: e.OID, Type: TypeBlob}
	}
	if err := r.checkoutTree(old, newT); err != nil {
		return MergeResult{}, err
	}
	msg := fmt.Sprintf("Merge branch '%s' into %s", branch, oursBr)
	if err := r.writeMergeState(theirsOID, msg+"\n"); err != nil {
		return MergeResult{}, err
	}
	c, err := r.Commit(msg, "GoGit <gogit@local>")
	if err != nil {
		return MergeResult{}, err
	}
	return MergeResult{Commit: &c}, nil
}

func (r *Repo) mergeFile(path string, base, ours, theirs FlatEntry, oursBr, theirsBr string) (*IndexEntry, []byte, bool, error) {
	bOID, oOID, tOID := base.OID, ours.OID, theirs.OID
	if oOID == tOID {
		if oOID == "" {
			return nil, nil, false, nil
		}
		e := IndexEntry{Path: path, Mode: pickMode(ours, theirs), OID: oOID}
		return &e, nil, false, nil
	}
	if oOID == bOID {
		if tOID == "" {
			return nil, nil, false, nil
		}
		e := IndexEntry{Path: path, Mode: pickMode(theirs, ours), OID: tOID}
		return &e, nil, false, nil
	}
	if tOID == bOID {
		if oOID == "" {
			return nil, nil, false, nil
		}
		e := IndexEntry{Path: path, Mode: pickMode(ours, theirs), OID: oOID}
		return &e, nil, false, nil
	}
	// both changed
	var bData, oData, tData []byte
	var err error
	if bOID != "" {
		bData, err = r.ReadBlob(bOID)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if oOID != "" {
		oData, err = r.ReadBlob(oOID)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if tOID != "" {
		tData, err = r.ReadBlob(tOID)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if (oOID != "" && IsBinary(oData)) || (tOID != "" && IsBinary(tData)) || (bOID != "" && IsBinary(bData)) {
		body := []byte(fmt.Sprintf("[binary conflict] ours=%s theirs=%s\n", oOID, tOID))
		if oData != nil {
			body = oData
		}
		return nil, body, true, nil
	}
	merged, conflict := merge3Text(string(bData), string(oData), string(tData), oursBr, theirsBr)
	if conflict {
		return nil, []byte(merged), true, nil
	}
	oid, err := r.store.Write(TypeBlob, []byte(merged))
	if err != nil {
		return nil, nil, false, err
	}
	e := IndexEntry{Path: path, Mode: pickMode(ours, theirs), OID: oid, Size: int64(len(merged))}
	return &e, nil, false, nil
}

func pickMode(a, b FlatEntry) string {
	if a.Mode != "" {
		return a.Mode
	}
	if b.Mode != "" {
		return b.Mode
	}
	return "100644"
}

func merge3Text(base, ours, theirs, oursLabel, theirsLabel string) (string, bool) {
	bL := splitKeep(base)
	oL := splitKeep(ours)
	tL := splitKeep(theirs)
	merged, conflict := diff3(bL, oL, tL, oursLabel, theirsLabel)
	return strings.Join(merged, ""), conflict
}

func splitKeep(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

type hunk struct {
	a0, a1 int
	b0, b1 int
}

func diff3(base, ours, theirs []string, oursLabel, theirsLabel string) ([]string, bool) {
	ha := hunksOf(base, ours)
	hb := hunksOf(base, theirs)
	var out []string
	conflict := false
	i, j := 0, 0
	pos := 0
	for i < len(ha) || j < len(hb) {
		nextA, nextB := len(base), len(base)
		if i < len(ha) {
			nextA = ha[i].a0
		}
		if j < len(hb) {
			nextB = hb[j].a0
		}
		next := nextA
		if nextB < next {
			next = nextB
		}
		if next > pos {
			out = append(out, base[pos:next]...)
			pos = next
		}
		if i < len(ha) && j < len(hb) && overlap(ha[i], hb[j]) {
			aEnd, bEnd := ha[i].a1, hb[j].a1
			aB0, aB1 := ha[i].b0, ha[i].b1
			bB0, bB1 := hb[j].b0, hb[j].b1
			i++
			j++
			for i < len(ha) && ha[i].a0 < aEnd+1 && overlap(hunk{a0: next, a1: aEnd, b0: aB0, b1: aB1}, ha[i]) ||
				(j < len(hb) && hb[j].a0 < bEnd+1 && overlap(hunk{a0: next, a1: bEnd, b0: bB0, b1: bB1}, hb[j])) {
				if i < len(ha) && ha[i].a0 <= aEnd {
					if ha[i].a1 > aEnd {
						aEnd = ha[i].a1
					}
					aB1 = ha[i].b1
					i++
				} else if j < len(hb) && hb[j].a0 <= bEnd {
					if hb[j].a1 > bEnd {
						bEnd = hb[j].a1
					}
					bB1 = hb[j].b1
					j++
				} else {
					break
				}
			}
			oSlice := sliceSafe(ours, aB0, aB1)
			tSlice := sliceSafe(theirs, bB0, bB1)
			if sameLines(oSlice, tSlice) {
				out = append(out, oSlice...)
			} else {
				conflict = true
				out = append(out, "<<<<<<< "+oursLabel+"\n")
				out = append(out, oSlice...)
				out = append(out, "=======\n")
				out = append(out, tSlice...)
				out = append(out, ">>>>>>> "+theirsLabel+"\n")
			}
			if aEnd > bEnd {
				pos = aEnd
			} else {
				pos = bEnd
			}
			continue
		}
		if i < len(ha) && nextA <= nextB {
			out = append(out, sliceSafe(ours, ha[i].b0, ha[i].b1)...)
			pos = ha[i].a1
			i++
			continue
		}
		if j < len(hb) {
			out = append(out, sliceSafe(theirs, hb[j].b0, hb[j].b1)...)
			pos = hb[j].a1
			j++
		}
	}
	if pos < len(base) {
		out = append(out, base[pos:]...)
	}
	return out, conflict
}

func overlap(a, b hunk) bool {
	return a.a0 < b.a1 && b.a0 < a.a1 || a.a0 == b.a0
}

func sliceSafe(s []string, i, j int) []string {
	if i < 0 {
		i = 0
	}
	if j > len(s) {
		j = len(s)
	}
	if i >= j {
		return nil
	}
	return s[i:j]
}

func sameLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hunksOf(a, b []string) []hunk {
	ops := lcsDiff(a, b)
	var hs []hunk
	var cur *hunk
	ia, ib := 0, 0
	flush := func() {
		if cur != nil {
			hs = append(hs, *cur)
			cur = nil
		}
	}
	for _, op := range ops {
		switch op {
		case 0: // eq
			flush()
			ia++
			ib++
		case 1: // del from a
			if cur == nil {
				cur = &hunk{a0: ia, a1: ia, b0: ib, b1: ib}
			}
			cur.a1 = ia + 1
			ia++
		case 2: // ins into b
			if cur == nil {
				cur = &hunk{a0: ia, a1: ia, b0: ib, b1: ib}
			}
			cur.b1 = ib + 1
			ib++
		}
	}
	flush()
	return hs
}

// lcsDiff returns a sequence of ops: 0=eq, 1=del, 2=ins
func lcsDiff(a, b []string) []int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []int
	i, j := 0, 0
	for i < m && j < n {
		if a[i] == b[j] {
			ops = append(ops, 0)
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, 1)
			i++
		} else {
			ops = append(ops, 2)
			j++
		}
	}
	for i < m {
		ops = append(ops, 1)
		i++
	}
	for j < n {
		ops = append(ops, 2)
		j++
	}
	return ops
}
