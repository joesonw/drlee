Dr.LEE - Distributed Lua Execution Environment

![Go](https://github.com/joesonw/drlee/workflows/Go/badge.svg)

# WIP
I'm still doing extensive test on my own (with some real world scenarios). After it's proven to mostly meet the requirements, detailed usage and API docs will be added.

# Architecture

 * Peer nodes receives metadata(registered rpc methods, rpc port) through GOSSIP protocol.
 * RPC calls are transported through direct GRPC connection between/among each nodes.
 * Each node has a inbox/reply queue, thus unhandled messages are stored on hdd (nsq-diskqueue).
 
# Example

### Chat
[source](https://github.com/joesonw/drlee/tree/master/example/chat)
```bash
drlee server a.yaml server.lua
drlee server b.yaml server.lua --join localhost:4100
```
now you can open two tabs each at [http://localhost:8080](http://localhost:8080) and [http://localhost:8180](http://localhost:8180) to send messages to each other.

you can also do `nc localhost 8082` or `nc localhost 8182` to join the same chat room, messages are delimited by `EOL`.

notice you were connecting to two different nodes, but the built in cross-node RPC came with `drlee` kicked in to help to easily develop a distributed service like it's on one machine. 


# Installation
There are no releases currently, to get the latest binary: visit [https://bintray.com/joesonw/drlee/drlee](https://bintray.com/joesonw/drlee/drlee) and download a binary for you platform

# LUA API
see [API.md](https://github.com/joesonw/drlee/tree/master/API.md)

# Notice
Though all asynchronous methods are in callback style.
However, in Lua, this is not a problem, one can easily wrap them in co-routine style.
For the sake of code simplicity and clean architecture, callback are how Dr.LEE implements asynchronous methods.
