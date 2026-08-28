package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-jwt/jwt/v5"
)

type KeySet struct {
	KeyID      string
	PrivateKey *rsa.PrivateKey
}

// LoadOrGenerateKey reads an RSA PEM key from path, generating and
// persisting a new 2048-bit key on first run. The directory is created if
// missing so a fresh install works without a separate provisioning step.
func LoadOrGenerateKey(path string) (*KeySet, error) {
	if data, err := os.ReadFile(path); err == nil {
		key, err := parsePEM(data)
		if err != nil {
			return nil, fmt.Errorf("parse signing key %s: %w", path, err)
		}
		return &KeySet{KeyID: keyID(key), PrivateKey: key}, nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create signing key dir: %w", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write signing key: %w", err)
	}

	return &KeySet{KeyID: keyID(key), PrivateKey: key}, nil
}

func parsePEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func keyID(key *rsa.PrivateKey) string {
	sum := key.PublicKey.N.Bytes()
	if len(sum) > 8 {
		sum = sum[:8]
	}
	return base64.RawURLEncoding.EncodeToString(sum)
}

// JWK returns the public key as a JSON Web Key for the /jwks endpoint.
func (k *KeySet) JWK() map[string]interface{} {
	pub := k.PrivateKey.PublicKey
	return map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": k.KeyID,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigIntToBytes(pub.E)),
	}
}

func bigIntToBytes(e int) []byte {
	b := make([]byte, 0, 4)
	for e > 0 {
		b = append([]byte{byte(e & 0xff)}, b...)
		e >>= 8
	}
	if len(b) == 0 {
		b = []byte{0}
	}
	return b
}

func (k *KeySet) SigningMethod() jwt.SigningMethod {
	return jwt.SigningMethodRS256
}
