local websocket = require "websocket"
local rpc = require "rpc"
local env = require "env"
local fs = require "fs"
local http = require "http"
local network = require "network"

local connections = {}

function wrap_tcp_conn(conn, delimiter)
    buf = ""
    local size = 64
    local c = {}
    function c:read_frame(cb)
        conn:read(64, function(err, b)
            if err ~= nil then
                return cb(err)
            end
            buf = buf .. b
            while true do
                local idx = string.find(buf, delimiter)
                if idx == nil then
                    break
                end
                local b = buf:sub(0, idx)
                buf = buf:sub(idx)
                cb(nil, b)
            end
        end)
    end
    
    function c:write_frame(frame, cb)
        conn:write(frame .. "\n", cb)
    end

    function c:close(cb)
        conn:close(cb)
    end

    return c 
end

function handle_error(conn, err)
    if err ~= nil then
        print(err)
        close_conn(conn)
        return true
    end
    return false
end

function close_conn(conn)
    connections[conn.id] = nil
    conn:close()
    rpc.broadcast("leave", {
        node = env.node,
        worker_id = env.worker_id,
        id = conn.id,
    })
end

function handle_conn(conn)
    conn:read_frame(function(err, body)
        if handle_error(err) then return else end
        if body == nil then
            return close_conn(conn)
        end
        if body == "" then
            return
        end
        local m = json_decode(body)
        if m.type == "message" then
            rpc.broadcast("message", {
                node = env.node,
                worker_id = env.worker_id,
                id = conn.id,
                message = m.body,
            })
        end

        handle_conn(conn)
    end)
end

local indexHtml = ""
fs.readfile(__dirname__ .. "/index.html", function(err, content)
    assert(err == nil, "unable to read index.html")
    content = string.gsub(content, "{WS_ADDR}", "ws://localhost" .. env.args[2])
    indexHtml = content
end)

local httpServer = http.create_server(env.args[1], function(req, res)
    res:set("Content-Type", "text/html")
    res:set_status(200)
    res:write(indexHtml, function()
        res:finish()
    end)
end)
httpServer:start(function(err)
    assert(err == nil, "start http server error")
    print("http server started")
end)

function handler(conn)
    conn.id = uuid()
    connections[conn.id] = conn
    rpc.broadcast("join", {
        node = env.node,
        worker_id = env.worker_id,
        id = conn.id,
    })
    handle_conn(conn)
end

local wsServer = websocket.create_server(env.args[2], handler)

wsServer:start(function(err)
    assert(err == nil, "start websocket server error")
    print("websocket server started")
end)

local tcpServer = network.create_server("tcp", env.args[3], function(conn)
    handler(wrap_tcp_conn(conn, "\n"))
end)
tcpServer:start(function(err)
    assert(err == nil, "start tcp server error")
    print("tcp server started")
end)

rpc.register("join", function(message, reply)
    for id, conn in pairs(connections) do
        conn:write_frame("user " .. message.id .." joined from " .. message.worker_id .. "@" .. message.node, function(err) handle_error(conn, err) end)
    end
    reply()
end)

rpc.register("leave", function(message, reply)
    for id, conn in pairs(connections) do
        conn:write_frame("user " .. message.id .." left from " .. message.worker_id .. "@" .. message.node, function(err) handle_error(conn, err) end)
    end
    reply()
end)

rpc.register("message", function(message, reply)
    for id, conn in pairs(connections) do
        conn:write_frame("user " .. message.id .." from " .. message.worker_id .. "@" .. message.node .. "  said: " .. message.message, function(err) handle_error(conn, err) end)
    end
    reply()
end)

rpc.start()
print("rpc server started")
