Dr.LEE - Distributed Lua Execution Environment

# WIP
I'm still doing extensive test on my own (with some real world scenarios). After it's proven to mostly meet the requirements, detailed usage and API docs will be added.


# LUA API

* [Core](#core)
  * [File System](#file-system)
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
* [Helpers](#helpers)
     * [parallel_callback(list, cb?)](#parallel_callbacklist-cb)
     * [series_callback(list, cb?)](#series_callbacklist-cb)
     * [readall(reader, cb?)](#readallreader-cb)



## Core

> cb means Callbacks: a function with sinagure `function(err, result) end`

### File System

```lua
local fs = require "fs"
```

#### fs_open(path, flag?, mode?, cb?)

`fs.open(path, cb?)`
> opens a file descriptor at path

`fs.open(path, flag, cb?)`
> opens a file descriptor at the path with flag

Flags

|      flag     | value   |
|:-------------:|:-------:|
|  READ ONLY   |  0x0     | 
| WRITE ONLY   |  0x1     |
| READ WRITE   |  0x2     |
|   APPEND     |  0x8     |
|   CREATE     |  0x200   |
|    EXCL      |  0x800   |
|    SYNC      |  0x80    |
|    TRUNC     |  0x400   |

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

###

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

## Helpers

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
