# parameters
GOBUILD=HOME=$$(pwd) GOPATH=$$(pwd)/gopath go build
GOCLEAN=HOME=$$(pwd) GOPATH=$$(pwd)/gopath go clean
PROTOC=protoc
STATIK=statik
BINARY_NAME=rtmp-auth
GOPATH=$$(pwd)/gopath
PROTOC_GEN_GO := $(GOPATH)/bin/protoc-gen-go

PROTO_GENERATED=storage/storage.pb.go
STATIK_GENERATED=statik/statik.go
PUBLIC_FILES=$(wildcard public/*)

.DEFAULT_GOAL := build

$(PROTOC_GEN_GO):
	mkdir -p $$(pwd)/gopath
	HOME=$$(pwd) GOPATH=$$(pwd)/gopath go get -u github.com/golang/protobuf/protoc-gen-go

%.pb.go: %.proto
	$(PROTOC) -I=storage/ --go_out=storage/ $<

$(STATIK_GENERATED): $(PUBLIC_FILES)
	mkdir -p $$(pwd)/gopath
	HOME=$$(pwd) GOPATH=$$(pwd)/gopath go get -u github.com/rakyll/statik
	echo "$(PUBLIC_FILES)"
	$(STATIK) -f -src=public/ -dest=.

build: $(PROTO_GENERATED) $(STATIK_GENERATED)
	$(GOBUILD) -o $(BINARY_NAME) -v
.PHONY: build

clean:
	$(GOCLEAN)
	rm -f $(PROTO_GENERATED)
	rm -f $(STATIK_GENERATED)
.PHONY: clean

all: build
.PHONY: all

install: rtmp-auth
	mkdir -p $$(pwd)/debian/$(BINARY_NAME)/usr/local/bin
	install -m 0755 $(BINARY_NAME) $$(pwd)/debian/$(BINARY_NAME)/usr/local/bin 