# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2025-10-15] - Authentication Service Implementation

### Added
**Complete Authentication Service with Pure Domain Library**: Full-featured authentication service with cryptographic functions and clean architecture separation.
- **Pure Auth Library**: Independent authentication library at `pkg/lib/auth` with cryptographic functions, password hashing, email validation, and user permissions
- **Auth Service Generation**: Automatic generation of complete auth service with signup, signin, signout, and user management endpoints
- **Workspace Integration**: Seamless integration with Go workspace for multi-module development
- **SQLite Repository**: Production-ready user repository with proper password hashing using Argon2
- **Domain Separation**: Clean separation between pure domain models and service persistence layer with explicit conversions
- **Test Scripts**: Comprehensive curl test scripts with unique email generation and consistent password management

### Technical Details
- Authentication service automatically generated as part of main generation flow
- Pure cryptographic functions using industry-standard algorithms (Argon2, HMAC-SHA256)
- Complete email validation and normalization with lookup hash generation for privacy
- User status validation and permissions system foundation
- Clean architecture with domain models in pure library and service-specific models for persistence

## [2025-10-13] - HATEOAS and Enhanced Code Generation

### Added
**HATEOAS-Enabled Handlers with Enhanced Logging**: Improved REST API generation with hypermedia support and better developer experience.
- **HATEOAS Support**: Aggregate handlers now include hypermedia links for REST API discoverability
- **Enhanced Logging**: Improved generator output with clear progress indicators and structured logging
- **Code Generation Improvements**: Better template processing and error handling in the generation pipeline

## [2025-10-13] - Core Library Architecture

### Changed
**Centralize core library with Go workspace integration**: Move core library to dedicated module with proper workspace support.
- Core library now exists as independent Go module at `pkg/lib/core`
- Go workspace handles multi-module development seamlessly
- Services import core library through workspace-resolved paths

## [2025-10-13] - Aggregate Repository Pattern

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
