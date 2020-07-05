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
    local wrappedCallback = function (err, result)
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
    if total  == 0 then
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


