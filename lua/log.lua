local bytes = ngx.shared.stat_bytes
local counts = ngx.shared.stat_counts
local user = ngx.ctx.user

if user == nil then
    return
end

local key = user .. ":" .. ngx.var.host .. ":" .. ( ngx.var.connect_port or ngx.var.proxy_port )
ngx.log(ngx.INFO, "log key: ", key)
counts:incr(key, 1, 0)
bytes:incr(key, ngx.var.bytes_sent, 0)

