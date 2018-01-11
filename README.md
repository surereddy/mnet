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

Mnet current has no implementation but is planned.

### Websockets

Mnet current has no implementation but is planned.



## Contributions

Contributors are welcome to file issue tickets and provide PRs in accordance to the following [Guidelines](./contrib.md)