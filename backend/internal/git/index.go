package git

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type IndexEntry struct {
	Path  string `json:"path"`
	Mode  string `json:"mode"`
	OID   string `json:"oid"`
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime,omitempty"`
}

type Index struct {
	Entries []IndexEntry `json:"entries"`
}

func (idx *Index) Map() map[string]IndexEntry {
	m := make(map[string]IndexEntry, len(idx.Entries))
	for _, e := range idx.Entries {
		m[e.Path] = e
	}
	return m
}

func (idx *Index) Upsert(e IndexEntry) {
	for i, old := range idx.Entries {
		if old.Path == e.Path {
			idx.Entries[i] = e
			return
		}
	}
	idx.Entries = append(idx.Entries, e)
}

func (idx *Index) Remove(path string) {
	out := idx.Entries[:0]
	for _, e := range idx.Entries {
		if e.Path != path {
			out = append(out, e)
		}
	}
	idx.Entries = out
}

func (idx *Index) Sort() {
	sort.Slice(idx.Entries, func(i, j int) bool {
		return idx.Entries[i].Path < idx.Entries[j].Path
	})
}

func LoadIndex(path string, algo Algo) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{}, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	if !sc.Scan() {
		return nil, fmt.Errorf("%w: empty index", ErrValidation)
	}
	if strings.TrimSpace(sc.Text()) != "gogit-index-v1" {
		return nil, fmt.Errorf("%w: unknown index magic", ErrValidation)
	}
	if !sc.Scan() {
		return nil, fmt.Errorf("%w: index missing count", ErrValidation)
	}
	count, err := strconv.Atoi(strings.TrimSpace(sc.Text()))
	if err != nil || count < 0 {
		return nil, fmt.Errorf("%w: index count invalid", ErrValidation)
	}
	idx := &Index{Entries: make([]IndexEntry, 0, count)}
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			return nil, fmt.Errorf("%w: index entry must have 5 fields", ErrValidation)
		}
		e := IndexEntry{Path: parts[0], Mode: parts[1], OID: parts[2]}
		if err := validateRelPath(e.Path); err != nil {
			return nil, err
		}
		if e.Mode != "100644" && e.Mode != "100755" {
			return nil, fmt.Errorf("%w: invalid index mode %q", ErrValidation, e.Mode)
		}
		if !algo.ValidHex(e.OID) {
			return nil, fmt.Errorf("%w: invalid index oid", ErrValidation)
		}
		e.Size, err = strconv.ParseInt(parts[3], 10, 64)
		if err != nil || e.Size < 0 {
			return nil, fmt.Errorf("%w: invalid index size", ErrValidation)
		}
		e.Mtime, err = strconv.ParseInt(parts[4], 10, 64)
		if err != nil || e.Mtime < 0 {
			return nil, fmt.Errorf("%w: invalid index mtime", ErrValidation)
		}
		idx.Entries = append(idx.Entries, e)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(idx.Entries) != count {
		return nil, fmt.Errorf("%w: index count mismatch", ErrValidation)
	}
	return idx, nil
}

func SaveIndex(path string, idx *Index) error {
	idx.Sort()
	var b strings.Builder
	b.WriteString("gogit-index-v1\n")
	fmt.Fprintf(&b, "%d\n", len(idx.Entries))
	for _, e := range idx.Entries {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%d\t%d\n", e.Path, e.Mode, e.OID, e.Size, e.Mtime)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
