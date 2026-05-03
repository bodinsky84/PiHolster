package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024 // 64 MB in KiB
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLen     = 16
	argonKeyLen      = 32
)

var (
	errInvalidHashFormat = errors.New("auth: invalid hash format")
	errInvalidHashParams = errors.New("auth: invalid hash parameters")
)

// HashPassword produces an Argon2id hash string in PHC format.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		argonIterations,
		argonMemory,
		argonParallelism,
		argonKeyLen,
	)

	enc := base64.RawStdEncoding
	hash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		enc.EncodeToString(salt),
		enc.EncodeToString(key),
	)
	return hash, nil
}

// VerifyPassword checks password against a PHC-format Argon2id hash.
// Returns false (not an error) when the password is wrong.
func VerifyPassword(password, hash string) (bool, error) {
	salt, key, params, err := parseHash(hash)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		params.keyLen,
	)

	match := subtle.ConstantTimeCompare(candidate, key) == 1
	return match, nil
}

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	keyLen      uint32
}

func parseHash(hash string) (salt, key []byte, p argonParams, err error) {
	// expected: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<key>
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return nil, nil, p, errInvalidHashFormat
	}

	var version int
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil {
		return nil, nil, p, errInvalidHashFormat
	}
	if version != argon2.Version {
		return nil, nil, p, fmt.Errorf("auth: unsupported argon2 version %d", version)
	}

	var mem, iters uint32
	var par uint8
	if _, scanErr := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iters, &par); scanErr != nil {
		return nil, nil, p, errInvalidHashParams
	}
	if mem == 0 || iters == 0 || par == 0 {
		return nil, nil, p, errInvalidHashParams
	}

	enc := base64.RawStdEncoding
	salt, err = enc.DecodeString(parts[4])
	if err != nil {
		return nil, nil, p, fmt.Errorf("auth: decode salt: %w", err)
	}
	key, err = enc.DecodeString(parts[5])
	if err != nil {
		return nil, nil, p, fmt.Errorf("auth: decode key: %w", err)
	}

	p = argonParams{
		memory:      mem,
		iterations:  iters,
		parallelism: par,
		keyLen:      uint32(len(key)),
	}
	return salt, key, p, nil
}
