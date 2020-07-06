local _fs = require "_fs"
local fs = {}

fs.open = _fs.open
fs.remove = _fs.remove
fs.remove_all = _fs.remove_all
fs.stat = _fs.stat
fs.read_dir = _fs.read_dir
fs.mkdir = _fs.mkdir
fs.mkdir_all = _fs.mkdir_all

fs.readfile = function(path, cb)
    fs.stat(path, function(err, stat)
        if err ~= nil then
            cb(err)
            return
        end
        fs.open(path, function(err, file)
            if err ~= nil then
                cb(err)
                return
            end
            file:read(stat.size, function(err, content)
                file:close()
                cb(err, content)
            end)
        end)
    end)
end

return fs