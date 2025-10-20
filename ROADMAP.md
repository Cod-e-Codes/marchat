## Roadmap

### Phase 1 — DB Abstraction Layer
- Refactor database connection and initialization logic into a unified function.
- Dynamically select DB driver and connection string at runtime.
- Add support for PostgreSQL and MySQL in addition to SQLite.

### Phase 2 — Multi-Backend Compatibility
- Ensure schema and queries work with:
  - SQLite (existing)
  - PostgreSQL
  - MySQL
- Adjust types where necessary (BOOLEAN, TIMESTAMP/DATETIME).
- Consider a unified schema that works across all backends.

### Phase 3 — Schema & Query Adaptation
- Add conditional logic for DB-specific schema tweaks.
- Validate CREATE TABLE statements in all target backends.
- Test queries for compatibility and performance.

### Phase 4 — Performance Enhancements
- ✅ Enable SQLite Write-Ahead Logging (WAL) mode for performance gains.
- Implement batch TTL-based message deletion.
- Add indexing for frequently queried columns.
- Cache displayed messages in the terminal to reduce DB queries.

### Phase 5 — Testing & Documentation
- Write unit/integration tests for all supported backends.
- Document setup steps for PostgreSQL and MySQL.
- Provide working connection string examples.
- Include troubleshooting tips for common DB connection issues.

### Phase 6 — Future Improvements
- Consider using `sqlx` or a lightweight ORM to reduce SQL dialect handling.
- Explore migrations tooling for schema changes.
- Evaluate other client-server DB options based on user demand.
