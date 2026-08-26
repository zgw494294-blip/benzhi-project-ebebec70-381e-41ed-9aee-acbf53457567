package service

import (
	"crypto/sha256"
	"encoding/hex"
)

func digestText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
