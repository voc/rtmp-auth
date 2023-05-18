# parameters
GOPATH = $(shell pwd)/gopath
GOCACHE = $(shell pwd)/gocache
GO = GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) GO111MODULE=on $(shell pwd)/go/bin/go
STATIK = $(GOPATH)/bin/statik
PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC = PATH=$(PATH):/usr/bin protoc
BINARY_NAME = rtmp-auth

PROTO_GENERATED=storage/storage.pb.go
STATIK_GENERATED=statik/statik.go
PUBLIC_FILES=$(wildcard public/*)

.DEFAULT_GOAL := build

$(PROTOC_GEN_GO):
	mkdir -p $(GOPATH)
	$(GO) get -u github.com/golang/protobuf/protoc-gen-go
	$(GO) build -o $(PROTOC_GEN_GO)

storage/storage.pb.go: storage/storage.proto | $(PROTOC_GEN_GO)
	@ if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi
	$(PROTOC) -I=storage/ --go_opt=paths=source_relative --go_out=storage/ $<

$(STATIK_GENERATED): $(PUBLIC_FILES)
	mkdir -p $(GOPATH)
	$(GO) get -u github.com/rakyll/statik
	$(GO) build -o ./gopath/bin/$(BINARY_NAME)
	echo "$(PUBLIC_FILES)"
	$(STATIK) -f -src=public/ -dest=.

build: $(PROTO_GENERATED) $(STATIK_GENERATED)
	$(GO) build -o $(BINARY_NAME) ./cmd/rtmp-auth
.PHONY: build

clean:
	$(GO) clean
	rm -rf ./gopath
	rm -rf ./gocache
	rm -f $(BINARY_NAME)
	rm -f $(PROTO_GENERATED)
	rm -f $(STATIK_GENERATED)
.PHONY: clean

all: build
.PHONY: all

install: rtmp-auth
	mkdir -p $$(pwd)/debian/$(BINARY_NAME)/usr/bin
	install -m 0755 $(BINARY_NAME) $$(pwd)/debian/$(BINARY_NAME)/usr/bin 