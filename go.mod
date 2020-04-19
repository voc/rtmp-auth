module github.com/voc/rtmp-auth

go 1.13

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/golang/protobuf v1.4.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/csrf v1.6.2
	github.com/gorilla/mux v1.7.4
	github.com/rakyll/statik v0.1.7
)
