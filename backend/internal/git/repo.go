package git

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gogit/internal/logger"
)

type Repo struct {
	workDir string
	gitDir  string
	algo    Algo
	store   *Store
	log     *logger.Logger
}

func GitDir(workDir string) string { return filepath.Join(workDir, ".gogit") }

func Open(workDir string, log *logger.Logger) (*Repo, error) {
	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	gitDir := GitDir(workDir)
	if st, err := os.Stat(gitDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("%w: repository not initialized", ErrNotFound)
	}
	algo, err := readAlgo(gitDir)
	if err != nil {
		return nil, err
	}
	r := &Repo{workDir: workDir, gitDir: gitDir, algo: algo, store: NewStore(gitDir, algo), log: log}
	return r, nil
}

func Init(workDir string, algo Algo, log *logger.Logger) (*Repo, error) {
	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, err
	}
	gitDir := GitDir(workDir)
	if _, err := os.Stat(gitDir); err == nil {
		return nil, fmt.Errorf("%w: repository already exists", ErrAlreadyExists)
	}
	for _, d := range []string{
		filepath.Join(gitDir, "objects"),
		filepath.Join(gitDir, "refs", "heads"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	if err := DefaultConfig(algo).Save(gitDir); err != nil {
		return nil, err
	}
	r := &Repo{workDir: workDir, gitDir: gitDir, algo: algo, store: NewStore(gitDir, algo), log: log}
	if err := r.WriteHEAD("main"); err != nil {
		return nil, err
	}
	if err := SaveIndex(r.indexPath(), &Index{}); err != nil {
		return nil, err
	}
	if log != nil {
		log.Info("repository initialized", "path", workDir, "algo", string(algo))
	}
	return r, nil
}

func OpenOrInit(workDir string, algo Algo, log *logger.Logger, seed bool) (*Repo, error) {
	r, err := Open(workDir, log)
	if err == nil {
		return r, nil
	}
	r, err = Init(workDir, algo, log)
	if err != nil {
		return nil, err
	}
	if seed {
		if err := r.SeedDemo(); err != nil && log != nil {
			log.Error("seed demo failed", "err", err)
		}
	}
	return r, nil
}

func readAlgo(gitDir string) (Algo, error) {
	cfg, err := LoadConfig(gitDir)
	if err != nil {
		return SHA1, err
	}
	return cfg.HashAlgo, nil
}

func (r *Repo) WorkDir() string { return r.workDir }
func (r *Repo) Algo() Algo      { return r.algo }
func (r *Repo) indexPath() string {
	return filepath.Join(r.gitDir, "index")
}

func (r *Repo) loadIndex() (*Index, error) {
	return LoadIndex(r.indexPath(), r.algo)
}

func (r *Repo) Index() (*Index, error) {
	return r.loadIndex()
}

type RepoInfo struct {
	Path            string `json:"path"`
	HashAlgo        string `json:"hash_algo"`
	CurrentBranch   string `json:"current_branch"`
	Head            string `json:"head"`
	ObjectCount     int    `json:"object_count"`
	MergeInProgress bool   `json:"merge_in_progress"`
}

func (r *Repo) Info() (RepoInfo, error) {
	info := RepoInfo{
		Path:            r.workDir,
		HashAlgo:        string(r.algo),
		MergeInProgress: r.MergeInProgress(),
	}
	br, err := r.CurrentBranch()
	if err != nil {
		return info, err
	}
	info.CurrentBranch = br
	head, err := r.ResolveHEAD()
	if err == nil {
		info.Head = head
	} else if !isUnborn(err) {
		return info, err
	}
	n, err := r.store.Count()
	if err != nil {
		return info, err
	}
	info.ObjectCount = n
	return info, nil
}

func (r *Repo) Add(paths []string) ([]IndexEntry, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: paths is required", ErrValidation)
	}
	idx, err := r.loadIndex()
	if err != nil {
		return nil, err
	}
	var added []IndexEntry
	for _, p := range paths {
		rel, err := normalizeRel(p)
		if err != nil {
			return nil, err
		}
		entries, err := r.addPath(idx, rel)
		if err != nil {
			return nil, err
		}
		added = append(added, entries...)
	}
	if err := SaveIndex(r.indexPath(), idx); err != nil {
		return nil, err
	}
	return added, nil
}

func (r *Repo) addPath(idx *Index, rel string) ([]IndexEntry, error) {
	abs, err := r.abs(rel)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, rel)
		}
		return nil, err
	}
	if st.IsDir() {
		var out []IndexEntry
		err := filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skipName(d.Name()) && p != abs {
					return filepath.SkipDir
				}
				return nil
			}
			if skipName(d.Name()) {
				return nil
			}
			rp, err := filepath.Rel(r.workDir, p)
			if err != nil {
				return err
			}
			slash := filepath.ToSlash(rp)
			if r.IsIgnored(slash) {
				return nil
			}
			e, err := r.hashWorktreeFile(slash)
			if err != nil {
				return err
			}
			idx.Upsert(e)
			out = append(out, e)
			return nil
		})
		return out, err
	}
	e, err := r.hashWorktreeFile(rel)
	if err != nil {
		return nil, err
	}
	idx.Upsert(e)
	return []IndexEntry{e}, nil
}

func (r *Repo) hashWorktreeFile(rel string) (IndexEntry, error) {
	abs, err := r.abs(rel)
	if err != nil {
		return IndexEntry{}, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return IndexEntry{}, err
	}
	if int64(len(data)) > maxFileBytes {
		return IndexEntry{}, fmt.Errorf("%w: file exceeds 2MB", ErrValidation)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return IndexEntry{}, err
	}
	oid, err := r.store.Write(TypeBlob, data)
	if err != nil {
		return IndexEntry{}, err
	}
	return IndexEntry{
		Path:  rel,
		Mode:  fileMode(st),
		OID:   oid,
		Size:  st.Size(),
		Mtime: st.ModTime().Unix(),
	}, nil
}

func (r *Repo) Commit(message, author string) (Commit, error) {
	message = strings.TrimSpace(message)
	author = strings.TrimSpace(author)
	if message == "" {
		return Commit{}, fmt.Errorf("%w: message is required", ErrValidation)
	}
	if author == "" {
		author = "GoGit <gogit@local>"
	}
	idx, err := r.loadIndex()
	if err != nil {
		return Commit{}, err
	}
	if len(idx.Entries) == 0 {
		return Commit{}, fmt.Errorf("%w: nothing to commit (empty index)", ErrValidation)
	}
	treeOID, err := r.writeTreeFromIndex(idx)
	if err != nil {
		return Commit{}, err
	}
	var parents []string
	head, err := r.ResolveHEAD()
	if err == nil {
		parents = append(parents, head)
	} else if !isUnborn(err) {
		return Commit{}, err
	}
	if mh, err := r.ReadMergeHEAD(); err == nil {
		parents = append(parents, mh)
	} else if r.MergeInProgress() && err != ErrNotFound {
		return Commit{}, err
	}
	payload, err := EncodeCommit(Commit{
		Tree:    treeOID,
		Parents: parents,
		Author:  author,
		Message: message,
		Unix:    NowBeijing().Unix(),
	}, r.algo)
	if err != nil {
		return Commit{}, err
	}
	oid, err := r.store.Write(TypeCommit, payload)
	if err != nil {
		return Commit{}, err
	}
	br, err := r.CurrentBranch()
	if err != nil {
		return Commit{}, err
	}
	if err := r.WriteBranch(br, oid); err != nil {
		return Commit{}, err
	}
	r.clearMergeState()
	c, err := r.ReadCommit(oid)
	if err != nil {
		return Commit{}, err
	}
	if r.log != nil {
		r.log.Info("commit created", "hash", oid, "branch", br)
	}
	return c, nil
}

func isUnborn(err error) bool {
	return err != nil && (err == ErrUnbornHEAD || strings.Contains(err.Error(), "unborn"))
}

func (r *Repo) writeTreeFromIndex(idx *Index) (string, error) {
	type node struct {
		files map[string]IndexEntry
		dirs  map[string]*node
	}
	root := &node{files: map[string]IndexEntry{}, dirs: map[string]*node{}}
	ensure := func(n *node, name string) *node {
		if n.dirs[name] == nil {
			n.dirs[name] = &node{files: map[string]IndexEntry{}, dirs: map[string]*node{}}
		}
		return n.dirs[name]
	}
	for _, e := range idx.Entries {
		parts := strings.Split(e.Path, "/")
		cur := root
		for i, part := range parts {
			if i == len(parts)-1 {
				cur.files[part] = e
			} else {
				cur = ensure(cur, part)
			}
		}
	}
	var write func(n *node) (string, error)
	write = func(n *node) (string, error) {
		var entries []TreeEntry
		for name, e := range n.files {
			entries = append(entries, TreeEntry{Mode: e.Mode, Name: name, OID: e.OID, Type: TypeBlob})
		}
		for name, child := range n.dirs {
			oid, err := write(child)
			if err != nil {
				return "", err
			}
			entries = append(entries, TreeEntry{Mode: "40000", Name: name, OID: oid, Type: TypeTree})
		}
		content, err := EncodeTree(entries, r.algo)
		if err != nil {
			return "", err
		}
		return r.store.Write(TypeTree, content)
	}
	return write(root)
}

func (r *Repo) ReadCommit(oid string) (Commit, error) {
	typ, content, err := r.store.Read(oid)
	if err != nil {
		return Commit{}, err
	}
	if typ != TypeCommit {
		return Commit{}, fmt.Errorf("%w: object is %s not commit", ErrValidation, typ)
	}
	c, err := DecodeCommit(content, r.algo)
	if err != nil {
		return Commit{}, err
	}
	c.Hash = oid
	return c, nil
}

func (r *Repo) ReadBlob(oid string) ([]byte, error) {
	typ, content, err := r.store.Read(oid)
	if err != nil {
		return nil, err
	}
	if typ != TypeBlob {
		return nil, fmt.Errorf("%w: object is %s not blob", ErrValidation, typ)
	}
	return content, nil
}

func (r *Repo) ReadTree(oid string) ([]TreeEntry, error) {
	typ, content, err := r.store.Read(oid)
	if err != nil {
		return nil, err
	}
	if typ != TypeTree {
		return nil, fmt.Errorf("%w: object is %s not tree", ErrValidation, typ)
	}
	return DecodeTree(content, r.algo)
}

type FlatEntry struct {
	Path string     `json:"path"`
	Mode string     `json:"mode"`
	OID  string     `json:"oid"`
	Type ObjectType `json:"type"`
}

func (r *Repo) FlattenTree(treeOID, prefix string) ([]FlatEntry, error) {
	entries, err := r.ReadTree(treeOID)
	if err != nil {
		return nil, err
	}
	var out []FlatEntry
	for _, e := range entries {
		p := e.Name
		if prefix != "" {
			p = path.Join(prefix, e.Name)
		}
		if e.IsTree() {
			kids, err := r.FlattenTree(e.OID, p)
			if err != nil {
				return nil, err
			}
			out = append(out, kids...)
		} else {
			out = append(out, FlatEntry{Path: p, Mode: e.Mode, OID: e.OID, Type: TypeBlob})
		}
	}
	return out, nil
}

func (r *Repo) FlattenMap(treeOID string) (map[string]FlatEntry, error) {
	list, err := r.FlattenTree(treeOID, "")
	if err != nil {
		return nil, err
	}
	m := make(map[string]FlatEntry, len(list))
	for _, e := range list {
		m[e.Path] = e
	}
	return m, nil
}

func (r *Repo) Log(branch string, limit int) ([]Commit, error) {
	if branch == "" {
		var err error
		branch, err = r.CurrentBranch()
		if err != nil {
			return nil, err
		}
	}
	oid, err := r.ResolveBranch(branch)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	seen := map[string]bool{}
	var out []Commit
	queue := []string{oid}
	for len(queue) > 0 && len(out) < limit-1 {
		cur := queue[0]
		queue = queue[1:]
		if seen[cur] {
			continue
		}
		seen[cur] = true
		c, err := r.ReadCommit(cur)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
		queue = append(queue, c.Parents...)
	}
	return out, nil
}

func (r *Repo) Inspect(oid string) (map[string]any, error) {
	if !r.algo.ValidHex(oid) {
		return nil, fmt.Errorf("%w: invalid object id", ErrValidation)
	}
	typ, content, err := r.store.Read(oid)
	if err != nil {
		return nil, err
	}
	switch typ {
	case TypeBlob:
		return map[string]any{
			"type":    typ,
			"hash":    oid,
			"size":    len(content),
			"binary":  IsBinary(content),
			"content": blobPreview(content),
		}, nil
	case TypeTree:
		entries, err := DecodeTree(content, r.algo)
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": typ, "hash": oid, "entries": entries}, nil
	case TypeCommit:
		c, err := DecodeCommit(content, r.algo)
		if err != nil {
			return nil, err
		}
		c.Hash = oid
		return map[string]any{
			"type":         typ,
			"hash":         oid,
			"tree":         c.Tree,
			"parents":      c.Parents,
			"author":       c.Author,
			"message":      c.Message,
			"committed_at": c.CommittedAt,
		}, nil
	default:
		return nil, fmt.Errorf("%w: unknown type", ErrValidation)
	}
}

func blobPreview(content []byte) string {
	if IsBinary(content) {
		return ""
	}
	s := string(content)
	if len(s) > 64*1024 {
		return s[:64*1024]
	}
	return s
}

func (r *Repo) CreateBranch(name string) (BranchInfo, error) {
	if err := validateBranchName(name); err != nil {
		return BranchInfo{}, err
	}
	if _, err := os.Stat(r.branchPath(name)); err == nil {
		return BranchInfo{}, fmt.Errorf("%w: branch %s", ErrAlreadyExists, name)
	}
	head, err := r.ResolveHEAD()
	if err != nil {
		return BranchInfo{}, err
	}
	if err := r.WriteBranch(name, head); err != nil {
		return BranchInfo{}, err
	}
	return BranchInfo{Name: name, Hash: head, Current: false}, nil
}

func (r *Repo) Checkout(name string) error {
	if r.MergeInProgress() {
		return ErrMergeInProgress
	}
	target, err := r.ResolveBranch(name)
	if err != nil {
		return err
	}
	var currentTree map[string]FlatEntry
	head, err := r.ResolveHEAD()
	if err == nil {
		hc, err := r.ReadCommit(head)
		if err != nil {
			return err
		}
		currentTree, err = r.FlattenMap(hc.Tree)
		if err != nil {
			return err
		}
	} else if !isUnborn(err) {
		return err
	} else {
		currentTree = map[string]FlatEntry{}
	}
	tc, err := r.ReadCommit(target)
	if err != nil {
		return err
	}
	targetTree, err := r.FlattenMap(tc.Tree)
	if err != nil {
		return err
	}
	idx, err := r.loadIndex()
	if err != nil {
		return err
	}
	im := idx.Map()
	// refuse if a tracked file differs from HEAD and would be overwritten
	for p, te := range targetTree {
		abs, err := r.abs(p)
		if err != nil {
			return err
		}
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		wt, err := r.hashWorktreeFile(p)
		if err != nil {
			return err
		}
		headOID := ""
		if ce, ok := currentTree[p]; ok {
			headOID = ce.OID
		}
		indexOID := ""
		if ie, ok := im[p]; ok {
			indexOID = ie.OID
		}
		dirty := (indexOID != "" && indexOID != headOID) || wt.OID != headOID && headOID != "" || (headOID == "" && (indexOID != "" || fileExists(abs)))
		if dirty && te.OID != wt.OID {
			// allow if worktree already matches target
			if wt.OID != te.OID {
				return fmt.Errorf("%w: %s", ErrDirtyWorktree, p)
			}
		}
	}
	// restore target snapshot
	if err := r.checkoutTree(currentTree, targetTree); err != nil {
		return err
	}
	newIdx := &Index{}
	for _, e := range targetTree {
		blob, err := r.ReadBlob(e.OID)
		if err != nil {
			return err
		}
		newIdx.Upsert(IndexEntry{Path: e.Path, Mode: e.Mode, OID: e.OID, Size: int64(len(blob))})
	}
	if err := SaveIndex(r.indexPath(), newIdx); err != nil {
		return err
	}
	return r.WriteHEAD(name)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (r *Repo) checkoutTree(oldT, newT map[string]FlatEntry) error {
	for p := range oldT {
		if _, keep := newT[p]; keep {
			continue
		}
		abs, err := r.abs(p)
		if err != nil {
			return err
		}
		_ = os.Remove(abs)
		removeEmptyParents(r.workDir, abs)
	}
	for p, e := range newT {
		blob, err := r.ReadBlob(e.OID)
		if err != nil {
			return err
		}
		if err := r.writeWorktreeFile(p, blob, e.Mode); err != nil {
			return err
		}
	}
	return nil
}

func removeEmptyParents(root, file string) {
	dir := filepath.Dir(file)
	for dir != root && strings.HasPrefix(dir, root) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

func (r *Repo) writeWorktreeFile(rel string, data []byte, mode string) error {
	abs, err := r.abs(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if mode == "100755" {
		perm = 0o755
	}
	return os.WriteFile(abs, data, perm)
}

func (r *Repo) WriteWorktree(rel, content string) error {
	rel, err := normalizeRel(rel)
	if err != nil {
		return err
	}
	if rel == "" {
		return fmt.Errorf("%w: path is required", ErrInvalidPath)
	}
	if int64(len(content)) > maxFileBytes {
		return fmt.Errorf("%w: file exceeds 2MB", ErrValidation)
	}
	return r.writeWorktreeFile(rel, []byte(content), "100644")
}

func (r *Repo) DeleteWorktree(rel string) error {
	rel, err := normalizeRel(rel)
	if err != nil {
		return err
	}
	abs, err := r.abs(rel)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, rel)
		}
		return err
	}
	idx, err := r.loadIndex()
	if err != nil {
		return err
	}
	idx.Remove(rel)
	return SaveIndex(r.indexPath(), idx)
}

func (r *Repo) ReadWorktree(rel string) ([]byte, error) {
	rel, err := normalizeRel(rel)
	if err != nil {
		return nil, err
	}
	abs, err := r.abs(rel)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, rel)
		}
		return nil, err
	}
	return b, nil
}

type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	Mode string `json:"mode,omitempty"`
}

func (r *Repo) ListWorktree(rel string) ([]DirEntry, error) {
	rel, err := normalizeRel(rel)
	if err != nil {
		return nil, err
	}
	abs, err := r.abs(rel)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, rel)
		}
		return nil, err
	}
	var out []DirEntry
	for _, e := range ents {
		if skipName(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		p := e.Name()
		if rel != "" {
			p = path.Join(rel, e.Name())
		}
		typ := "file"
		mode := fileMode(info)
		if e.IsDir() {
			typ = "dir"
			mode = "40000"
		}
		out = append(out, DirEntry{Name: e.Name(), Path: p, Type: typ, Size: info.Size(), Mode: mode})
	}
	return out, nil
}

func (r *Repo) ancestorSet(oid string) (map[string]bool, error) {
	seen := map[string]bool{}
	queue := []string{oid}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == "" || seen[cur] {
			continue
		}
		seen[cur] = true
		c, err := r.ReadCommit(cur)
		if err != nil {
			return nil, err
		}
		queue = append(queue, c.Parents...)
	}
	return seen, nil
}

func (r *Repo) mergeBase(ours, theirs string) (string, error) {
	a, err := r.ancestorSet(ours)
	if err != nil {
		return "", err
	}
	queue := []string{theirs}
	seen := map[string]bool{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if seen[cur] {
			continue
		}
		seen[cur] = true
		if a[cur] {
			return cur, nil
		}
		c, err := r.ReadCommit(cur)
		if err != nil {
			return "", err
		}
		queue = append(queue, c.Parents...)
	}
	return "", nil
}
