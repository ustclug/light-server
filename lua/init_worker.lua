local used_bytes = ngx.shared.used_bytes

local conf = require("setting")

Redis_read_url = conf.get_conf("redis_read_url")
Redis_read_auth = conf.get_conf("redis_read_auth")
Redis_write_url = conf.get_conf("redis_write_url")
Redis_write_auth = conf.get_conf("redis_write_auth")
local data_updater_interval = conf.get_conf("data_updater_interval")

local function update_data_used()
    local rc = require("resty.redis.connector").new({
        url = Redis_write_url,
        password = Redis_write_auth
    })
    local redis, err = rc:connect()
    if err then
        ngx.log(ngx.ERR, "redis connect fail: ", Redis_write_url, ":", err)
        return
    end

    local users = used_bytes:get_keys(0)
    redis:init_pipeline()

    for _, user in ipairs(users) do
        local bytes = used_bytes:get(user)
        redis:hincrby("user:" .. user, "Data-Used", bytes)
    end

    local ret, err = redis:commit_pipeline()
    if ret then
        used_bytes:flush_all()
        ngx.log(ngx.DEBUG, "redis update user used data")
    else
        ngx.log(ngx.ERR, "redis update user used data fail: ", err)
    end

    local ok, err = rc:set_keepalive(redis)
    if not ok then
        ngx.log(ngx.ERR, err)
    end
end

-- timers only run on one worker
if 0 == ngx.worker.id() then
    local ok, err = ngx.timer.every(data_updater_interval, update_data_used)
    if not ok then
        ngx.log(ngx.ERR, "failed to create timer: ", err)
        return
    end
end
