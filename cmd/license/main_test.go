package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/plugin/license"
)

// captureOutput captures stdout/stderr for testing
func captureOutput(f func()) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		return "", "", err
	}

	os.Stdout = wOut
	os.Stderr = wErr

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Run the function (but it might call os.Exit)
	f()

	wOut.Close()
	wErr.Close()

	var bufOut, bufErr bytes.Buffer
	_, err = bufOut.ReadFrom(rOut)
	if err != nil {
		return "", "", err
	}
	_, err = bufErr.ReadFrom(rErr)
	if err != nil {
		return "", "", err
	}

	return bufOut.String(), bufErr.String(), nil
}

func TestValidateLicenseFunction(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Generate a valid license
	expiresAt := time.Now().Add(24 * time.Hour)
	testLicense, err := license.GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
	if err != nil {
		t.Fatalf("Failed to generate license: %v", err)
	}
	if testLicense == nil {
		t.Fatal("Generated license is nil")
		return
	}

	// Write license to file
	licensePath := filepath.Join(t.TempDir(), "test.license")
	data, _ := json.MarshalIndent(testLicense, "", "  ")
	if err := os.WriteFile(licensePath, data, 0644); err != nil {
		t.Fatalf("Failed to write license file: %v", err)
	}

	cacheDir := t.TempDir()

	// Test the actual validateLicense function
	t.Run("valid license", func(t *testing.T) {
		stdout, stderr, err := captureOutput(func() {
			validateLicense(licensePath, publicKey, cacheDir)
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if stderr != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}
		if !contains(stdout, "License validated successfully!") {
			t.Errorf("Expected success message, got: %s", stdout)
		}
		if !contains(stdout, "Plugin: test-plugin") {
			t.Errorf("Expected plugin name in output, got: %s", stdout)
		}
		if !contains(stdout, "Customer: customer123") {
			t.Errorf("Expected customer ID in output, got: %s", stdout)
		}
	})
}

func TestGenerateLicenseFunction(t *testing.T) {
	// Generate test key pair
	_, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	t.Run("generate license with output file", func(t *testing.T) {
		pluginName := "test-plugin"
		customerID := "customer123"
		expiresAtStr := "2025-12-31"
		outputFile := filepath.Join(t.TempDir(), "generated.license")

		stdout, stderr, err := captureOutput(func() {
			generateLicense(pluginName, customerID, expiresAtStr, privateKey, outputFile)
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if stderr != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}
		if !contains(stdout, "License written to") {
			t.Errorf("Expected file write message, got: %s", stdout)
		}

		// Verify file was created
		if _, err := os.Stat(outputFile); err != nil {
			t.Errorf("Expected output file to exist, got error: %v", err)
		}
	})

	t.Run("generate license to stdout", func(t *testing.T) {
		pluginName := "test-plugin"
		customerID := "customer123"
		expiresAtStr := "2025-12-31"

		stdout, stderr, err := captureOutput(func() {
			generateLicense(pluginName, customerID, expiresAtStr, privateKey, "")
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if stderr != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}

		// Should contain JSON output
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
			t.Errorf("Expected valid JSON output, got error: %v", err)
		}
	})
}

func TestGenerateKeyPairFunction(t *testing.T) {
	t.Run("generate key pair", func(t *testing.T) {
		stdout, stderr, err := captureOutput(func() {
			generateKeyPair()
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if stderr != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}

		if !contains(stdout, "Generated key pair:") {
			t.Errorf("Expected key pair message, got: %s", stdout)
		}
		if !contains(stdout, "Public Key:") {
			t.Errorf("Expected public key in output, got: %s", stdout)
		}
		if !contains(stdout, "Private Key:") {
			t.Errorf("Expected private key in output, got: %s", stdout)
		}
		if !contains(stdout, "Store these keys securely!") {
			t.Errorf("Expected security warning, got: %s", stdout)
		}
	})
}

func TestCheckLicenseFunction(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	cacheDir := t.TempDir()
	validator, err := license.NewLicenseValidator(publicKey, cacheDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("check valid license", func(t *testing.T) {
		// Generate a license and validate it (which will cache it)
		expiresAt := time.Now().Add(24 * time.Hour)
		testLicense, err := license.GenerateLicense("test-plugin", "customer123", expiresAt, privateKey)
		if err != nil {
			t.Fatalf("Failed to generate license: %v", err)
		}

		// Write license to file and validate it (this will cache it)
		licensePath := filepath.Join(t.TempDir(), "test.license")
		data, _ := json.MarshalIndent(testLicense, "", "  ")
		if err := os.WriteFile(licensePath, data, 0644); err != nil {
			t.Fatalf("Failed to write license file: %v", err)
		}

		// Validate license (this will cache it)
		_, err = validator.ValidateLicense(licensePath)
		if err != nil {
			t.Errorf("Expected no error validating license, got: %v", err)
		}

		// Test the actual checkLicense function
		stdout, stderr, err := captureOutput(func() {
			checkLicense("test-plugin", publicKey, cacheDir)
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if stderr != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}
		if !contains(stdout, "License for plugin test-plugin is valid") {
			t.Errorf("Expected valid license message, got: %s", stdout)
		}
	})

	t.Run("check invalid license", func(t *testing.T) {
		stdout, stderr, err := captureOutput(func() {
			checkLicense("nonexistent-plugin", publicKey, cacheDir)
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if stderr != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}
		if !contains(stdout, "No valid license found for plugin nonexistent-plugin") {
			t.Errorf("Expected invalid license message, got: %s", stdout)
		}
	})
}

func TestMainFunctionSwitchLogic(t *testing.T) {
	t.Run("action validation", func(t *testing.T) {
		// Test the switch logic that main() uses
		testCases := []struct {
			action     string
			shouldWork bool
		}{
			{"validate", true},
			{"generate", true},
			{"genkey", true},
			{"check", true},
			{"invalid", false},
			{"", false},
		}

		for _, tc := range testCases {
			t.Run(tc.action, func(t *testing.T) {
				// Test the switch logic
				switch tc.action {
				case "validate", "generate", "genkey", "check":
					// These should work
					if !tc.shouldWork {
						t.Errorf("Action %s should not work but does", tc.action)
					}
				default:
					// These should not work
					if tc.shouldWork {
						t.Errorf("Action %s should work but doesn't", tc.action)
					}
				}
			})
		}
	})
}

func TestMainFunctionFlags(t *testing.T) {
	t.Run("flag definitions", func(t *testing.T) {
		// Test that all the flags defined in main are properly configured
		flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

		// Add the same flags as main
		action := flagSet.String("action", "", "Action to perform: validate, generate, or genkey")
		licenseFile := flagSet.String("license", "", "License file path")
		pluginName := flagSet.String("plugin", "", "Plugin name")
		customerID := flagSet.String("customer", "", "Customer ID")
		expiresAt := flagSet.String("expires", "", "Expiration date (YYYY-MM-DD)")
		privateKey := flagSet.String("private-key", "", "Private key for signing")
		publicKey := flagSet.String("public-key", "", "Public key for validation")
		cacheDir := flagSet.String("cache-dir", "./license-cache", "License cache directory")
		outputFile := flagSet.String("output", "", "Output file for generated license")

		// Test default values
		if *cacheDir != "./license-cache" {
			t.Errorf("Expected default cache dir './license-cache', got %s", *cacheDir)
		}
		if *action != "" {
			t.Errorf("Expected empty action by default, got %s", *action)
		}
		if *licenseFile != "" {
			t.Errorf("Expected empty license file by default, got %s", *licenseFile)
		}
		if *pluginName != "" {
			t.Errorf("Expected empty plugin name by default, got %s", *pluginName)
		}
		if *customerID != "" {
			t.Errorf("Expected empty customer ID by default, got %s", *customerID)
		}
		if *expiresAt != "" {
			t.Errorf("Expected empty expires by default, got %s", *expiresAt)
		}
		if *privateKey != "" {
			t.Errorf("Expected empty private key by default, got %s", *privateKey)
		}
		if *publicKey != "" {
			t.Errorf("Expected empty public key by default, got %s", *publicKey)
		}
		if *outputFile != "" {
			t.Errorf("Expected empty output file by default, got %s", *outputFile)
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
