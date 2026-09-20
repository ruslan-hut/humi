package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// New returns a 256-bit random node token, hex encoded.
func New() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Hash returns the SHA-256 of a token. Tokens carry full entropy, so a plain
// digest is enough and stays cheap on every ingest request.
func Hash(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}
