package git

import (
	"fmt"
	"sort"
)

type FsckIssue struct {
	OID     string `json:"oid"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type FsckReport struct {
	ObjectCount int         `json:"object_count"`
	Reachable   int         `json:"reachable"`
	Dangling    []string    `json:"dangling"`
	Issues      []FsckIssue `json:"issues"`
	OK          bool        `json:"ok"`
}

func (r *Repo) Fsck() (FsckReport, error) {
	all, err := r.store.List()
	if err != nil {
		return FsckReport{}, err
	}
	rep := FsckReport{ObjectCount: len(all), Dangling: []string{}, Issues: []FsckIssue{}}
	reachable := map[string]bool{}
	tips, err := r.allTips()
	if err != nil {
		return FsckReport{}, err
	}
	for _, tip := range tips {
		if err := r.walkReachable(tip, reachable, &rep); err != nil {
			return FsckReport{}, err
		}
	}
	rep.Reachable = len(reachable)
	for _, oid := range all {
		if !reachable[oid] {
			rep.Dangling = append(rep.Dangling, oid)
		}
		if err := r.verifyObject(oid); err != nil {
			rep.Issues = append(rep.Issues, FsckIssue{OID: oid, Kind: "corrupt", Message: err.Error()})
		}
	}
	sort.Strings(rep.Dangling)
	rep.OK = len(rep.Issues) == 0
	return rep, nil
}

func (r *Repo) allTips() ([]string, error) {
	var tips []string
	if head, err := r.ResolveHEAD(); err == nil {
		tips = append(tips, head)
	} else if !isUnborn(err) {
		return nil, err
	}
	brs, err := r.ListBranches()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, t := range tips {
		seen[t] = true
	}
	for _, b := range brs {
		if !seen[b.Hash] {
			tips = append(tips, b.Hash)
			seen[b.Hash] = true
		}
	}
	return tips, nil
}

func (r *Repo) walkReachable(oid string, seen map[string]bool, rep *FsckReport) error {
	if oid == "" || seen[oid] {
		return nil
	}
	seen[oid] = true
	typ, content, err := r.store.Read(oid)
	if err != nil {
		rep.Issues = append(rep.Issues, FsckIssue{OID: oid, Kind: "missing", Message: err.Error()})
		return nil
	}
	switch typ {
	case TypeCommit:
		c, err := DecodeCommit(content, r.algo)
		if err != nil {
			rep.Issues = append(rep.Issues, FsckIssue{OID: oid, Kind: "corrupt", Message: err.Error()})
			return nil
		}
		if err := r.walkReachable(c.Tree, seen, rep); err != nil {
			return err
		}
		for _, p := range c.Parents {
			if err := r.walkReachable(p, seen, rep); err != nil {
				return err
			}
		}
	case TypeTree:
		ents, err := DecodeTree(content, r.algo)
		if err != nil {
			rep.Issues = append(rep.Issues, FsckIssue{OID: oid, Kind: "corrupt", Message: err.Error()})
			return nil
		}
		for _, e := range ents {
			if err := r.walkReachable(e.OID, seen, rep); err != nil {
				return err
			}
		}
	case TypeBlob:
		// leaf
	default:
		rep.Issues = append(rep.Issues, FsckIssue{OID: oid, Kind: "unknown", Message: fmt.Sprintf("type %s", typ)})
	}
	return nil
}

func (r *Repo) verifyObject(oid string) error {
	typ, content, err := r.store.Read(oid)
	if err != nil {
		return err
	}
	raw := EncodeObject(typ, content)
	got := r.algo.Sum(raw)
	if got != oid {
		return fmt.Errorf("hash mismatch: stored=%s computed=%s", oid, got)
	}
	return nil
}
