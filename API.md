* [Core](#core)
 * [File System](#file-system)
    * [fs.flags](#fsflags)
    * [fs_open(path, flag?, mode?, cb?)](#fs_openpath-flag-mode-cb)
    * [fs.remove(path, cb?)](#fsremovepath-cb)
    * [fs.remove_all(path, cb?)](#fsremove_allpath-cb)
    * [fs.stat(path, cb?)](#fsstatpath-cb)
    * [fs.read_dir(path, cb?)](#fsread_dirpath-cb)
    * [fs.mkdir(path, cb?)](#fsmkdirpath-cb)
    * [fs.mkdir_all(path, cb?)](#fsmkdir_allpath-cb)
    * [fs.readfile(path, cb?)](#fsreadfilepath-cb)
    * [file:read(amount, cb?)](#filereadamount-cb)
    * [file:write(data, cb?)](#filewritedata-cb)
    * [file:close(cb?)](#fileclosecb)
 * [HTTP](#http)
    * [http.get(path, options?, cb?)](#httpgetpath-options-cb)
    * [http.post(path, options?, cb?)](#httppostpath-options-cb)
    * [http.put(path, options?, cb?)](#httpputpath-options-cb)
    * [http.delete(path, options?, cb?)](#httpdeletepath-options-cb)
    * [http.patch(path, options?, cb?)](#httppatchpath-options-cb)
    * [http.request(method, path, options?, cb?)](#httprequestmethod-path-options-cb)
       * [httpResponse](#httpresponse)
          * [httpResponse.headers](#httpresponseheaders)
          * [httpResponse.status_code](#httpresponsestatus_code)
          * [httpResponse.status](#httpresponsestatus)
          * [httpResponse:read(n, cb?)](#httpresponsereadn-cb)
          * [httpResponse:close(cb?)](#httpresponseclosecb)
    * [http.create_server(addr, handler?)](#httpcreate_serveraddr-handler)
       * [httpRequest](#httprequest)
          * [httpRequest.headers](#httprequestheaders)
          * [httpRequest.request_uri](#httprequestrequest_uri)
          * [httpRequest.url](#httprequesturl)
          * [httpRequest.method](#httprequestmethod)
          * [httpRequest:read(n, cb?)](#httprequestreadn-cb)
          * [httpRequest:close(cb?)](#httprequestclosecb)
       * [httpResponseWriter](#httpresponsewriter)
          * [httpResponseWriter:set_status(code)](#httpresponsewriterset_statuscode)
          * [httpResponseWriter:set(name, value)](#httpresponsewritersetname-value)
          * [httpResponseWriter:get(name)](#httpresponsewritergetname)
          * [httpResponseWriter:write(body, cb)](#httpresponsewriterwritebody-cb)
          * [httpResponseWriter:finish(cb)](#httpresponsewriterfinishcb)
    * [httpServer:start(cb?)](#httpserverstartcb)
    * [httpServer:stop()](#httpserverstop)
 * [Network](#network)
    * [network.dial(network, addr, cb)](#networkdialnetwork-addr-cb)
    * [network.create_server(network, addr, handler)](#networkcreate_servernetwork-addr-handler)
    * [conn](#conn)
       * [conn:write(body, cb?)](#connwritebody-cb)
       * [conn:read(size, cb?)](#connreadsize-cb)
       * [conn:close(cb?)](#connclosecb)
 * [websocket](#websocket)
    * [websocket.dial(addr, cb)](#websocketdialaddr-cb)
    * [network.create_server(addr, handler)](#networkcreate_serveraddr-handler)
    * [conn](#conn-1)
       * [conn:write_frame(frame, cb?)](#connwrite_frameframe-cb)
       * [conn:read_frame(cb?)](#connread_framecb)
       * [conn:close(cb?)](#connclosecb-1)
 * [redis](#redis)
    * [redis:do(..., cb)](#redisdo-cb)
 * [rpc](#rpc)
    * [rpc.register(name, handler)](#rpcregistername-handler)
    * [rpc.call(name, message, cb?)](#rpccallname-message-cb)
    * [rpc.broadcast(name, message, cb?)](#rpcbroadcastname-message-cb)
 * [sql](#sql)
    * [sql.open(uri, cb)](#sqlopenuri-cb)
    * [conn](#conn-2)
       * [conn:close(cb)](#connclosecb-2)
       * [conn:query(query, ..., cb)](#connqueryquery--cb)
       * [conn:exec(query, ..., cb)](#connexecquery--cb)
       * [conn:begin(cb)](#connbegincb)
    * [tx](#tx)
       * [tx:query(query, ..., cb)](#txqueryquery--cb)
       * [tx:exec(query, ..., cb)](#txexecquery--cb)
       * [tx:commit(cb)](#txcommitcb)
       * [tx:rollback(cb)](#txrollbackcb)
 * [Time](#time)
    * [time.now()](#timenow)
    * [time.timeout(ms, cb)](#timetimeoutms-cb)
    * [time.tick(ms)](#timetickms)
       * [ticker:next_tick(cb)](#tickernext_tickcb)
       * [ticker:stop()](#tickerstop)
 * [Env](#env)
    * [env.node](#envnode)
    * [env.worker_id](#envworker_id)
    * [env.worker_dir](#envworker_dir)
    * [args](#args)
 * [Log](#log)
    * [log.debug(...)](#logdebug)
    * [log.info(...)](#loginfo)
    * [log.warn(...)](#logwarn)
    * [log.error(...)](#logerror)
    * [log.fatal(...)](#logfatal)
* [Globals](#globals)
    * [parallel_callback(list, cb?)](#parallel_callbacklist-cb)
    * [series_callback(list, cb?)](#series_callbacklist-cb)
    * [readall(reader, cb?)](#readallreader-cb)
    * [uuid()](#uuid)
    * [bit_or(a, b)](#bit_ora-b)
    * [bit_and(a, b)](#bit_anda-b)
    * [bit_xor(a, b)](#bit_xora-b)
    * [__dirname__](#__dirname__)
    * [json_encode(value)](#json_encodevalue)
    * [json_decode(value)](#json_decodevalue)





## Core

> cb means Callbacks: a function with sinagure `function(err, result) end`

### File System

```lua
local fs = require "fs"
```

#### fs.flags
> flags is a table of constants for file flags, one of READ_ONLY, WRITE_ONLY or READ_WRITE is required when specifying flags
 * fs.flags.READ_ONLY
 * fs.flags.WRITE_ONLY
 * fs.flags.READ_WRITE
 * fs.flags.APPEND
 * fs.flags.CREATE
 * fs.flags.EXCL
 * fs.flags.SYNC
 * fs.flags.TRUNC

#### fs_open(path, flag?, mode?, cb?)

`fs.open(path, cb?)`
> opens a file descriptor at path

`fs.open(path, flag, cb?)`
> opens a file descriptor at the path with flag

`fs.open(path, flag, mode, cb?)`
> opens a file descriptor at the path with flag and mode

#### fs.remove(path, cb?)
> removes a file at path

#### fs.remove_all(path, cb?)
> removes all at path

#### fs.stat(path, cb?)
> get file stat at path

Stat table

|   key    |   type  | description  |
|----------|---------|--------------|
| name     |  string | file name    |
| isdir    |  bool   | is directory |
| mode     | number  | file mode    |
| size     | number  | file size    |
| timestamp| Timestamp | modification time |

#### fs.read_dir(path, cb?)
> get list of file stats at path

#### fs.mkdir(path, cb?)
> mkdir

#### fs.mkdir_all(path, cb?)
> mkdir -p

#### fs.readfile(path, cb?)
> this function will stat a file first to get the file size and then read all datta.

#### file:read(amount, cb?)
> read give amount of data

#### file:write(data, cb?)
> write data to file

#### file:close(cb?)
> close file handler


### HTTP

```lua
local http = require "http"
```

#### http.get(path, options?, cb?)
#### http.post(path, options?, cb?)
#### http.put(path, options?, cb?)
#### http.delete(path, options?, cb?)
#### http.patch(path, options?, cb?)
#### http.request(method, path, options?, cb?)
> all takes the same arguments

```lua
http.request(METHOD, URL, { headers={} }, function(err, res)
    assert(err == nil, "handle http error")
    readall(res, function(err, body)
        assert(err == nil, "read http response")
        print(body)
        res:close()
    end)
end)
```

##### httpResponse
> response returned by http request

###### httpResponse.headers
> table of headers
###### httpResponse.status_code
> status code
###### httpResponse.status
> status text
###### httpResponse:read(n, cb?)
> read body
###### httpResponse:close(cb?)
> close response, release resource


options 

|    key    |   type  | description |
|-----------|---------|-------------|
| body      | string  | request body|
| headers   | table   | request headers |

#### http.create_server(addr, handler?)
> create a http server
```lua
local server = http.create_server(":80", function(req, res) end)
```

##### httpRequest
> http request received by server
###### httpRequest.headers
> table of headers
###### httpRequest.request_uri
> request uri 
###### httpRequest.url
> request url 
###### httpRequest.method
> request method
###### httpRequest:read(n, cb?)
> read body
###### httpRequest:close(cb?)
> close request, release resource

##### httpResponseWriter
> response writer used by server
###### httpResponseWriter:set_status(code)
> set status 
###### httpResponseWriter:set(name, value)
> set header 
###### httpResponseWriter:get(name)
> get header 
###### httpResponseWriter:write(body, cb)
> write body
###### httpResponseWriter:finish(cb)
> finish response writer, release resources


#### httpServer:start(cb?)
> start the http server
```lua
server:start(function(err)
    assert(err == nil, "start server error")
    print("server started")
end)
```

#### httpServer:stop()
> stop the http server

### Network

#### network.dial(network, addr, cb)
`function cb(err, conn)`
> network can be either `tcp` or `udp`

#### network.create_server(network, addr, handler)
`function handler(conn)`


#### conn 
##### conn:write(body, cb?)
##### conn:read(size, cb?)
##### conn:close(cb?)

### websocket 

#### websocket.dial(addr, cb)
`function cb(err, conn)`
> network can be either `tcp` or `udp`

#### network.create_server(addr, handler)
`function handler(conn)`

#### conn 
##### conn:write_frame(frame, cb?)
##### conn:read_frame(cb?)
##### conn:close(cb?)

### redis
#### redis:do(..., cb)
```lua
redis.do('get', 'key', function(err, value) end)
```

### rpc
#### rpc.register(name, handler)
`function handler(message, reply)`

`function reply(err, result)`

#### rpc.call(name, message, cb?)

#### rpc.broadcast(name, message, cb?)

### sql

#### sql.open(uri, cb)
`function cb(err, conn)`

#### conn

##### conn:close(cb)

##### conn:query(query, ..., cb)
```lua
conn:query("SELECT id, name FROM users WHERE id = ?", 1, function (err, users)
end)
```

##### conn:exec(query, ..., cb)
```lua
conn:query("INSERT INTO users(id, name) VALUES (?, ?)", 2, "hello", function (err, res)
    print(res.last_inserted_id)
    print(res.rows_affected)
end)
```

##### conn:begin(cb)
`function cb(err, tX)`

#### tx

##### tx:query(query, ..., cb)
same as [conn:query(query, ..., cb)](#connqueryquery--cb)

##### tx:exec(query, ..., cb)
same as [conn:exec(query, ..., cb)](#connexecquery--cb)

##### tx:commit(cb)

##### tx:rollback(cb)

### Time

#### time.now()
> returns a time obejct 
```lua
local t = time.now()
print(t.year)
print(t.month)
print(t.day)
print(t.weekday)
print(t.hour)
print(t.minute)
print(t.second)
print(t.millisecond)
print(t.milliunix) -- unix epoch in milliseconds
print(t:format("2006-01-02T15:04:05.000Z07:00"))
```

#### time.timeout(ms, cb)
```lua
time.timeout(1000, function()
    print("1 second later")
end)
```

#### time.tick(ms)
```lua
local ticker = time.tick(1000)
```

##### ticker:next_tick(cb)
```lua
ticker:next_tick(function(t)
    print("1 seoncd later: " .. t:format("2006-01-02T15:04:05.000Z07:00"))
end)
```

##### ticker:stop()
> stops the ticker

### Env

#### env.node
> gossip node name

#### env.worker_id
> current worker id

#### env.worker_dir
> server cwd

#### args
> args from config `script-args` field 

### Log

#### log.debug(...)
#### log.info(...)
#### log.warn(...)
#### log.error(...)
#### log.fatal(...)

## Globals 

#### parallel_callback(list, cb?)
```lua
local time = require("time")
parallel_callback({
    function(cb)
        time:timeout(1000, cb)
    end,
    function(cb)
        time:timeout(1000, cb)
    end,
    function(cb)
        time:timeout(1000, cb)
    end,
}, function(err, res)
    print "all async operations are done in 1 second"
end)
```

#### series_callback(list, cb?)
 ```lua
 local time = require("time")
 series_callback({
     function(cb)
         time:timeout(1000, cb)
     end,
     function(cb)
         time:timeout(1000, cb)
     end,
     function(cb)
         time:timeout(1000, cb)
     end,
 }, function(err, res)
     print "all async operations are done in sequential order in 3 seconds"
 end)
 ```

#### readall(reader, cb?)
>read all data (until reads 0 or EOF) of a read (which has `read(n, cb)`)

```lua
read_all(req, function(err, body) end)
```

#### uuid()
> generates a UUID

#### bit_or(a, b)
> bitwise or

#### bit_and(a, b)
> bitwise and

#### bit_xor(a, b)
> bitwise xor

#### \_\_dirname\_\_
> directory of loaded lua script

#### json_encode(value)
> encode value to json string

#### json_decode(value)
> decode string to value 

