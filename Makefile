.PHONY: test swagger fmt lint build run

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


