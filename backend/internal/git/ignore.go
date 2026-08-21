package git

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type IgnoreSet struct {
	rules []ignoreRule
}

type ignoreRule struct {
	raw     string
	neg     bool
	dirOnly bool
	pattern string
}

func LoadIgnore(workDir string) (*IgnoreSet, error) {
	ig := &IgnoreSet{}
	p := filepath.Join(workDir, ".gogitignore")
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return ig, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule := ignoreRule{raw: line}
		if strings.HasPrefix(line, "!") {
			rule.neg = true
			line = strings.TrimSpace(line[1:])
		}
		if strings.HasSuffix(line, "/") {
			rule.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		rule.pattern = strings.ReplaceAll(line, "\\", "/")
		if rule.pattern != "" {
			ig.rules = append(ig.rules, rule)
		}
	}
	return ig, sc.Err()
}

func (ig *IgnoreSet) Match(rel string, isDir bool) bool {
	if ig == nil || len(ig.rules) == 0 {
		return false
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "./")
	ignored := false
	for _, rule := range ig.rules {
		if rule.dirOnly && !isDir {
			// still match if any path prefix is that directory name
			if !pathHasDir(rel, rule.pattern) && !matchIgnore(rel, rule.pattern) {
				continue
			}
		} else if !matchIgnore(rel, rule.pattern) {
			continue
		}
		ignored = !rule.neg
	}
	return ignored
}

func pathHasDir(rel, dir string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if matchSeg(seg, path.Base(dir)) && !strings.Contains(dir, "/") {
			return true
		}
		if strings.HasPrefix(rel, dir+"/") || rel == dir {
			return true
		}
	}
	return false
}

func matchIgnore(rel, pattern string) bool {
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "/") {
		base := path.Base(rel)
		if matchSeg(base, pattern) {
			return true
		}
		for _, seg := range strings.Split(rel, "/") {
			if matchSeg(seg, pattern) {
				return true
			}
		}
		return false
	}
	pattern = strings.TrimPrefix(pattern, "/")
	return matchPath(rel, pattern)
}

func matchPath(rel, pattern string) bool {
	if matchSeg(rel, pattern) {
		return true
	}
	r := strings.Split(rel, "/")
	p := strings.Split(pattern, "/")
	return matchSegs(r, p)
}

func matchSegs(rel, pat []string) bool {
	if len(pat) == 0 {
		return len(rel) == 0
	}
	if pat[0] == "**" {
		if matchSegs(rel, pat[1:]) {
			return true
		}
		if len(rel) == 0 {
			return false
		}
		return matchSegs(rel[1:], pat)
	}
	if len(rel) == 0 {
		return false
	}
	if !matchSeg(rel[0], pat[0]) {
		return false
	}
	return matchSegs(rel[1:], pat[1:])
}

func matchSeg(s, pat string) bool {
	if pat == "*" || pat == s {
		return true
	}
	// simple glob: prefix*, *suffix, *mid*
	if strings.Count(pat, "*") == 0 {
		return s == pat
	}
	parts := strings.Split(pat, "*")
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	rest := s[len(parts[0]):]
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if p == "" {
			if i == len(parts)-1 {
				return true
			}
			continue
		}
		idx := strings.Index(rest, p)
		if idx < 0 {
			return false
		}
		rest = rest[idx+len(p):]
	}
	return true
}

func (r *Repo) loadIgnore() *IgnoreSet {
	ig, err := LoadIgnore(r.workDir)
	if err != nil || ig == nil {
		return &IgnoreSet{}
	}
	return ig
}

func (r *Repo) IsIgnored(rel string) bool {
	return r.loadIgnore().Match(rel, false)
}
