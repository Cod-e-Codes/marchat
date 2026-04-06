# Marchat Test Suite

This document describes the test suite for the Marchat chat application.

## Overview

The Marchat test suite provides foundational coverage of the application's core functionality, including:

- **Unit Tests**: Testing individual components and functions in isolation
- **Integration Tests**: Testing the interaction between different components
- **Crypto Tests**: Testing cryptographic functions and E2E encryption
- **Database Tests**: Testing database operations and schema management
- **Server Tests**: Testing WebSocket handling, message routing, and user management

**Note**: This is a foundational test suite with good coverage for smaller utility packages and significantly improved coverage for client and server components. Overall coverage is 33.9% across all packages.

**Database backends:** Automated tests open **SQLite** (usually in-memory or a temp file). PostgreSQL and MySQL/MariaDB are supported at runtime via `MARCHAT_DB_PATH`. The default `go test ./...` run does not start Postgres or MySQL; exercise those with manual smoke tests or your own CI against real servers. Schema creation is dialect-aware (including MySQL/MariaDB rules for indexed text).

**Release workflow (maintainers):** `.github/workflows/release.yml` uses a **`resolve-version`** job so the version string is available to Docker and all matrix legs (GitHub Actions does not support job outputs from matrix jobs). The **`build`** job sets **`CGO_ENABLED=0`** for static cross-compiled binaries. On a **published** release (not `workflow_dispatch`), **`upload-assets`** uploads all matrix **`.zip`** files with **`gh release upload`**, and the **`docker`** job appends Docker Hub pull instructions with **`gh release edit`** (GitHub CLI on the runner, **`GITHUB_TOKEN`**), avoiding third-party actions that still declare Node 20. **Termux** installs use the **linux-arm64** zip; `install.sh` / `install.ps1` map Android+aarch64 to that asset when needed.

## Test Structure

### Test Files

| File | Description | Coverage |
|------|-------------|----------|
| `internal/doctor/doctor_test.go` | CLI diagnostics | Env masking, GitHub release JSON parsing, update check with fake HTTP transport, plain `-doctor` output when writer is not a terminal, server report env includes `.env` after load |
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
| `server/db_dialect_test.go` | SQL dialect helpers | DSN → driver detection, Postgres placeholder rebinding |
| `server/message_state_test.go` | Durable reactions | Reaction persistence and replay helpers |
| `server/config_test.go` | Server configuration | Server configuration logic and validation |
| `server/client_test.go` | Server client management | WebSocket client initialization, message handling, admin operations |
| `server/health_test.go` | Server health monitoring | Health checks, system metrics, HTTP endpoints, concurrent access |
| `plugin/sdk/plugin_test.go` | Plugin SDK | Message types, JSON serialization, validation |
| `plugin/host/host_test.go` | Plugin Host | Plugin lifecycle, communication, enable/disable |
| `plugin/store/store_test.go` | Plugin Store | Registry management, platform resolution, filtering |
| `plugin/manager/manager_test.go` | Plugin Manager | Installation, uninstallation, command execution |
| `plugin/integration_test.go` | Plugin Integration | End-to-end plugin system workflows |
| `shared/version_test.go` | Version information | Version functions, variable validation, format consistency |
| `plugin/license/validator_test.go` | License validation | Signature verification, caching, expiration, tampered cache, plugin/cache key mismatch |
| `cmd/license/main_test.go` | License CLI tool functions | CLI functions (validateLicense, generateLicense, generateKeyPair, checkLicense) |

### Detailed File Coverage (from coverage profile)
| File | Coverage | Package | Description |
|------|----------|---------|-------------|
| `shared/version.go` | 100.0% | shared | Version information functions |
| `client/file_picker.go` | 98.2% | client | File selection TUI component |
| `server/health.go` | 88.9% | server | Health monitoring and status |
| `shared/crypto.go` | 81.8% | shared | Cryptographic operations |
| `plugin/license/validator.go` | 83.1% | plugin/license | License validation and verification |
| `client/crypto/keystore.go` | 79.5% | client/crypto | Keystore management, raw encrypt/decrypt |
| `config/config.go` | 78.6% | config | Configuration management |
| `server/db.go` | 75.0% | server | Database operations |
| `client/config/interactive_ui.go` | 66.6% | client/config | Interactive configuration UI |
| `server/config.go` | 66.7% | server | Server configuration |
| `server/logger.go` | 63.0% | server | Logging functionality |
| `server/hub.go` | 62.7% | server | WebSocket hub management, channels, DMs |
| `client/code_snippet.go` | 53.4% | client | Code snippet TUI component |
| `plugin/store/store.go` | 47.0% | plugin/store | Plugin store operations |
| `server/handlers.go` | 44.5% | server | HTTP/WebSocket handlers, edit/delete/pin/search |
| `cmd/license/main.go` | 42.2% | cmd/license | License CLI tool |
| `server/admin_web.go` | 36.5% | server | Admin web interface |
| `server/config_ui.go` | 35.0% | server | Server configuration UI |
| `client/config/config.go` | 32.9% | client/config | Client configuration |
| `server/client.go` | 26.1% | server | Client management, message type routing |
| `plugin/host/host.go` | 21.1% | plugin/host | Plugin hosting |
| `client/main.go` | 15.6% | client | Client main application |
| `server/admin_panel.go` | 14.8% | server | Admin panel functionality |
| `server/plugin_commands.go` | 14.2% | server | Plugin command handling |
| `plugin/manager/manager.go` | 12.4% | plugin/manager | Plugin management |
| `cmd/server/main.go` | 5.3% | cmd/server | Server main application |

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

- Go 1.25.8 or later
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
| `shared` | 85.9% | High | 283 | Small |
| `plugin/license` | 83.1% | High | 229 | Small |
| `client/crypto` | 79.5% | High | 355 | Small |
| `config` | 73.2% | High | 319 | Small |
| `client/config` | 54.5% | Medium | 1862 | Medium |
| `plugin/store` | 47.0% | Medium | 558 | Medium |
| `cmd/license` | 42.2% | Medium | 160 | Small |
| `server` | 33.7% | Medium | 6100+ | Large |
| `plugin/manager` | 23.8% | Low | 652 | Medium |
| `client` | 23.3% | Low | 4900+ | Large |
| `plugin/host` | 21.1% | Low | 566 | Medium |
| `cmd/server` | 5.3% | Low | 454 | Medium |

**Overall coverage: 34.1%** (all packages)

### High Coverage (70%+)
- **Shared Package**: Cryptographic operations, data types, message handling, version utilities (85.9%)
- **Plugin License Package**: License validation, signature verification, caching (83.1%)
- **Client Crypto Package**: Keystore management, encryption/decryption, file operations, raw encrypt/decrypt for file transfers (79.5%)
- **Config Package**: Configuration loading, validation, environment variables (73.2%)

### Medium Coverage (40-70%)
- **Client Config Package**: Configuration management, path utilities, keystore migration, interactive UI (54.5%)
- **Plugin Store**: Registry management, platform resolution, filtering, caching (47.0%)
- **Server Package**: WebSocket handling, admin panel, database operations, message edit/delete/pin/search, channels, DMs (33.7%)
- **Command License**: CLI functions for license management (42.2%)

### Low Coverage (<40%)
- **Plugin Manager**: Installation, uninstallation, command execution (23.8%)
- **Client Package**: Message rendering, user lists, encryption functions, flag validation, new message types (23.3%)
- **Plugin Host**: Plugin lifecycle management, communication, enable/disable (21.1%)
- **Server Main**: Flag parsing, configuration validation (5.3%)

### Detailed File Coverage
| File | Coverage | Package | Description |
|------|----------|---------|-------------|
| `shared/version.go` | 100.0% | shared | Version information functions |
| `client/file_picker.go` | 98.2% | client | File selection TUI component |
| `server/health.go` | 88.9% | server | Health monitoring and status |
| `shared/crypto.go` | 81.8% | shared | Cryptographic operations |
| `plugin/license/validator.go` | 83.1% | plugin/license | License validation and verification |
| `client/crypto/keystore.go` | 79.5% | client/crypto | Keystore management, raw encrypt/decrypt |
| `config/config.go` | 78.6% | config | Configuration management |
| `server/db.go` | 75.0% | server | Database operations |
| `client/config/interactive_ui.go` | 66.6% | client/config | Interactive configuration UI |
| `server/config.go` | 66.7% | server | Server configuration |
| `server/logger.go` | 63.0% | server | Logging functionality |
| `server/hub.go` | 62.7% | server | WebSocket hub management, channels, DMs |
| `client/code_snippet.go` | 53.4% | client | Code snippet TUI component |
| `plugin/store/store.go` | 47.0% | plugin/store | Plugin store operations |
| `server/handlers.go` | 44.5% | server | HTTP/WebSocket handlers, message edit/delete/pin/search |
| `cmd/license/main.go` | 42.2% | cmd/license | License CLI tool |
| `server/admin_web.go` | 36.5% | server | Admin web interface |
| `server/config_ui.go` | 35.0% | server | Server configuration UI |
| `client/config/config.go` | 32.9% | client/config | Client configuration |
| `server/client.go` | 26.1% | server | Client management, message type routing |
| `plugin/host/host.go` | 21.1% | plugin/host | Plugin hosting |
| `client/main.go` | 15.6% | client | Client main application |
| `server/admin_panel.go` | 14.8% | server | Admin panel functionality |
| `server/plugin_commands.go` | 14.2% | server | Plugin command handling |
| `plugin/manager/manager.go` | 12.4% | plugin/manager | Plugin management |
| `cmd/server/main.go` | 5.3% | cmd/server | Server main application |

### Areas for Future Testing
- **Server Package**: Advanced WebSocket handling, complex message routing scenarios (current: 33.7%)
- **Client Package**: WebSocket communication, full TUI integration (current: 23.3%)
- **Plugin Host**: Live plugin execution, WebSocket communication (current: 21.1%)
- **Plugin Manager**: Installation, uninstallation, command execution (current: 23.8%)
- **Server Main**: Full main function execution, server startup, admin panel integration (current: 5.3%)
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

- **Total Tests**: 380+ individual test cases across 13 packages
- **Coverage by Package**: 85.9% (shared), 83.1% (plugin/license), 79.5% (client/crypto), 73.2% (config), 54.5% (client/config), 47.0% (plugin/store), 42.2% (cmd/license), 33.7% (server), 23.8% (plugin/manager), 23.3% (client), 21.1% (plugin/host), 5.3% (cmd/server)
- **Overall Coverage**: 34.1% across all packages
- **Execution Time**: <3 seconds for full suite
- **Reliability**: 100% deterministic, no flaky tests, no hanging tests
- **Test Files**: 30 `_test.go` files covering core functionality, client components, plugin system, server operations, admin interfaces, and CLI diagnostics (`internal/doctor`)

This foundational test suite provides a solid base for testing core functionality, with room for significant expansion in the main application components.
