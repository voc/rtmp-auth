# rtmp-auth
Simple auth backend for nginx-rtmp module written in go.

## Features
  * Expiring auth
  * Single static binary
  * Persists state to simple file (no database required)
  * Web-UI with subpath support (planned)

In the future I might also add support for removing active streams when they expire.

## Usage

Add on_publish/on_publish_done callbacks to your nginx-rtmp config
```
application myrtmp {
  live on;
  meta copy;

  hls off;

  allow publish all;
  allow play all;

  # add this for authentication
  on_publish http://127.0.0.1:8080/publish;
  on_publish_done http://127.0.0.1:8080/unpublish;
}
```

Then start the daemon
```
./rtmp-auth -app "myrtmp" -apiAddr "localhost:8000" -frontendAddr "localhost:8082"
```

**Note: You will need to set the -insecure flag when testing over http.**

After you reload your nginx with ```systemctl reload nginx``` or similar your
rtmp publish-requests will be authenticated against the daemon.
You can then visit localhost:8082 to add streams.

For production usage you will want to deploy the frontend behind a Reverse-Proxy with TLS-support like the nginx you already have.

## Build Dependencies
  * protoc with go-support
  * statik ```go get github.com/rakyll/statik```

## Runtime Dependencies
None apart from nginx-rtmp obviously

## Build
```
make
```
