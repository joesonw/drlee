local _http = require "_http"
local http = {}

http.request = _http.request

http.get = function(...)
    _http.request("GET", ...)
end

http.post = function(...)
    _http.request("POST", ...)
end

http.put = function(...)
    _http.request("PUT", ...)
end

http.delete = function(...)
    _http.request("DELETE", ...)
end

http.patch = function(...)
    _http.request("PATCH", ...)
end

http.create_server = _http.create_server
return http
