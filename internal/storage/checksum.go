package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func SHA256Reader(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
