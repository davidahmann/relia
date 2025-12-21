package crypto

import "crypto/ed25519"

// KeyPairFromSeed derives an Ed25519 keypair from a 32-byte seed.
func KeyPairFromSeed(seed []byte) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, nil, ErrInvalidSeedSize
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return privateKey, publicKey, nil
}
