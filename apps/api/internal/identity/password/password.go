package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory      = 64 * 1024
	iterations  = 3
	parallelism = 2
	saltLength  = 16
	keyLength   = 32
)

var ErrInvalidHash = errors.New("invalid password hash")

func Hash(plainText string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(plainText), salt, iterations, memory, parallelism, keyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func Verify(plainText, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, ErrInvalidHash
	}

	var parsedMemory uint64
	var parsedIterations uint64
	var parsedParallelism uint64
	for _, item := range strings.Split(parts[3], ",") {
		keyValue := strings.SplitN(item, "=", 2)
		if len(keyValue) != 2 {
			return false, ErrInvalidHash
		}
		value, err := strconv.ParseUint(keyValue[1], 10, 32)
		if err != nil {
			return false, ErrInvalidHash
		}
		switch keyValue[0] {
		case "m":
			parsedMemory = value
		case "t":
			parsedIterations = value
		case "p":
			parsedParallelism = value
		default:
			return false, ErrInvalidHash
		}
	}
	if parsedMemory == 0 || parsedIterations == 0 || parsedParallelism == 0 || parsedParallelism > 255 {
		return false, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return false, ErrInvalidHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 {
		return false, ErrInvalidHash
	}
	actual := argon2.IDKey(
		[]byte(plainText),
		salt,
		uint32(parsedIterations),
		uint32(parsedMemory),
		uint8(parsedParallelism),
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
