package normalize

import (
	"crypto/sha256"
	"encoding/hex"
)

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
