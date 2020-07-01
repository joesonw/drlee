Dr.LEE - Distributed Lua Execution Environment

# WIP
I'm still doing extensive test on my own (with some real world scenarios). After it's proven to mostly meet the requirements, detailed usage and API docs will be added.

### Async Helpers

```lua
function parallel(list, cb)
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

function series(list, cb)
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
```