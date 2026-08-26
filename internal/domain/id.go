package domain

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomToken() string { b := make([]byte, 8); _, _ = rand.Read(b); return hex.EncodeToString(b) }
