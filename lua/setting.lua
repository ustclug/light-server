local _M = {}

local conf = {
    redis_read_url="redis://127.0.0.1:6379/0",
    redis_read_auth="",
    redis_write_url="redis://127.0.0.1:6379/0",
    redis_write_auth="",
    domain_updater_interval = 10,
    data_updater_interval = 10,
    influx_measurement = "light"
}

function _M.get_conf(name)
    return conf[name]
end

return _M
