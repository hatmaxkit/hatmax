GO_TEST_COMMAND ?= go test ./...

.PHONY: check lint generate format run

check:
	go build ./...
	$(GO_TEST_COMMAND)
	$(MAKE) lint

lint:
	golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5

generate:
	sqlc generate
	go mod tidy

format:
	gofmt -w .

run:
	go run .
