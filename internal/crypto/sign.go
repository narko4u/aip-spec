package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// KeyPair holds an Ed25519 key pair for signing AIP contracts and evidence receipts.
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateKeyPair creates a new Ed25519 key pair.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("aip/crypto: generate key pair: %w", err)
	}
	return &KeyPair{PrivateKey: priv, PublicKey: pub}, nil
}

// Sign signs a message with the private key and returns a hex-encoded signature.
func (kp *KeyPair) Sign(message []byte) string {
	sig := ed25519.Sign(kp.PrivateKey, message)
	return hex.EncodeToString(sig)
}

// Verify checks a hex-encoded signature against a message and public key.
func Verify(publicKey ed25519.PublicKey, message []byte, signatureHex string) bool {
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, message, sig)
}

// PublicKeyHex returns the hex-encoded public key.
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey)
}
