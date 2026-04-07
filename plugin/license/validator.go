package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// License represents a plugin license
type License struct {
	PluginName string    `json:"plugin_name"`
	CustomerID string    `json:"customer_id"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Features   []string  `json:"features"`
	MaxUsers   int       `json:"max_users,omitempty"`
	Signature  string    `json:"signature"`
}

// LicenseValidator validates plugin licenses
type LicenseValidator struct {
	publicKey ed25519.PublicKey
	cacheDir  string
}

// NewLicenseValidator creates a new license validator
func NewLicenseValidator(publicKeyBase64, cacheDir string) (*LicenseValidator, error) {
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size")
	}

	return &LicenseValidator{
		publicKey: ed25519.PublicKey(publicKey),
		cacheDir:  cacheDir,
	}, nil
}

// ValidateLicense validates a license file
func (lv *LicenseValidator) ValidateLicense(licensePath string) (*License, error) {
	// Read license file
	data, err := os.ReadFile(licensePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read license file: %w", err)
	}

	// Parse license
	var license License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, fmt.Errorf("failed to parse license: %w", err)
	}

	// Validate signature
	if err := lv.validateSignature(&license, nil); err != nil {
		return nil, fmt.Errorf("invalid license signature: %w", err)
	}

	// Check expiration
	if time.Now().After(license.ExpiresAt) {
		return nil, fmt.Errorf("license has expired")
	}

	// Cache valid license
	if err := lv.cacheLicense(&license); err != nil {
		return nil, fmt.Errorf("failed to cache license: %w", err)
	}

	return &license, nil
}

// ValidateCachedLicense loads a license from the on-disk cache, verifies plugin_name matches
// the requested plugin and re-checks the Ed25519 signature, then enforces expiry. Bad cache
// files are removed.
func (lv *LicenseValidator) ValidateCachedLicense(pluginName string) (*License, error) {
	cachePath := filepath.Join(lv.cacheDir, pluginName+".license")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("no cached license found: %w", err)
	}

	var license License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, fmt.Errorf("failed to parse cached license: %w", err)
	}

	if license.PluginName != pluginName {
		_ = os.Remove(cachePath)
		return nil, fmt.Errorf("cached license plugin mismatch")
	}

	if err := lv.validateSignature(&license, nil); err != nil {
		_ = os.Remove(cachePath)
		return nil, fmt.Errorf("invalid cached license signature: %w", err)
	}

	// Check if cached license is still valid
	if time.Now().After(license.ExpiresAt) {
		// Remove expired cache
		_ = os.Remove(cachePath)
		return nil, fmt.Errorf("cached license has expired")
	}

	return &license, nil
}

// validateSignature validates the license signature
func (lv *LicenseValidator) validateSignature(license *License, data []byte) error {
	// Use provided data if available, otherwise create from license
	var signatureData []byte
	var err error

	if len(data) > 0 {
		// Use the provided data for signature validation
		signatureData = data
	} else {
		// Create the data that was signed (everything except the signature)
		licenseCopy := *license
		licenseCopy.Signature = ""
		signatureData, err = json.Marshal(licenseCopy)
		if err != nil {
			return fmt.Errorf("failed to marshal license data: %w", err)
		}
	}

	// Hash the data
	hash := sha256.Sum256(signatureData)

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(license.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(lv.publicKey, hash[:], signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// cacheLicense caches a valid license
func (lv *LicenseValidator) cacheLicense(license *License) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(lv.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write license to cache
	cachePath := filepath.Join(lv.cacheDir, license.PluginName+".license")
	data, err := json.MarshalIndent(license, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal license: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// IsLicenseValid checks if a license is valid for a plugin.
// Returns (false, nil) when no license file exists; returns (false, err) when a
// license was found but failed validation (signature, expiry, or name mismatch).
func (lv *LicenseValidator) IsLicenseValid(pluginName string) (bool, error) {
	if _, err := lv.ValidateCachedLicense(pluginName); err == nil {
		return true, nil
	}

	licensePath := filepath.Join("plugins", pluginName, pluginName+".license")
	if _, err := os.Stat(licensePath); err == nil {
		license, err := lv.ValidateLicense(licensePath)
		if err != nil {
			return false, fmt.Errorf("license validation failed for %s: %w", pluginName, err)
		}
		if license.PluginName != pluginName {
			return false, fmt.Errorf("license plugin name mismatch: expected %s, got %s", pluginName, license.PluginName)
		}
		return true, nil
	}

	return false, nil
}

// GenerateLicense generates a test license (for development only)
func GenerateLicense(pluginName, customerID string, expiresAt time.Time, privateKeyBase64 string) (*License, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}

	license := &License{
		PluginName: pluginName,
		CustomerID: customerID,
		IssuedAt:   time.Now(),
		ExpiresAt:  expiresAt,
		Features:   []string{"basic", "advanced"},
		MaxUsers:   100,
	}

	// Create signature data
	licenseCopy := *license
	licenseCopy.Signature = ""

	signatureData, err := json.Marshal(licenseCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license data: %w", err)
	}

	// Hash and sign
	hash := sha256.Sum256(signatureData)
	signature := ed25519.Sign(ed25519.PrivateKey(privateKey), hash[:])

	license.Signature = base64.StdEncoding.EncodeToString(signature)

	return license, nil
}

// GenerateKeyPair generates a new key pair for license signing
func GenerateKeyPair() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	publicKey = base64.StdEncoding.EncodeToString(pub)
	privateKey = base64.StdEncoding.EncodeToString(priv)

	return publicKey, privateKey, nil
}
