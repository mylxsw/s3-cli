GitCommit := $(shell git rev-parse HEAD)
CompileTime := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DIR := $(shell pwd)
LDFLAGS := -s -w -X main.GitCommit=$(GitCommit) -X main.CompileTime=$(CompileTime) -X main.Debug=true

build:
	go build -race -ldflags "$(LDFLAGS)" -o build/debug/s3-cli main.go

release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/s3-cli-linux main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/s3-cli-darwin main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o build/release/s3-cli-darwin-m1 main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/s3-cli.exe main.go

.PHONY: build release
