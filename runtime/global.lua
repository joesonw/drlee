string = require "string"
math = require "math"
coroutine = require "coroutine"

function parallel_callback(list, cb)
    local total = table.getn(list)
    if total == 0 then
        cb(nil, {})
        return
    end
    local resolved = false
    local count = 0
    local resultList = {}
    local wrappedCallback = function(err, result)
        if resolved then
            return
        end
        if err ~= nil then
            resolved = true
            cb(err)
            return
        end
        table.insert(resultList, result)
        count = count + 1
        if count == total then
            cb(nil, resultList)
        end
    end
    for i, f in ipairs(list) do
        f(wrappedCallback)
    end
end

function series_callback(list, cb)
    local total = table.getn(list)
    if total == 0 then
        cb()
        return
    end
    local index = 1
    local wrappedCallback
    wrappedCallback = function(err, result)
        if err ~= nil then
            cb(err)
            return
        end
        index = index + 1
        if index > total then
            cb(nil, result)
            return
        end
        list[index](wrappedCallback)
    end

    list[1](wrappedCallback)
end

function readall(reader, cb)
    local size = 512
    local buf = ""
    local more
    more = function(err, res, n)
        if err ~= nil then
            return cb(err)
        end
        if n < size then
            res = string.sub(res, 0, n)
        end
        buf = buf .. res
        if n <= 0 or n < size then
            return cb(nil, buf)
        end
        reader:read(size, more)
    end
    reader:read(size, more)
end


-- http://lua-users.org/wiki/SimpleLuaClasses
function class(base, init)
    local c = {} -- a new class instance
    if not init and type(base) == 'function' then
        init = base
        base = nil
    elseif type(base) == 'table' then
        -- our new class is a shallow copy of the base class!
        for i, v in pairs(base) do
            c[i] = v
        end
        c._base = base
    end
    -- the class will be the metatable for all its objects,
    -- and they will look up their methods in it.
    c.__index = c

    -- expose a constructor which can be called by <classname>(<args>)
    local mt = {}
    mt.__call = function(class_tbl, ...)
        local obj = {}
        setmetatable(obj, c)
        if init then
            init(obj, ...)
        else
            -- make sure that any stuff from the base class is initialized!
            if base and base.init then
                base.init(obj, ...)
            end
        end
        return obj
    end
    c.init = init
    c.is = function(self, klass)
        local m = getmetatable(self)
        while m do
            if m == klass then return true end
            m = m._base
        end
        return false
    end
    setmetatable(c, mt)
    return c
end

