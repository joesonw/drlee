local websocket = require "websocket"
local rpc = require "rpc"
local env = require "env"
local fs = require "fs"
local http = require "http"

local connections = {}

function handleError(conn, err)
    if err ~= nil then
        print(err)
        closeConn(conn)
        return true
    end
    return false
end

function closeConn(conn)
    connections[conn.id] = nil
    conn:close()
    rpc.broadcast("leave", {
        node = env.node,
        worker_id = env.worker_id,
        id = conn.id,
    })
end

function handleConn(conn)
    conn:read_frame(function(err, body)
        if handleError(err) then return else end
        if body == nil then
            closeConn(conn)
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

        handleConn(conn)
    end)
end

local indexHtml = ""
fs.readfile(__dirname__ .. "/index.html", function(err, content)
    assert(err == nil, "unable to read index.html")
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

local wsServer = websocket.create_server(env.args[2], function(conn)
    conn.id = uuid()
    connections[conn.id] = conn
    rpc.broadcast("join", {
        node = env.node,
        worker_id = env.worker_id,
        id = conn.id,
    })
    handleConn(conn)
end)

wsServer:start(function(err)
    assert(err == nil, "start websocket server error")
    print("websocket server started")
end)

rpc.register("join", function(message, reply)
    for id, conn in pairs(connections) do
        conn:write_frame("user " .. id .." joined from " .. message.worker_id .. "@" .. message.node, function(err) handleError(conn, err) end)
    end
    reply()
end)

rpc.register("leave", function(message, reply)
    for id, conn in pairs(connections) do
        conn:write_frame("user " .. id .." left from " .. message.worker_id .. "@" .. message.node, function(err) handleError(conn, err) end)
    end
    reply()
end)

rpc.register("message", function(message, reply)
    for id, conn in pairs(connections) do
        conn:write_frame("user " .. id .." from " .. message.worker_id .. "@" .. message.node .. "  said: " .. message.message, function(err) handleError(conn, err) end)
    end
    reply()
end)

rpc.start()
print("rpc server started")
