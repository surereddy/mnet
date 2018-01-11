MNet
------
Mnet is a collection of superfast networking packages with implementations from ontop of `tcp`, `udp`, and others as planned. It exists to provide a lightweight foundation where other higher level APIs can 
be built on.

## Install

```
go get -v github.com/influx6/mnet/...
```

## Protocols Implemented

### TCP

Mnet provides the [mtcp](./mtcp) package which implements a lightweight tcp server and client implementations with blazing fast data transfers by combining minimal data copy with data buffering techniques. 

Mtcp like all Mnet packages are foundation, in that they lay the necessary foundation to transfer at blazing speed without being too opinionated on how you build on top.


See [MTCP](./mtcp) for more.

### UDP

Mnet provides the [mudp](./mudp) package which implements a lightweight udp server and client implementations with blazing fast data transfers by combining minimal data copy with data buffering techniques. 

Mudp like all Mnet packages are foundation, in that they lay the necessary foundation to transfer at blazing speed without being too opinionated on how you build on top.

### Websockets

Mnet provides the [msocks](./msocks) package which implements a lightweight websocket server and client implementations with blazing fast data transfers by combining minimal data copy with data buffering techniques. 

Msocks like all Mnet packages are foundation, in that they lay the necessary foundation to transfer at blazing speed without being too opinionated on how you build on top.


## Contributions

1. Fork this repository to your own GitHub account and then clone it to your local device
2. Make your changes with clear git commit messages.
3. Create a PR request with detail reason for change.

Do reach out anything, if PR takes time for review. :)

