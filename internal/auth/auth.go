package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
)

type keyRecord struct {
	KeyID    string
	Hash     string
	TenantID string
}

type AuthManager struct {
	mu   sync.RWMutex
	Keys map[string]keyRecord
}

func NewAuthManager() *AuthManager {
	return &AuthManager{
		Keys: make(map[string]keyRecord),
	}
}

func (a *AuthManager) CreateKey(cts context.Context, tenantID string) (string, error) {
	// Generate random 16-byte secret
	secret := make([]byte, 16)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Generate random 8-byte keyID
	keyIDBytes := make([]byte, 8)
	_, err = rand.Read(keyIDBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate keyID: %w", err)
	}

	keyID := base64.RawURLEncoding.EncodeToString(keyIDBytes)

	// Hash secret
	hash := sha256.Sum256(secret)
	encodedHash := base64.RawURLEncoding.EncodeToString(hash[:])

	// Store
	a.mu.Lock()
	a.Keys[keyID] = keyRecord{KeyID: keyID, TenantID: tenantID, Hash: encodedHash}
	a.mu.Unlock()

	// Return "kid.secret" (secret base64url encoded)
	secretStr := base64.RawURLEncoding.EncodeToString(secret)
	return fmt.Sprintf("%s.%s", keyID, secretStr), nil

}

func (a *AuthManager) ValidateKey(ctx context.Context, token string) (string, string, bool) {

	kid, secretStr, ok := strings.Cut(token, ".")

	if !ok {
		return "", "", false
	}

	secret, err := base64.RawURLEncoding.DecodeString(secretStr)

	if err != nil {
		return "", "", false
	}

	a.mu.RLock()
	record, ok := a.Keys[kid]
	a.mu.RUnlock()

	if !ok {
		return "", "", false
	}

	// Recompute hash
	hash := sha256.Sum256(secret)
	expected, err := base64.RawURLEncoding.DecodeString(record.Hash)
	if err != nil {
		return "", "", false
	}

	// Constant-time compare
	if hmac.Equal(hash[:], expected) {
		return kid, record.TenantID, true
	}
	return "", "", false

}

// RevokeKey removes a key by KeyID.
func (a *AuthManager) RevokeKey(ctx context.Context, kid string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.Keys[kid]; !ok {
		return fmt.Errorf("key not found")
	}
	delete(a.Keys, kid)
	return nil
}
