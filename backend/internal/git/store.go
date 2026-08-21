package git

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	dir  string
	algo Algo
}

func NewStore(gitDir string, algo Algo) *Store {
	return &Store{dir: filepath.Join(gitDir, "objects"), algo: algo}
}

func (s *Store) Write(typ ObjectType, content []byte) (string, error) {
	raw := EncodeObject(typ, content)
	oid := s.algo.Sum(raw)
	path, err := s.pathFor(oid)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return oid, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("mkdir objects: %w", err)
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return "", fmt.Errorf("zlib compress: %w", err)
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("zlib close: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write object: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rename object: %w", err)
	}
	return oid, nil
}

func (s *Store) Read(oid string) (ObjectType, []byte, error) {
	path, err := s.pathFor(oid)
	if err != nil {
		return "", nil, err
	}
	compressed, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("%w: object %s", ErrNotFound, oid)
		}
		return "", nil, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", nil, fmt.Errorf("%w: zlib header: %v", ErrValidation, err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return "", nil, fmt.Errorf("%w: zlib body: %v", ErrValidation, err)
	}
	return DecodeObject(raw)
}

func (s *Store) ReadRawCompressed(oid string) ([]byte, error) {
	path, err := s.pathFor(oid)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: object %s", ErrNotFound, oid)
		}
		return nil, err
	}
	return b, nil
}

func (s *Store) List() ([]string, error) {
	var out []string
	err := filepath.WalkDir(s.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".tmp") {
			return nil
		}
		fanout := filepath.Base(filepath.Dir(path))
		if len(fanout) != 2 {
			return nil
		}
		oid := fanout + name
		if s.algo.ValidHex(oid) {
			out = append(out, oid)
		}
		return nil
	})
	return out, err
}

func (s *Store) ListPrefix(prefix string) ([]string, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, oid := range all {
		if strings.HasPrefix(oid, prefix) {
			out = append(out, oid)
		}
	}
	return out, nil
}

func (s *Store) Count() (int, error) {
	all, err := s.List()
	return len(all), err
}

func (s *Store) pathFor(oid string) (string, error) {
	if !s.algo.ValidHex(oid) {
		return "", fmt.Errorf("%w: invalid object id", ErrValidation)
	}
	return filepath.Join(s.dir, oid[:2], oid[2:]), nil
}

func IsZlib(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	// zlib CMF/FLG: CMF typically 0x78, checksum (cmf*256+flg) % 31 == 0
	cmf, flg := int(data[0]), int(data[1])
	return data[0] == 0x78 && (cmf*256+flg)%31 == 0
}
