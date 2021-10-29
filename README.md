# rtmp-auth
Simple stream auth backend for:
  - nginx-rtmp
  - srtrelay
  - srs

## Features
  * Expiring auth
  * Single static binary
  * Persists state to simple file (no database required)
  * Web-UI with subpath support

In the future I might also add support for removing active streams when they expire.

## Build Dependencies
  * protoc with go-support
  * statik ```go get github.com/rakyll/statik```

## Usage
Build the daemon with
```
make
```

Then start it
```bash
./rtmp-auth -app "myrtmp" -apiAddr "localhost:8000" -frontendAddr "localhost:8082"
```
It will now authenticate streams for the rtmp-app "myrtmp" (the app is the "directory" part of a rtmp url) like ```rtmp://<host>/<app>/<stream>```

### Nginx-RTMP
Add on_publish/on_publish_done callbacks to your nginx-rtmp config
```nginx
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

### srtrelay
Change the auth to "http" in your srtrelay config and add the api url:
```toml
[auth]
type = "http"

[auth.http]
url = "http://localhost:8080/publish"
```

srtrelay doesn't currently support unpublish.

### SRS
Add the http_hooks config inside your srs vhost config:
```nginx
vhost __defaultVhost__ {
    ...
    http_hooks {
        enabled         on;
        on_publish      http://172.17.0.1:8080/publish;
        on_unpublish    http://172.17.0.1:8080/unpublish;
    }
    ...
}
```

### WebUI
**Note: You will need to set the -insecure flag when testing over http.**

After reloading your nginx/srs the rtmp publish-requests will be authenticated against the daemon.
You can visit http://localhost:8082 to add streams.

For production usage you will want to deploy the frontend behind a Reverse-Proxy with TLS-support like nginx.

### Publish a stream
Now that you have set up your software you can start publishing streams

```bash
# publish without auth
ffmpeg -i test.mp4 -c copy -f flv rtmp://server/app/stream

# publish with auth
ffmpeg -i test.mp4 -c copy -f flv rtmp://server/app/stream?auth=foobar2342

```

