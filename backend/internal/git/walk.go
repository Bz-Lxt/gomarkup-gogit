package git

import (
	"os"
	"path/filepath"
)

func walkWorktree(root string, fn func(rel string)) error {
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipName(d.Name()) && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if skipName(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		fn(filepath.ToSlash(rel))
		return nil
	})
}
