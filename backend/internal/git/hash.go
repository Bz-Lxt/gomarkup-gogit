package git

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

type Algo string

const (
	SHA1   Algo = "sha1"
	SHA256 Algo = "sha256"
)

func ParseAlgo(s string) (Algo, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "sha1":
		return SHA1, nil
	case "sha256":
		return SHA256, nil
	default:
		return "", fmt.Errorf("%w: hash_algo must be sha1 or sha256", ErrValidation)
	}
}

func (a Algo) Size() int {
	if a == SHA256 {
		return 32
	}
	return 20
}

func (a Algo) HexLen() int {
	return a.Size() * 2
}

func (a Algo) New() hash.Hash {
	if a == SHA256 {
		return sha256.New()
	}
	return sha1.New()
}

func (a Algo) Sum(payload []byte) string {
	h := a.New()
	_, _ = h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func (a Algo) ValidHex(oid string) bool {
	if len(oid) != a.HexLen() {
		return false
	}
	_, err := hex.DecodeString(oid)
	return err == nil
}

func (a Algo) DecodeOID(oid string) ([]byte, error) {
	if !a.ValidHex(oid) {
		return nil, fmt.Errorf("%w: invalid object id %q", ErrValidation, oid)
	}
	return hex.DecodeString(oid)
}
