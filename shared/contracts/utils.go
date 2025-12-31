package contracts

import (
	"crypto/rand"
	"encoding/hex"
)

// generateEventID generates a unique event ID
func generateEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

