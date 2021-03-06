# parameters
GOBUILD=go build
GOCLEAN=go clean
PROTOC=protoc
STATIK=statik
BINARY_NAME=rtmp-auth

PROTO_GENERATED=storage/storage.pb.go
STATIK_GENERATED=statik/statik.go
PUBLIC_FILES=$(wildcard public/*)

.DEFAULT_GOAL := build

%.pb.go: %.proto
	$(PROTOC) -I=storage/ --go_out=storage/ $<

$(STATIK_GENERATED): $(PUBLIC_FILES)
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
