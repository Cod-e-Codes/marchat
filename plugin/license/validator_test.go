package license

import (
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
