package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// APITokenPrefix makes tokens greppable and lets the auth middleware route
// them without touching the JWT path. Full format: hkdp_[A-Za-z0-9]{40}.
const APITokenPrefix = "hkdp_"

const (
	apiTokenAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	apiTokenLength   = 40 // ~238 bits of entropy
	// APITokenDisplayLength is how much of the token is stored and shown for
	// identification (prefix + 7 chars).
	APITokenDisplayLength = 12
)

// GenerateAPIToken returns the full secret token (shown to the user exactly
// once), the hex SHA-256 hash to persist, and the display prefix.
func GenerateAPIToken() (token, hash, prefix string, err error) {
	buf := make([]byte, 0, len(APITokenPrefix)+apiTokenLength)
	buf = append(buf, APITokenPrefix...)
	max := big.NewInt(int64(len(apiTokenAlphabet)))
	for i := 0; i < apiTokenLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", "", "", fmt.Errorf("generate token: %w", err)
		}
		buf = append(buf, apiTokenAlphabet[n.Int64()])
	}
	token = string(buf)
	return token, HashAPIToken(token), token[:APITokenDisplayLength], nil
}

// HashAPIToken is the canonical hash used both at creation and lookup.
func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
