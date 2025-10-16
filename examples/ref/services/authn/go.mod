module github.com/adrianpk/hatmax-ref/services/authn

go 1.23

// Dependencies are resolved by go.work workspace
// The workspace includes both the monorepo root and this service

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/google/uuid v1.6.0
	github.com/knadh/koanf/parsers/yaml v1.1.0
	github.com/knadh/koanf/providers/env v1.1.0
	github.com/knadh/koanf/providers/posflag v1.0.1
	github.com/knadh/koanf/providers/rawbytes v1.0.0
	github.com/knadh/koanf/v2 v2.3.0
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/spf13/pflag v1.0.10
	go.mongodb.org/mongo-driver v1.17.4
	golang.org/x/crypto v0.43.0
)
