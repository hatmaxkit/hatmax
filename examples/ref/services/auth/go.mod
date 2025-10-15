module github.com/adrianpk/hatmax-ref/services/auth

go 1.24.0

toolchain go1.24.7

// Dependencies are resolved by go.work workspace
// The workspace includes both the monorepo root and this service

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/google/uuid v1.6.0
)
