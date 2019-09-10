local ngx_re = require "ngx.re"
local bytes = ngx.shared.stat_bytes
local counts = ngx.shared.stat_counts
local meta = ngx.shared.stat_meta

local now = ngx.time()
local last_update = meta:get("last_update")
local intervals = type(last_update) ~= "number"
    and 0 or now - last_update

local keys = bytes:get_keys(0)
for _, key in pairs(keys) do

    local res, err = ngx_re.split(key, ":")
    local user = res[1]
    local domain = res[2]
    local port = res[3]

    if user == nil or domain == nil or port == nil then
        ngx.log(ngx.ERR, "wrong stat key format: ", key)
        goto continue
    end

    local b = bytes:get(key)
    bytes:delete(key)

    local c = counts:get(key)
    counts:delete(key)

    c = c == nil and 0 or c
    -- race condition
    -- set default value 0

    ngx.say("light",
    ",user=", user,
    " ",
    "intervals=", intervals)
    ngx.say("light",
    ",user=", user,
    ",domain=", domain,
    ",port=", port,
    " ",
    "bytes=", b,
    ",counts=", c)

    ::continue::
end

meta:set("last_update", now)
