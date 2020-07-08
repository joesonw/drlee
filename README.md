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

# BenchmarkS
[http benchmark test](https://github.com/joesonw/drlee/tree/master/benchmarks/http)
```
 go test -run=Bench -bench=. ./benchmarks/http/...
goos: darwin
goarch: amd64
pkg: github.com/joesonw/drlee/benchmarks/http
BenchmarkLua-12                               	   10236	    107047 ns/op
BenchmarkLuaParallel4-12                      	   10690	    100500 ns/op
BenchmarkLuaSleep-12                          	   10948	    100196 ns/op
BenchmarkLuaSleepParallel4-12                 	   10430	    102018 ns/op
BenchmarkLuaConcurrent4-12                    	   17148	    131602 ns/op
BenchmarkLuaParallel4Concurrent4-12           	    8458	    210623 ns/op
BenchmarkLuaSleepConcurrent4-12               	    3180	    321741 ns/op
BenchmarkLuaSleepParallel4Concurrent4-12      	    2889	    347221 ns/op
BenchmarkPlain-12                             	     690	   1686879 ns/op
BenchmarkPlainParallel4-12                    	     595	   2042301 ns/op
BenchmarkPlainSleep-12                        	       9	 119222153 ns/op
BenchmarkPlainSleepParallel4-12               	       9	 120673633 ns/op
BenchmarkPlainConcurrent4-12                  	     914	   1375349 ns/op
BenchmarkPlainParallel4Concurrent4-12         	     690	   1680510 ns/op
BenchmarkPlainSleepConcurrent4-12             	      38	  32632435 ns/op
BenchmarkPlainSleepParallel4Concurrent4-12    	      31	  32852115 ns/op
PASS
ok  	github.com/joesonw/drlee/benchmarks/http	40.411s
```

`BenchmarkPlain` is in plain go, `BenchmarkLua` ran with lua.

> Benchmarks contains `Sleep` sleeps 100ms before writing back to client.
> Concurrency means request concurrency 


# Installation
There are no releases currently, to get the latest binary: visit [https://bintray.com/joesonw/drlee/drlee](https://bintray.com/joesonw/drlee/drlee) and download a binary for you platform

# LUA API
see [API.md](https://github.com/joesonw/drlee/tree/master/API.md)

# Notice
Though all asynchronous methods are in callback style.
However, in Lua, this is not a problem, one can easily wrap them in co-routine style.
For the sake of code simplicity and clean architecture, callback are how Dr.LEE implements asynchronous methods.
