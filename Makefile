# parameters
GO = go
GOPATH ?= $(shell go env GOPATH)
PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
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

$(PROTOC_GEN_GO): go.sum
	$(GO) install -buildvcs=false google.golang.org/protobuf/cmd/protoc-gen-go

proto: $(PROTO_GENERATED)
.PHONY: proto

$(PROTO_GENERATED): $(PROTO_SRC) | $(PROTOC_GEN_GO)
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

