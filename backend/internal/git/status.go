package git

type FileStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type Status struct {
	Staged    []FileStatus `json:"staged"`
	Unstaged  []FileStatus `json:"unstaged"`
	Untracked []FileStatus `json:"untracked"`
}

func (r *Repo) Status() (Status, error) {
	idx, err := r.loadIndex()
	if err != nil {
		return Status{}, err
	}
	im := idx.Map()
	headMap := map[string]FlatEntry{}
	if head, err := r.ResolveHEAD(); err == nil {
		c, err := r.ReadCommit(head)
		if err != nil {
			return Status{}, err
		}
		headMap, err = r.FlattenMap(c.Tree)
		if err != nil {
			return Status{}, err
		}
	} else if !isUnborn(err) {
		return Status{}, err
	}

	wt, err := r.listAllWorktreeFiles()
	if err != nil {
		return Status{}, err
	}

	st := Status{Staged: []FileStatus{}, Unstaged: []FileStatus{}, Untracked: []FileStatus{}}
	seenWT := map[string]bool{}

	for _, p := range wt {
		seenWT[p] = true
		ie, inIndex := im[p]
		he, inHead := headMap[p]
		blob, err := r.hashWorktreeFile(p)
		if err != nil {
			return Status{}, err
		}
		if !inIndex {
			st.Untracked = append(st.Untracked, FileStatus{Path: p, Status: "untracked"})
			continue
		}
		if blob.OID != ie.OID {
			st.Unstaged = append(st.Unstaged, FileStatus{Path: p, Status: "modified"})
		}
		if !inHead {
			st.Staged = append(st.Staged, FileStatus{Path: p, Status: "added"})
		} else if ie.OID != he.OID {
			st.Staged = append(st.Staged, FileStatus{Path: p, Status: "modified"})
		}
	}

	for p, ie := range im {
		if seenWT[p] {
			continue
		}
		st.Unstaged = append(st.Unstaged, FileStatus{Path: p, Status: "deleted"})
		if he, ok := headMap[p]; !ok {
			st.Staged = append(st.Staged, FileStatus{Path: p, Status: "added"})
		} else if ie.OID != he.OID {
			st.Staged = append(st.Staged, FileStatus{Path: p, Status: "modified"})
		}
	}

	for p := range headMap {
		_, inIndex := im[p]
		if !inIndex && !seenWT[p] {
			st.Staged = append(st.Staged, FileStatus{Path: p, Status: "deleted"})
		}
	}
	return st, nil
}

func (r *Repo) listAllWorktreeFiles() ([]string, error) {
	ig := r.loadIgnore()
	var out []string
	err := walkWorktree(r.workDir, func(rel string) {
		if ig.Match(rel, false) {
			return
		}
		out = append(out, rel)
	})
	return out, err
}
