package crypto

import "errors"

var (
	ErrFloatNotAllowed  = errors.New("float values are not allowed")
	ErrNonStringMapKey  = errors.New("map keys must be strings")
	ErrUnsupportedType  = errors.New("unsupported type for canonicalization")
	ErrKeyCollision     = errors.New("normalized map key collision")
	ErrInvalidSeedSize  = errors.New("invalid ed25519 seed size")
	ErrInvalidDigestLen = errors.New("invalid digest length")
)
