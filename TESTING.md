# Marchat Test Suite

This document describes the test suite for the Marchat chat application.

## Overview

The Marchat test suite provides foundational coverage of the application's core functionality, including:

- **Unit Tests**: Testing individual components and functions in isolation
- **Integration Tests**: Testing the interaction between different components
- **Crypto Tests**: Testing cryptographic functions and E2E encryption
- **Database Tests**: Testing database operations and schema management
- **Server Tests**: Testing WebSocket handling, message routing, and user management

**Note**: This is a foundational test suite with good coverage for smaller utility packages and significantly improved coverage for client and server components. Overall coverage is 35.3% across all packages.

## Test Structure

### Test Files

| File | Description | Coverage |
|------|-------------|----------|
| `config/config_test.go` | Configuration loading and validation | Environment variables, validation rules |
| `shared/crypto_test.go` | Cryptographic operations | Key generation, encryption, decryption, session keys |
| `shared/types_test.go` | Data structures and serialization | Message types, JSON marshaling/unmarshaling |
| `client/crypto/keystore_test.go` | Client keystore management | Keystore initialization, encryption/decryption, file I/O |
| `client/config/config_test.go` | Client configuration management | Config loading/saving, path utilities, keystore migration |
| `client/config/interactive_ui_test.go` | Client interactive UI components | TUI forms, profile selection, authentication prompts |
| `client/code_snippet_test.go` | Client code snippet functionality | Text editing, selection, clipboard, syntax highlighting |
| `client/file_picker_test.go` | Client file picker functionality | File browsing, selection, size validation, directory navigation |
| `client/main_test.go` | Client main functionality | Message rendering, user lists, URL handling, encryption functions, flag validation |
| `cmd/server/main_test.go` | Server main function and startup | Flag parsing, configuration validation, TLS setup, admin management |
| `server/handlers_test.go` | Server-side request handling | Database operations, message insertion, IP extraction |
| `server/hub_test.go` | WebSocket hub management | User bans, kicks, connection management |
| `server/integration_test.go` | End-to-end workflows | Message flow, ban flow, concurrent operations |
| `server/admin_web_test.go` | Admin web interface | HTTP endpoints, authentication, admin panel functionality |
| `server/config_ui_test.go` | Server configuration UI | Configuration management, environment handling |
| `server/admin_panel_test.go` | Admin panel functionality | Admin-specific operations and controls |
| `server/db_test.go` | Database operations | Database initialization, schema setup |
| `server/config_test.go` | Server configuration | Server configuration logic and validation |
| `server/client_test.go` | Server client management | WebSocket client initialization, message handling, admin operations |
| `server/health_test.go` | Server health monitoring | Health checks, system metrics, HTTP endpoints, concurrent access |
| `plugin/sdk/plugin_test.go` | Plugin SDK | Message types, JSON serialization, validation |
| `plugin/host/host_test.go` | Plugin Host | Plugin lifecycle, communication, enable/disable |
| `plugin/store/store_test.go` | Plugin Store | Registry management, platform resolution, filtering |
| `plugin/manager/manager_test.go` | Plugin Manager | Installation, uninstallation, command execution |
| `plugin/integration_test.go` | Plugin Integration | End-to-end plugin system workflows |
| `shared/version_test.go` | Version information | Version functions, variable validation, format consistency |
| `plugin/license/validator_test.go` | License validation | Signature verification, caching, expiration checks |
| `cmd/license/main_test.go` | License CLI tool functions | CLI functions (validateLicense, generateLicense, generateKeyPair, checkLicense) |

### Detailed File Coverage (from coverage.html)
| File | Coverage | Package | Description |
|------|----------|---------|-------------|
| `shared/version.go` | 100.0% | shared | Version information functions |
| `client/file_picker.go` | 98.2% | client | File selection TUI component |
| `server/health.go` | 88.9% | server | Health monitoring and status |
| `plugin/license/validator.go` | 83.1% | plugin/license | License validation and verification |
| `shared/crypto.go` | 81.8% | shared | Cryptographic operations |
| `config/config.go` | 78.6% | config | Configuration management |
| `client/crypto/keystore.go` | 76.5% | client/crypto | Keystore management |
| `server/db.go` | 75.0% | server | Database operations |
| `client/config/interactive_ui.go` | 66.6% | client/config | Interactive configuration UI |
| `server/config.go` | 66.7% | server | Server configuration |
| `server/logger.go` | 63.0% | server | Logging functionality |
| `server/hub.go` | 62.7% | server | WebSocket hub management |
| `client/code_snippet.go` | 53.4% | client | Code snippet TUI component |
| `plugin/store/store.go` | 46.8% | plugin/store | Plugin store operations |
| `server/handlers.go` | 44.5% | server | HTTP/WebSocket handlers |
| `cmd/license/main.go` | 42.2% | cmd/license | License CLI tool |
| `server/admin_web.go` | 36.5% | server | Admin web interface |
| `server/config_ui.go` | 35.0% | server | Server configuration UI |
| `client/config/config.go` | 32.9% | client/config | Client configuration |
| `server/client.go` | 26.1% | server | Client management |
| `plugin/host/host.go` | 22.3% | plugin/host | Plugin hosting |
| `client/main.go` | 15.6% | client | Client main application |
| `server/admin_panel.go` | 14.8% | server | Admin panel functionality |
| `plugin/manager/manager.go` | 12.4% | plugin/manager | Plugin management |
| `server/plugin_commands.go` | 14.2% | server | Plugin command handling |
| `cmd/server/main.go` | 5.6% | cmd/server | Server main application |

### Test Categories

#### 1. Unit Tests
- **Crypto Functions**: Key generation, encryption/decryption, session key derivation
- **Data Types**: Message structures, JSON serialization, validation
- **Utility Functions**: IP extraction, message sorting, database stats
- **Configuration**: Environment variable parsing, validation rules
- **Client Keystore**: Keystore initialization, encryption/decryption, file operations, passphrase handling
- **Client Config**: Configuration loading/saving, path utilities, keystore migration
- **Client Interactive UI**: TUI forms, profile selection, authentication prompts, navigation, validation
- **Client Code Snippet**: Text editing, selection, clipboard operations, syntax highlighting, state management
- **Client Main**: Message rendering, user lists, URL handling, encryption functions, flag validation
- **Client File Picker**: File browsing, directory navigation, file selection, size validation, error handling
- **Server Main**: Flag parsing, multi-flag handling, banner display, admin username normalization
- **Server Admin Web**: HTTP endpoints, authentication, admin panel functionality, web interface
- **Server Configuration**: Configuration management, environment handling, UI components
- **Server Database**: Database initialization, schema setup, connection management
- **Server Admin Panel**: Admin-specific operations, user management, administrative controls

#### 2. Integration Tests
- **Message Flow**: Complete message lifecycle from insertion to retrieval
- **User Management**: Ban/kick/unban workflows with database persistence
- **Concurrent Operations**: Thread-safe operations and race condition testing
- **Database Operations**: Schema creation, message capping, backup functionality

#### 3. Server Tests
- **WebSocket Handling**: Connection management, authentication, message routing
- **Hub Management**: User registration, broadcasting, cleanup operations
- **Admin Functions**: Ban management, user administration, system commands
- **Server Main Function**: Flag parsing, configuration validation, TLS setup, admin username normalization, banner display
- **Client Management**: WebSocket client initialization, message handling, admin operations, connection settings
- **Health Monitoring**: Database health checks, system metrics collection, HTTP endpoints, concurrent access safety

## Running Tests

### Prerequisites

- Go 1.24 or later
- SQLite support (built into Go)
- PowerShell (for Windows test script)

### Basic Test Execution

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./server
go test ./shared
go test ./plugin/sdk
```

### Using Test Scripts

#### Linux/macOS
```bash
# Run the test suite
./test.sh
```

#### Windows PowerShell
```powershell
# Run basic test suite
.\test.ps1

# Run with coverage report
.\test.ps1 -Coverage

# Run with verbose output
.\test.ps1 -Verbose
```

### Test Coverage

Generate and view test coverage:

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func coverage.out

# Generate HTML coverage report
go tool cover -html coverage.out -o coverage.html

# View coverage percentages directly
go test -cover ./...
```

## Test Coverage Areas

### Current Coverage Status

| Package | Coverage | Status | Lines of Code | Weighted Impact |
|---------|----------|--------|---------------|-----------------|
| `plugin/license` | 83.1% | High | 229 | Small |
| `shared` | 82.4% | High | 283 | Small |
| `config` | 78.6% | High | 304 | Small |
| `client/crypto` | 76.5% | High | 289 | Small |
| `client/config` | 54.5% | Medium | 1816 | Medium |
| `plugin/store` | 46.8% | Medium | 559 | Medium |
| `cmd/license` | 42.2% | Medium | 160 | Small |
| `server` | 32.6% | Medium | 5500+ | Large |
| `client` | 26.2% | Medium | 3700+ | Large |
| `plugin/host` | 23.7% | Low | 486 | Medium |
| `plugin/manager` | 22.5% | Low | 464 | Medium |
| `cmd/server` | 5.3% | Low | 403 | Medium |

**Overall coverage: 35.3%** (all packages)

### High Coverage (70%+)
- **Plugin License Package**: License validation, signature verification, caching (83.1%)
- **Shared Package**: Cryptographic operations, data types, message handling (82.4%)
- **Config Package**: Configuration loading, validation, environment variables (78.6%)
- **Client Crypto Package**: Keystore management, encryption/decryption, file operations (76.5%)

### Medium Coverage (40-70%)
- **Client Config Package**: Configuration management, path utilities, keystore migration, interactive UI (55.2%)
- **Plugin Store**: Registry management, platform resolution, filtering, caching (46.8%)
- **Server Package**: WebSocket handling, admin panel, database operations, configuration management (27.0%)
- **Client Package**: Message rendering, user lists, encryption functions, flag validation (27.8%)

### Low Coverage (<40%)
- **Plugin Host**: Plugin lifecycle management, communication, enable/disable (22.3%)
- **Plugin Manager**: Installation, uninstallation, command execution (12.4%)
- **Server Main**: Flag parsing, configuration validation (5.6%)

### Detailed File Coverage
| File | Coverage | Package | Description |
|------|----------|---------|-------------|
| `shared/version.go` | 100.0% | shared | Version information functions |
| `client/file_picker.go` | 98.2% | client | File selection TUI component |
| `server/health.go` | 88.9% | server | Health monitoring and status |
| `plugin/license/validator.go` | 83.1% | plugin/license | License validation and verification |
| `shared/crypto.go` | 81.8% | shared | Cryptographic operations |
| `config/config.go` | 78.6% | config | Configuration management |
| `client/crypto/keystore.go` | 76.5% | client/crypto | Keystore management |
| `server/db.go` | 75.0% | server | Database operations |
| `client/config/interactive_ui.go` | 66.6% | client/config | Interactive configuration UI |
| `server/config.go` | 66.7% | server | Server configuration |
| `server/logger.go` | 63.0% | server | Logging functionality |
| `server/hub.go` | 62.7% | server | WebSocket hub management |
| `client/code_snippet.go` | 53.4% | client | Code snippet TUI component |
| `plugin/store/store.go` | 46.8% | plugin/store | Plugin store operations |
| `server/handlers.go` | 44.5% | server | HTTP/WebSocket handlers |
| `cmd/license/main.go` | 42.2% | cmd/license | License CLI tool |
| `server/admin_web.go` | 36.5% | server | Admin web interface |
| `server/config_ui.go` | 35.0% | server | Server configuration UI |
| `client/config/config.go` | 32.9% | client/config | Client configuration |
| `server/client.go` | 26.1% | server | Client management |
| `plugin/host/host.go` | 22.3% | plugin/host | Plugin hosting |
| `client/main.go` | 15.6% | client | Client main application |
| `server/admin_panel.go` | 14.8% | server | Admin panel functionality |
| `server/plugin_commands.go` | 14.2% | server | Plugin command handling |
| `plugin/manager/manager.go` | 12.4% | plugin/manager | Plugin management |
| `cmd/server/main.go` | 5.6% | cmd/server | Server main application |

### Areas for Future Testing
- **Server Package**: Advanced WebSocket handling, complex message routing scenarios (current: 35.4%)
- **Client Package**: WebSocket communication, full TUI integration (current: 27.8%)
- **Plugin Host**: Live plugin execution, WebSocket communication (current: 22.3%)
- **Plugin Manager**: Installation, uninstallation, command execution (current: 12.4%)
- **Server Main**: Full main function execution, server startup, admin panel integration (current: 5.6%)
- **File Transfer**: File upload/download functionality
- **Plugin License**: License validation and enforcement (current: 83.1%)

## Test Data and Fixtures

### Database Tests
- Uses in-memory SQLite databases for isolation
- Creates fresh schema for each test
- Tests both encrypted and plaintext messages
- Verifies message ordering and retrieval

### Cryptographic Tests
- Generates real keypairs using X25519
- Tests ChaCha20-Poly1305 encryption
- Verifies session key derivation
- Tests key validation and ID generation
- **Client Keystore**: Tests keystore initialization, encryption/decryption, file operations, passphrase handling

### Message Tests
- Tests various message types (text, file, admin)
- Verifies JSON serialization/deserialization
- Tests message ordering and timestamp handling
- Validates encrypted message handling

### Server Main Tests
- Tests command-line flag parsing and validation
- Verifies multi-flag functionality for admin users
- Tests configuration loading and environment variable handling
- Validates TLS configuration and WebSocket scheme selection
- Tests admin username normalization and duplicate detection
- Verifies banner display functionality
- Tests deprecated flag warnings and backward compatibility

## Continuous Integration

The test suite is designed to run in CI/CD environments:

- **Fast Execution**: Most tests complete in under 1 second
- **No External Dependencies**: Uses only Go standard library and SQLite
- **Parallel Safe**: Tests can run concurrently
- **Deterministic**: No flaky tests or race conditions

## Adding New Tests

### Guidelines

1. **Test Naming**: Use descriptive test names that explain the scenario
2. **Test Structure**: Follow the Arrange-Act-Assert pattern
3. **Isolation**: Each test should be independent and not rely on other tests
4. **Coverage**: Aim for meaningful coverage, not just line coverage
5. **Documentation**: Add comments for complex test scenarios

### Example Test Structure

```go
func TestFeatureName(t *testing.T) {
    // Arrange
    setup := createTestSetup()
    input := createTestInput()
    
    // Act
    result, err := functionUnderTest(input)
    
    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if result != expectedResult {
        t.Errorf("Expected %v, got %v", expectedResult, result)
    }
}
```

### Database Test Pattern

```go
func TestDatabaseOperation(t *testing.T) {
    // Create test database
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }
    defer db.Close()
    
    // Create schema
    CreateSchema(db)
    
    // Test the operation
    // ... test implementation
}
```

## Performance Considerations

- **In-Memory Databases**: Tests use SQLite in-memory mode for speed
- **Parallel Execution**: Tests are designed to run in parallel when possible
- **Minimal Setup**: Each test creates only the data it needs
- **Fast Cleanup**: Tests clean up resources immediately

## Troubleshooting

### Common Issues

1. **Import Errors**: Ensure all dependencies are properly imported
2. **Database Locks**: Tests use separate in-memory databases to avoid conflicts
3. **Race Conditions**: All concurrent tests use proper synchronization
4. **Memory Leaks**: Tests properly close database connections and channels
5. **PowerShell Execution Policy**: May need to enable script execution: `Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser`
6. **Coverage Tool Syntax**: Use `go tool cover -func coverage.out` (without equals sign)

### Debug Mode

Run tests with debug output:

```bash
# Enable Go test debugging
go test -v -race ./...
```

## Contributing

When adding new functionality to Marchat:

1. **Write Tests First**: Follow TDD principles where possible
2. **Update This Document**: Add new test categories and coverage areas
3. **Maintain Coverage**: Ensure new code is properly tested
4. **Run Full Suite**: Always run all tests before submitting changes

## Test Metrics

- **Total Tests**: 366+ individual test cases across 11 packages
- **Coverage by Package**: 83.1% (plugin/license), 82.4% (shared), 78.6% (config), 76.5% (client/crypto), 54.5% (client/config), 46.8% (plugin/store), 42.2% (cmd/license), 32.6% (server), 26.2% (client), 23.7% (plugin/host), 22.5% (plugin/manager), 5.3% (cmd/server)
- **Overall Coverage**: 35.3% across all packages
- **Execution Time**: <3 seconds for full suite
- **Reliability**: 100% deterministic, no flaky tests, no hanging tests
- **Test Files**: 26 test files covering core functionality, client components, plugin system, server operations, and admin interfaces

This foundational test suite provides a solid base for testing core functionality, with room for significant expansion in the main application components.
