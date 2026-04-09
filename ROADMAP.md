## Roadmap

This file tracks implementation status. Completed items use `- [x]`; future ideas are plain bullets without a checkbox.

### Recently Completed
- [x] CLI diagnostics (`-doctor`, `-doctor-json`) on client and server binaries for env/config/update checks
- [x] Message editing, deletion, and pinning
- [x] Message reactions
- [x] Direct messages between users
- [x] Chat channels (join/leave with per-channel messaging)
- [x] Typing indicators and read receipts
- [x] Message search (server-side)
- [x] E2E encryption for file transfers
- [x] Connection status indicator, @mention tab completion, unread count
- [x] Multi-line input (Alt+Enter / Ctrl+J)
- [x] Chat history export
- [x] WebSocket rate limiting
- [x] Docker Compose for local development

### Phase 1: DB Abstraction Layer
- [x] Refactor database connection and initialization logic into a unified function.
- [x] Dynamically select DB driver and connection string at runtime.
- [x] Add support for PostgreSQL and MySQL in addition to SQLite.

### Phase 2: Multi-Backend Compatibility
- [x] Ensure schema and queries work with SQLite, PostgreSQL, and MySQL.
- [x] Adjust types where necessary (BOOLEAN, TIMESTAMP/DATETIME).
- [x] Maintain a unified schema that works across all backends.

### Phase 3: Schema & Query Adaptation
- [x] Add conditional logic for DB-specific schema tweaks.
- [x] Validate CREATE TABLE statements in all target backends.
- [x] Test queries for compatibility and performance.

### Phase 4: Performance Enhancements
- [x] Enable SQLite Write-Ahead Logging (WAL) mode for performance gains.
- [x] Implement batch TTL-based message deletion.
- [x] Add indexing for frequently queried columns.
- [x] Cache displayed/recent messages in server memory to reduce repeated DB reads.

### Phase 5: Persistence & Durability
- [x] Persist reactions to the database.
- [x] Persist last channel per user across reconnects (`user_channels`).
- [x] Add read receipt state tracking per user.

### Phase 6: Testing & Documentation
- [x] Dialect tests (DSN detection, placeholder rebinding) and SQLite-backed integration tests; exercise PostgreSQL/MySQL against live databases separately.
- [x] Document setup steps for PostgreSQL and MySQL.
- [x] Provide working connection string examples.
- [x] Include troubleshooting tips for common DB connection issues.
- [x] Increase test coverage for client and server packages.

### Phase 7: Future Improvements
- Consider using `sqlx` or a lightweight ORM to reduce SQL dialect handling.
- Explore migrations tooling for schema changes.
- Evaluate other client-server DB options based on user demand.
- Per-user notification rules.
- Plugin auto-updates and dependency resolution.
