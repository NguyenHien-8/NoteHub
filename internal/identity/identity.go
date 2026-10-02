package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
)

var uidPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func NewUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func ValidUID(uid string) bool { return uidPattern.MatchString(uid) }

// NewBearerToken returns a high-entropy URL-safe token. Only HashToken(token)
// should be persisted; the clear token is shown to the user once.
func NewBearerToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "nhs_" + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func HashToken(token string) ([]byte, error) {
	if len(token) < 16 {
		return nil, errors.New("invalid share token")
	}
	d := sha256.Sum256([]byte(token))
	return d[:], nil
}
