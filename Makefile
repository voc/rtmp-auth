# parameters
GO = go
GOPATH ?= $(shell go env GOPATH)
PROTOC = PATH=$(PATH):$(GOPATH)/bin protoc
BINARY_NAME = rtmp-auth

PROTO_SRC=storage/storage.proto
PROTO_GENERATED=storage/storage.pb.go

.DEFAULT_GOAL := build

all: build
.PHONY: all

build: proto
	$(GO) build -o $(BINARY_NAME) ./cmd/rtmp-auth
.PHONY: build

auth-consul-migrate: proto
	$(GO) build -o auth-consul-migrate ./cmd/auth-consul-migrate
.PHONY: auth-consul-migrate

proto: $(PROTO_GENERATED)
.PHONY: proto

$(PROTO_GENERATED): $(PROTO_SRC)
	@ if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi
	$(PROTOC) -I=storage/ --go_opt=paths=source_relative --go_out=storage/ $<

clean:
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -f $(PROTO_GENERATED)
.PHONY: clean

lint:
	golangci-lint run ./...

test:
	go test ./...
