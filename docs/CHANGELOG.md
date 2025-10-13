# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2025-10-13]

### Added
**Aggregate Repository Pattern with Comprehensive Testing**: Implemented complete aggregate repository pattern supporting complex entity relationships with automatic test generation.
- **SQLite Aggregate Repositories**: Full transactional support with foreign key constraints and cascade operations for aggregate roots with child entities.
- **MongoDB Aggregate Repositories**: Native document operations with embedded child entity handling.
- **Table-Based Test Generation**: Automatic generation of comprehensive table-driven tests for SQLite repositories achieving 72.1% code coverage out of the box.
- **Test Infrastructure**: Temporary database files for reliable test isolation, automatic cleanup, and helper functions for setup/teardown.
- **Repository Interfaces**: Type-safe repository interfaces with full CRUD operations (Create, Get, Save, Delete, List) for aggregate patterns.

### Technical Details
- Template-driven code generation for aggregate roots and child collections with proper Go type safety.
- SQLite repositories use transactions for atomicity across aggregate boundaries.
- MongoDB repositories leverage native document structure for embedded entities.
- Tests use temporary SQLite files instead of in-memory databases for better reliability.
- Makefile generation includes coverage reporting and quality check targets.

## [Previous Work - Pre-Documentation]

### Added
- Basic model and repository generation for simple entities
- CRUD operations with SQLite and MongoDB driver integration  
- Configuration management with YAML-based service definitions
- Service scaffolding with handler and validator generation
