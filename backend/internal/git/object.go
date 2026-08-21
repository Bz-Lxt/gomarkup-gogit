package git

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type ObjectType string

const (
	TypeBlob   ObjectType = "blob"
	TypeTree   ObjectType = "tree"
	TypeCommit ObjectType = "commit"
)

func EncodeObject(typ ObjectType, content []byte) []byte {
	header := fmt.Sprintf("%s %d\x00", typ, len(content))
	out := make([]byte, 0, len(header)+len(content))
	out = append(out, header...)
	out = append(out, content...)
	return out
}

func DecodeObject(raw []byte) (ObjectType, []byte, error) {
	if len(raw) == 0 {
		return "", nil, fmt.Errorf("%w: empty object", ErrValidation)
	}
	nul := bytes.IndexByte(raw, 0)
	if nul <= 0 {
		return "", nil, fmt.Errorf("%w: object header missing NUL", ErrValidation)
	}
	header := string(raw[:nul])
	sp := strings.IndexByte(header, ' ')
	if sp <= 0 {
		return "", nil, fmt.Errorf("%w: object header missing type/size", ErrValidation)
	}
	typ := ObjectType(header[:sp])
	if typ != TypeBlob && typ != TypeTree && typ != TypeCommit {
		return "", nil, fmt.Errorf("%w: unknown object type %q", ErrValidation, typ)
	}
	size, err := strconv.Atoi(header[sp+1:])
	if err != nil || size < 0 {
		return "", nil, fmt.Errorf("%w: invalid object size", ErrValidation)
	}
	content := raw[nul+1:]
	if len(content) != size {
		return "", nil, fmt.Errorf("%w: object size mismatch: header=%d body=%d", ErrValidation, size, len(content))
	}
	return typ, content, nil
}

type TreeEntry struct {
	Mode string
	Name string
	OID  string
	Type ObjectType
}

func (e TreeEntry) IsTree() bool {
	return e.Mode == "40000" || e.Mode == "040000" || e.Type == TypeTree
}

func EncodeTree(entries []TreeEntry, algo Algo) ([]byte, error) {
	// Build an independent, normalized copy so the caller's slice (which may be
	// cached for later audit/retry) is never mutated.
	cp := make([]TreeEntry, len(entries))
	for i := range entries {
		e := entries[i]
		if strings.TrimSpace(e.Name) == "" || strings.ContainsAny(e.Name, "/\x00") {
			return nil, fmt.Errorf("%w: invalid tree entry name", ErrValidation)
		}
		if e.IsTree() {
			e.Mode = "40000"
			e.Type = TypeTree
		} else if e.Mode == "" {
			e.Mode = "100644"
			e.Type = TypeBlob
		}
		if _, err := algo.DecodeOID(e.OID); err != nil {
			return nil, err
		}
		cp[i] = e
	}
	sortTree(cp)
	var buf bytes.Buffer
	for _, e := range cp {
		raw, err := algo.DecodeOID(e.OID)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, "%s %s\x00", e.Mode, e.Name)
		buf.Write(raw)
	}
	return buf.Bytes(), nil
}

func DecodeTree(content []byte, algo Algo) ([]TreeEntry, error) {
	var entries []TreeEntry
	rest := content
	oidSize := algo.Size()
	for len(rest) > 0 {
		sp := bytes.IndexByte(rest, ' ')
		if sp <= 0 {
			return nil, fmt.Errorf("%w: malformed tree entry (mode)", ErrValidation)
		}
		mode := string(rest[:sp])
		rest = rest[sp+1:]
		nul := bytes.IndexByte(rest, 0)
		if nul <= 0 {
			return nil, fmt.Errorf("%w: malformed tree entry (name)", ErrValidation)
		}
		name := string(rest[:nul])
		rest = rest[nul+1:]
		if len(rest) < oidSize {
			return nil, fmt.Errorf("%w: malformed tree entry (oid truncated)", ErrValidation)
		}
		oid := fmt.Sprintf("%x", rest[:oidSize])
		rest = rest[oidSize:]
		typ := TypeBlob
		if mode == "40000" || mode == "040000" {
			typ = TypeTree
		}
		entries = append(entries, TreeEntry{Mode: mode, Name: name, OID: oid, Type: typ})
	}
	return entries, nil
}

func sortTree(entries []TreeEntry) {
	// Git sorts directory names as if they had a trailing slash.
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if treeSortKey(entries[j]) < treeSortKey(entries[i]) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

func treeSortKey(e TreeEntry) string {
	if e.IsTree() {
		return e.Name + "/"
	}
	return e.Name
}

type Commit struct {
	Hash        string   `json:"hash"`
	Tree        string   `json:"tree"`
	Parents     []string `json:"parents"`
	Author      string   `json:"author"`
	Message     string   `json:"message"`
	Unix        int64    `json:"-"`
	CommittedAt string   `json:"committed_at"`
}

func EncodeCommit(c Commit, algo Algo) ([]byte, error) {
	if !algo.ValidHex(c.Tree) {
		return nil, fmt.Errorf("%w: commit tree oid invalid", ErrValidation)
	}
	if strings.TrimSpace(c.Author) == "" {
		return nil, fmt.Errorf("%w: author is required", ErrValidation)
	}
	if strings.TrimSpace(c.Message) == "" {
		return nil, fmt.Errorf("%w: message is required", ErrValidation)
	}
	for _, p := range c.Parents {
		if !algo.ValidHex(p) {
			return nil, fmt.Errorf("%w: parent oid invalid", ErrValidation)
		}
	}
	when := c.Unix
	if when == 0 {
		when = NowBeijing().Unix()
	}
	ident := formatIdent(c.Author, when)
	var b strings.Builder
	fmt.Fprintf(&b, "tree %s\n", c.Tree)
	for _, p := range c.Parents {
		fmt.Fprintf(&b, "parent %s\n", p)
	}
	fmt.Fprintf(&b, "author %s\n", ident)
	fmt.Fprintf(&b, "committer %s\n", ident)
	b.WriteByte('\n')
	b.WriteString(strings.TrimRight(c.Message, "\n"))
	b.WriteByte('\n')
	return []byte(b.String()), nil
}

func DecodeCommit(content []byte, algo Algo) (Commit, error) {
	text := string(content)
	head, body, ok := strings.Cut(text, "\n\n")
	if !ok {
		return Commit{}, fmt.Errorf("%w: commit missing message separator", ErrValidation)
	}
	c := Commit{Message: strings.TrimRight(body, "\n"), Parents: []string{}}
	for _, line := range strings.Split(head, "\n") {
		key, val, found := strings.Cut(line, " ")
		if !found {
			return Commit{}, fmt.Errorf("%w: malformed commit header %q", ErrValidation, line)
		}
		switch key {
		case "tree":
			c.Tree = val
		case "parent":
			c.Parents = append(c.Parents, val)
		case "author":
			c.Author, c.Unix = parseIdent(val)
			c.CommittedAt = formatBeijing(c.Unix)
		case "committer":
			// author already captured; committer used for timestamp fallback
			if c.Unix == 0 {
				_, c.Unix = parseIdent(val)
				c.CommittedAt = formatBeijing(c.Unix)
			}
		}
	}
	if !algo.ValidHex(c.Tree) {
		return Commit{}, fmt.Errorf("%w: commit tree missing", ErrValidation)
	}
	if strings.TrimSpace(c.Message) == "" {
		return Commit{}, fmt.Errorf("%w: commit message empty", ErrValidation)
	}
	return c, nil
}

func IsBinary(data []byte) bool {
	n := len(data)
	if n > 8192 {
		n = 8192
	}
	return bytes.IndexByte(data[:n], 0) >= 0
}
