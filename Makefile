# -------- Settings --------
TAGS      ?= testutil
RACE      ?= -race
COUNT     ?= 1
TIMEOUT   ?= 5m
BIN_DIR   ?= bin

PKGS_ALL := $(shell go list ./...)
PKGS     := $(filter-out %/mocks %/mocks/% %/mock %/mock/% %/minimock %/minimock/%,$(PKGS_ALL))
COVERPKG := $(shell printf "%s\n" $(PKGS) | paste -sd, -)

# -------- Targets --------
.PHONY: build run seed-admin fmt vet lint test cover cover-html generate clean tools vuln print-pkgs

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/server ./cmd/server
	go build -o $(BIN_DIR)/seedadmin ./cmd/seedadmin

run:
	go run ./cmd/server

seed-admin:
	go run ./cmd/seedadmin

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
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
	go install golang.org/x/vuln/cmd/govulncheck@latest

clean:
	rm -f coverage.out coverage.html
	rm -rf $(BIN_DIR)

print-pkgs:
	@echo PKGS=$(PKGS)
	@echo COVERPKG=$(COVERPKG)