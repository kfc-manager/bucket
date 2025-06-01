package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

type Error struct {
	msg    string
	Status int
}

func (e *Error) Error() string {
	return e.msg
}

func Sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
