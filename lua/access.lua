
local function parse_authorization_header(header)
    if header == nil then
        return
    end

    local m, err = ngx.re.match(header, "^Basic +(?<credential>[a-zA-Z0-9+/=]+)$", "ij")
    if err then
        ngx.log(ngx.ERR, "error: ", err)
        return
    end
    if m == nil then -- not match
        return
    end

    local credential_base64 = m["credential"]
    if credential_base64 == nil or credential_base64 == "" then
        return
    end

    local credential = ngx.decode_base64(credential_base64)
    if credential == nil then
        return
    end

    local m, err = ngx.re.match(credential, "^(?<user>[^:]+):(?<pass>[^:]*)$")
    if err then
        ngx.log(ngx.ERR, "error: ", err)
        return
    end
    if m == nil then -- not match
        return
    end

    return m["user"], m["pass"]
end

local function redis_get_user_info(user)
    local rc = require("resty.redis.connector").new()

    local redis, err = rc:connect({
        url = Redis_read_url,
        password = Redis_read_auth
    })
    if not redis then
        ngx.say("failed to connect: ", err)
        return
    end

    local res, err = redis:hmget("user:" .. user, "Cleartext-Password", "Expiration", "Data-Plan", "Data-Used")
    if err then
        ngx.log(ngx.ERR, err)
    end

    local ok, err = rc:set_keepalive(redis)
    if not ok then
        ngx.log(ngx.ERR, err)
    end

    return res
end

local function reject(err, status_code)
    status_code = status_code or 407
    ngx.header["Proxy-Authenticate"] = 'Basic realm=""'
    ngx.header["X-Access-Point-Error"] = err
    ngx.exit(status_code)
end

local user, pass2check = parse_authorization_header(ngx.req.get_headers()["Proxy-Authorization"])
ngx.log(ngx.DEBUG, "get auth request with user: ", user, ", pass: ", pass)

if not user then
    reject("username not exist")
end

local user_info = redis_get_user_info(user)
if user_info == ngx.null then
    reject("username not exist")
end
local password = user_info[1]
local expire = tonumber(user_info[2])
local data_plan = tonumber(user_info[3])
local data_used = tonumber(user_info[4])

-- password check
if pass2check == nil or pass2check ~= password then
    reject("wrong password")
end

-- expiration check
ngx.log(ngx.DEBUG, "expire: ", expire, " ngx.time(): ", ngx.time())
if expire ~= nil and expire < ngx.time() then
    reject("account have expired")
end

-- data check
if data_plan ~= nil then
    if data_used == nil then
        data_used = 0
    end
    if data_used > data_plan then
        reject("account data run out")
    end
    ngx.log(ngx.DEBUG, "user data: ", user, ":", data_plan, ":", data_used)
end

-- auth success, trigger log phrase
ngx.ctx.user = user
