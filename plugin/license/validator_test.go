package license

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewLicenseValidator(t *testing.T) {
	// Generate test key pair
	publicKey, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cacheDir := t.TempDir()

	t.Run("valid public key", func(t *testing.T) {
		validator, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if validator == nil {
			t.Fatal("Expected validator, got nil")
		}
		if validator.cacheDir != cacheDir {
			t.Errorf("Expected cache dir %s, got %s", cacheDir, validator.cacheDir)
		}
	})

	t.Run("invalid base64 public key", func(t *testing.T) {
		_, err := NewLicenseValidator("invalid-base64", cacheDir)
		if err == nil {
			t.Error("Expected error for invalid base64, got nil")
		}
		if !contains(err.Error(), "failed to decode public key") {
			t.Errorf("Expected decode error, got: %v", err)
		}
	})

	t.Run("invalid public key size", func(t *testing.T) {
		invalidKey := "dGVzdA==" // "test" in base64, too short
		_, err := NewLicenseValidator(invalidKey, cacheDir)
		if err == nil {
			t.Error("Expected error for invalid key size, got nil")
		}
		if !contains(err.Error(), "invalid public key size") {
			t.Errorf("Expected size error, got: %v", err)
		}
	})
}

func TestValidateLicense(t *testing.T) {
	cacheDir := t.TempDir()

	t.Run("valid license", func(t *testing.T) {
		// Generate test key pair for this specific test
		publicKey, privateKey, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		validator, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// Generate a valid license using the same key pair
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		// Write license to file
		licensePath := filepath.Join(t.TempDir(), "test.license")
		data, _ := json.MarshalIndent(license, "", "  ")
		if err := os.WriteFile(licensePath, data, 0644); err != nil {
			t.Fatalf("Failed to write license file: %v", err)
		}

		// Validate license
		validatedLicense, err := validator.ValidateLicense(licensePath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if validatedLicense == nil {
			t.Fatal("Expected license, got nil")
		}
		if validatedLicense.PluginName != "test-plugin" {
			t.Errorf("Expected plugin name 'test-plugin', got %s", validatedLicense.PluginName)
		}
	})

	t.Run("nonexistent license file", func(t *testing.T) {
		// Generate test key pair for this specific test
		publicKey, _, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		validator, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		_, err = validator.ValidateLicense("/nonexistent/license.license")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
		if !contains(err.Error(), "failed to read license file") {
			t.Errorf("Expected read error, got: %v", err)
		}
	})

	t.Run("invalid JSON license", func(t *testing.T) {
		// Generate test key pair for this specific test
		publicKey, _, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		validator, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		licensePath := filepath.Join(t.TempDir(), "invalid.license")
		if err := os.WriteFile(licensePath, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid license file: %v", err)
		}

		_, err = validator.ValidateLicense(licensePath)
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
		if !contains(err.Error(), "failed to parse license") {
			t.Errorf("Expected parse error, got: %v", err)
		}
	})

	t.Run("expired license", func(t *testing.T) {
		// Generate test key pair for this specific test
		publicKey, privateKey, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		validator, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// Generate an expired license
		expiresAt := time.Now().Add(-24 * time.Hour)
		license, err := GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		licensePath := filepath.Join(t.TempDir(), "expired.license")
		data, _ := json.MarshalIndent(license, "", "  ")
		if err := os.WriteFile(licensePath, data, 0644); err != nil {
			t.Fatalf("Failed to write license file: %v", err)
		}

		_, err = validator.ValidateLicense(licensePath)
		if err == nil {
			t.Error("Expected error for expired license, got nil")
		}
		if !contains(err.Error(), "license has expired") {
			t.Errorf("Expected expiration error, got: %v", err)
		}
	})
}

func TestValidateCachedLicense(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cacheDir := t.TempDir()
	validator, err := NewLicenseValidator(publicKey, cacheDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("valid cached license", func(t *testing.T) {
		// Generate and cache a license
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		if err := validator.cacheLicense(license); err != nil {
			t.Fatalf("Failed to cache license: %v", err)
		}

		// Validate cached license
		cachedLicense, err := validator.ValidateCachedLicense("test-plugin")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if cachedLicense == nil {
			t.Fatal("Expected license, got nil")
		}
		if cachedLicense.PluginName != "test-plugin" {
			t.Errorf("Expected plugin name 'test-plugin', got %s", cachedLicense.PluginName)
		}
	})

	t.Run("no cached license", func(t *testing.T) {
		_, err := validator.ValidateCachedLicense("nonexistent-plugin")
		if err == nil {
			t.Error("Expected error for nonexistent cached license, got nil")
		}
		if !contains(err.Error(), "no cached license found") {
			t.Errorf("Expected cache miss error, got: %v", err)
		}
	})

	t.Run("expired cached license", func(t *testing.T) {
		// Generate and cache an expired license
		expiresAt := time.Now().Add(-24 * time.Hour)
		license, err := GenerateLicense("expired-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		if err := validator.cacheLicense(license); err != nil {
			t.Fatalf("Failed to cache license: %v", err)
		}

		// Validate cached license should fail and remove cache
		_, err = validator.ValidateCachedLicense("expired-plugin")
		if err == nil {
			t.Error("Expected error for expired cached license, got nil")
		}
		if !contains(err.Error(), "cached license has expired") {
			t.Errorf("Expected expiration error, got: %v", err)
		}
	})

	t.Run("tampered cached license signature", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("tampered-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		if err := validator.cacheLicense(license); err != nil {
			t.Fatalf("Failed to cache license: %v", err)
		}

		cachePath := filepath.Join(cacheDir, "tampered-plugin.license")
		data, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("Failed to read cached license: %v", err)
		}

		var tampered License
		if err := json.Unmarshal(data, &tampered); err != nil {
			t.Fatalf("Failed to unmarshal cached license: %v", err)
		}
		tampered.ExpiresAt = tampered.ExpiresAt.Add(48 * time.Hour)

		tamperedData, err := json.MarshalIndent(tampered, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal tampered license: %v", err)
		}
		if err := os.WriteFile(cachePath, tamperedData, 0644); err != nil {
			t.Fatalf("Failed to write tampered cache: %v", err)
		}

		_, err = validator.ValidateCachedLicense("tampered-plugin")
		if err == nil {
			t.Fatal("Expected error for tampered cached license, got nil")
		}
		if !contains(err.Error(), "invalid cached license signature") {
			t.Errorf("Expected signature error, got: %v", err)
		}
	})

	t.Run("cached license plugin mismatch", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("mismatch-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		if err := validator.cacheLicense(license); err != nil {
			t.Fatalf("Failed to cache license: %v", err)
		}

		cachePath := filepath.Join(cacheDir, "mismatch-plugin.license")
		data, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("Failed to read cached license: %v", err)
		}

		var mismatch License
		if err := json.Unmarshal(data, &mismatch); err != nil {
			t.Fatalf("Failed to unmarshal cached license: %v", err)
		}
		mismatch.PluginName = "other-plugin"
		mismatch.Signature = ""

		licenseCopy := mismatch
		licenseCopy.Signature = ""
		signatureData, err := json.Marshal(licenseCopy)
		if err != nil {
			t.Fatalf("Failed to marshal signature data: %v", err)
		}
		hash := sha256.Sum256(signatureData)
		privateKeyDecoded, err := base64.StdEncoding.DecodeString(privateKey)
		if err != nil {
			t.Fatalf("Failed to decode private key: %v", err)
		}
		signature := ed25519.Sign(ed25519.PrivateKey(privateKeyDecoded), hash[:])
		mismatch.Signature = base64.StdEncoding.EncodeToString(signature)

		mismatchData, err := json.MarshalIndent(mismatch, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal mismatch license: %v", err)
		}
		if err := os.WriteFile(cachePath, mismatchData, 0644); err != nil {
			t.Fatalf("Failed to write mismatch cache: %v", err)
		}

		_, err = validator.ValidateCachedLicense("mismatch-plugin")
		if err == nil {
			t.Fatal("Expected plugin mismatch error, got nil")
		}
		if !contains(err.Error(), "cached license plugin mismatch") {
			t.Errorf("Expected plugin mismatch error, got: %v", err)
		}
	})
}

func TestIsLicenseValid(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cacheDir := t.TempDir()
	validator, err := NewLicenseValidator(publicKey, cacheDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("valid cached license", func(t *testing.T) {
		// Generate and cache a license
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		if err := validator.cacheLicense(license); err != nil {
			t.Fatalf("Failed to cache license: %v", err)
		}

		valid, err := validator.IsLicenseValid("test-plugin")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !valid {
			t.Error("Expected license to be valid")
		}
	})

	t.Run("no license found", func(t *testing.T) {
		valid, err := validator.IsLicenseValid("nonexistent-plugin")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if valid {
			t.Error("Expected license to be invalid")
		}
	})

	t.Run("plugin name mismatch on file path", func(t *testing.T) {
		// Generate a license for "other-plugin" but place it in "target-plugin"'s directory.
		expiresAt := time.Now().Add(24 * time.Hour)
		lic, err := GenerateLicense("other-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		pluginDir := filepath.Join(cacheDir, "..", "plugins", "target-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatalf("Failed to create plugin dir: %v", err)
		}
		data, _ := json.MarshalIndent(lic, "", "  ")
		if err := os.WriteFile(filepath.Join(pluginDir, "target-plugin.license"), data, 0644); err != nil {
			t.Fatalf("Failed to write license: %v", err)
		}

		// Create a validator that looks in this test directory.
		v2, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// Override working directory so "plugins/target-plugin/" resolves.
		origDir, _ := os.Getwd()
		_ = os.Chdir(filepath.Join(cacheDir, ".."))
		defer func() { _ = os.Chdir(origDir) }()

		valid, err := v2.IsLicenseValid("target-plugin")
		if valid {
			t.Error("Expected license to be rejected due to plugin name mismatch")
		}
		if err == nil {
			t.Error("Expected error for plugin name mismatch, got nil")
		}
		if err != nil && !contains(err.Error(), "mismatch") {
			t.Errorf("Expected mismatch error, got: %v", err)
		}
	})

	t.Run("expired license file returns error", func(t *testing.T) {
		expiresAt := time.Now().Add(-24 * time.Hour)
		lic, err := GenerateLicense("expired-file-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		pluginDir := filepath.Join(cacheDir, "..", "plugins", "expired-file-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatalf("Failed to create plugin dir: %v", err)
		}
		data, _ := json.MarshalIndent(lic, "", "  ")
		if err := os.WriteFile(filepath.Join(pluginDir, "expired-file-plugin.license"), data, 0644); err != nil {
			t.Fatalf("Failed to write license: %v", err)
		}

		v2, err := NewLicenseValidator(publicKey, cacheDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		origDir, _ := os.Getwd()
		_ = os.Chdir(filepath.Join(cacheDir, ".."))
		defer func() { _ = os.Chdir(origDir) }()

		valid, err := v2.IsLicenseValid("expired-file-plugin")
		if valid {
			t.Error("Expected expired license to be invalid")
		}
		if err == nil {
			t.Error("Expected error for expired license, got nil")
		}
	})
}

func TestGenerateLicense(t *testing.T) {
	// Generate test key pair
	_, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	t.Run("valid license generation", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if license == nil {
			t.Fatal("Expected license, got nil")
		}
		if license.PluginName != "test-plugin" {
			t.Errorf("Expected plugin name 'test-plugin', got %s", license.PluginName)
		}
		if license.CustomerID != "customer123" {
			t.Errorf("Expected customer ID 'customer123', got %s", license.CustomerID)
		}
		if license.Signature == "" {
			t.Error("Expected signature to be set")
		}
		if len(license.Features) != 2 {
			t.Errorf("Expected 2 features, got %d", len(license.Features))
		}
		if license.MaxUsers != 100 {
			t.Errorf("Expected max users 100, got %d", license.MaxUsers)
		}
	})

	t.Run("invalid private key", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		_, err := GenerateLicense("test-plugin", "customer123", expiresAt, "invalid-key")
		if err == nil {
			t.Error("Expected error for invalid private key, got nil")
		}
		if !contains(err.Error(), "failed to decode private key") {
			t.Errorf("Expected decode error, got: %v", err)
		}
	})

	t.Run("invalid private key size", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		invalidKey := "dGVzdA==" // "test" in base64, too short
		_, err := GenerateLicense("test-plugin", "customer123", expiresAt, invalidKey)
		if err == nil {
			t.Error("Expected error for invalid key size, got nil")
		}
		if !contains(err.Error(), "invalid private key size") {
			t.Errorf("Expected size error, got: %v", err)
		}
	})
}

func TestGenerateKeyPair(t *testing.T) {
	t.Run("generate valid key pair", func(t *testing.T) {
		publicKey, privateKey, err := GenerateKeyPair()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if publicKey == "" {
			t.Error("Expected public key, got empty string")
		}
		if privateKey == "" {
			t.Error("Expected private key, got empty string")
		}
		if publicKey == privateKey {
			t.Error("Public and private keys should be different")
		}
	})
}

func TestCacheLicense(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cacheDir := t.TempDir()
	validator, err := NewLicenseValidator(publicKey, cacheDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("cache valid license", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		license, err := GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		err = validator.cacheLicense(license)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Check that file was created
		cachePath := filepath.Join(cacheDir, "test-plugin.license")
		if _, err := os.Stat(cachePath); err != nil {
			t.Errorf("Expected cache file to exist, got error: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
