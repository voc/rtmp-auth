# rtmp-auth
Simple Auth backend for nginx-rtmp module

  * expiring auth
  * single static binary
  * persists state to simple file
  * WebUI

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

Now all rtmp-publish requests for that application will be authenticated against the daemon.

You can visit localhost:8082 to add streams.

## Build
```
make
```