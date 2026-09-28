.PHONY: check lint generate format run

check:
	go test ./...

lint:
	golangci-lint run

generate:
	sqlc generate

format:
	gofmt -w .

run:
	go run .
