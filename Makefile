# -------- Settings --------
TAGS      ?= testutil
RACE      ?= -race
COUNT     ?= 1
TIMEOUT   ?= 5m
BIN ?= easygodocs

PKGS_ALL := $(shell go list ./...)
PKGS     := $(filter-out %/mocks %/mocks/% %/mock %/mock/% %/minimock %/minimock/%,$(PKGS_ALL))
COVERPKG := $(shell printf "%s\n" $(PKGS) | paste -sd, -)

# -------- Targets --------
.PHONY: build run fmt vet lint test cover cover-html clean tools vuln

build:
	mkdir -p bin
	go build -o bin/$(BIN) ./cmd/server
	go build -o bin/$(BIN) ./cmd/seedadmin

run:
	go run ./cmd

fmt:
	gofumpt -l -w .
	goimports -w .

vet:
	go vet -tags '$(TAGS)' ./...

lint:
	golangci-lint run --build-tags "$(TAGS)" ./...

vuln:
	govulncheck ./...

test:
	go test $(RACE) -tags $(TAGS) -timeout $(TIMEOUT) -count=$(COUNT) ./...

cover:
	go test $(RACE) -tags $(TAGS) -timeout $(TIMEOUT) -count=$(COUNT) \
	  -covermode=atomic -coverpkg=$(COVERPKG) -coverprofile=coverage.out $(PKGS)
	go tool cover -func=coverage.out

cover-html: cover
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

generate:
	go generate ./...
	swag init -g cmd/server/main.go

tools:
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

clean:
	rm -f coverage.out coverage.html

print-pkgs:
	@echo PKGS=$(PKGS)
	@echo COVERPKG=$(COVERPKG)