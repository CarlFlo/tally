package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      uint32 = 16 * 1024
	argonIterations  uint32 = 3
	argonParallelism uint8  = 1
	argonKeyLength   uint32 = 32

	currentArgonParams = "m=16384,t=3,p=1"
	legacyArgonParams  = "m=65536,t=2,p=2"
)

type argonParameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return "", e
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return "$argon2id$v=19$" + currentArgonParams + "$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

func allowedArgonParameters(value string) (argonParameters, bool) {
	switch value {
	case currentArgonParams:
		return argonParameters{memory: argonMemory, iterations: argonIterations, parallelism: argonParallelism}, true
	case legacyArgonParams:
		return argonParameters{memory: 64 * 1024, iterations: 2, parallelism: 2}, true
	default:
		return argonParameters{}, false
	}
}

func Verify(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	params, ok := allowedArgonParameters(parts[3])
	if !ok {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil || len(salt) != 16 {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[5])
	if e != nil || len(expected) != int(argonKeyLength) {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, argonKeyLength)
	return subtle.ConstantTimeCompare(expected, actual) == 1
}

func NeedsRehash(encoded string) bool {
	parts := strings.Split(encoded, "$")
	return len(parts) == 6 && parts[1] == "argon2id" && parts[2] == "v=19" && parts[3] != currentArgonParams
}
