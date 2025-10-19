module github.com/adrianpk/hatmax-ref/services/admin

go 1.24.7

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/google/uuid v1.6.0
	github.com/knadh/koanf/parsers/yaml v1.1.0
	github.com/knadh/koanf/providers/env v1.1.0
	github.com/knadh/koanf/providers/posflag v1.0.1
	github.com/knadh/koanf/providers/rawbytes v1.0.0
	github.com/knadh/koanf/v2 v2.3.0
	github.com/spf13/pflag v1.0.10
)

require (
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.3 // indirect
)

replace github.com/hatmax/hatmax/pkg/lib/auth => ../../pkg/lib/auth

replace github.com/hatmax/hatmax/pkg/lib/core => ../../pkg/lib/core
