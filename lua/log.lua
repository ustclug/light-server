local conf = require("setting")

local stat = ngx.shared.stat
local used_bytes = ngx.shared.used_bytes
local user = ngx.ctx.user
local influx_measurement = conf.get_conf("influx_measurement")

if user == nil then
    return
end

local line = string.format(
    [[%s,user=%s,client_addr=%s bytes=%si,target="%s:%s",duration=%.3f %d000000]],
    influx_measurement,
    user, ngx.var.remote_addr,
    ngx.var.bytes_sent,
    ngx.var.connect_host or ngx.var.host, ngx.var.connect_port or ngx.var.proxy_port,
    ngx.var.request_time,
    tostring(ngx.now() * 1000))

ngx.log(ngx.INFO, "log influx line: ", line)

local _, err = stat:lpush("influx", line)
if err then
    ngx.log(ngx.ERR, err)
end

used_bytes:incr(user, ngx.var.bytes_sent, 0)
