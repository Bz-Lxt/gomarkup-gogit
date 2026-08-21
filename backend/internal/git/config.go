package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	HashAlgo  Algo   `json:"hash_algo"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

func DefaultConfig(algo Algo) Config {
	if algo == "" {
		algo = SHA1
	}
	return Config{HashAlgo: algo, UserName: "GoGit", UserEmail: "gogit@local"}
}

func configPath(gitDir string) string {
	return filepath.Join(gitDir, "config")
}

func LoadConfig(gitDir string) (Config, error) {
	cfg := DefaultConfig(SHA1)
	f, err := os.Open(configPath(gitDir))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("%w: missing config", ErrValidation)
		}
		return cfg, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return cfg, fmt.Errorf("%w: malformed config line %q", ErrValidation, line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "hash_algo":
			algo, err := ParseAlgo(val)
			if err != nil {
				return cfg, err
			}
			cfg.HashAlgo = algo
		case "user.name":
			if val != "" {
				cfg.UserName = val
			}
		case "user.email":
			if val != "" {
				cfg.UserEmail = val
			}
		}
	}
	if err := sc.Err(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) Save(gitDir string) error {
	if c.HashAlgo == "" {
		c.HashAlgo = SHA1
	}
	if _, err := ParseAlgo(string(c.HashAlgo)); err != nil {
		return err
	}
	if strings.ContainsAny(c.UserName, "\n=") || strings.ContainsAny(c.UserEmail, "\n=") {
		return fmt.Errorf("%w: config values cannot contain newline or '='", ErrValidation)
	}
	body := fmt.Sprintf("hash_algo=%s\nuser.name=%s\nuser.email=%s\n", c.HashAlgo, c.UserName, c.UserEmail)
	tmp := configPath(gitDir) + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, configPath(gitDir))
}

func (c Config) Author() string {
	return FormatAuthor(c.UserName, c.UserEmail)
}

func (r *Repo) Config() (Config, error) {
	return LoadConfig(r.gitDir)
}

func (r *Repo) SetConfig(cfg Config) error {
	if cfg.HashAlgo == "" {
		cfg.HashAlgo = r.algo
	}
	if cfg.HashAlgo != r.algo {
		return fmt.Errorf("%w: hash_algo cannot change after init", ErrValidation)
	}
	return cfg.Save(r.gitDir)
}
