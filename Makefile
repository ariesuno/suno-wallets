.PHONY: test swagger fmt lint build run smoke smoke-dry smoke-reset smoke-inc

test:
	go test ./... -count=1 -v

swagger:
	@which swag >/dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@latest
	$(shell go env GOPATH)/bin/swag init -g src/main.go -o docs

fmt:
	@which goimports >/dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
	gofmt -w .
	goimports -w .

lint:
	@which golangci-lint >/dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest
	$(shell go env GOPATH)/bin/golangci-lint run ./...

build:
	go build ./...

run:
	go run ./src


smoke:
	go run ./cmd/e2e/phase1_smoke_runner.go

smoke-dry:
	ALLOW_DESTRUCTIVE_RESET=false go run ./cmd/e2e/phase1_smoke_runner.go

smoke-reset:
	ALLOW_DESTRUCTIVE_RESET=true go run ./cmd/e2e/phase1_smoke_runner.go

smoke-inc:
	go run ./cmd/e2e/phase1_smoke_runner.go --allow-reset=false


