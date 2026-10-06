package keyvault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func aead(master string) (cipher.AEAD, error) {
	key, e := hex.DecodeString(master)
	if e != nil || len(key) != 32 {
		return nil, errors.New("AI encryption key must be 32 hex-encoded bytes")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}
func Seal(master, secret string) ([]byte, error) {
	a, e := aead(master)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, a.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return nil, e
	}
	return a.Seal(nonce, nonce, []byte(secret), []byte("mockai:gemini:v1")), nil
}
func Open(master string, data []byte) (string, error) {
	a, e := aead(master)
	if e != nil {
		return "", e
	}
	if len(data) < a.NonceSize() {
		return "", errors.New("invalid encrypted key")
	}
	v, e := a.Open(nil, data[:a.NonceSize()], data[a.NonceSize():], []byte("mockai:gemini:v1"))
	return string(v), e
}
func Fingerprint(secret string) string {
	v := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(v[:])
}
