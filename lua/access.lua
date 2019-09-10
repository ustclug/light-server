
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

local function redis_hget_user_field(user, field)
    local rc = require("resty.redis.connector").new()

    local redis, err = rc:connect({
        url = redis_url
    })
    if not redis then
        ngx.say("failed to connect: ", err)
        return
    end

    local res, err = redis:hget("user:" .. user, field)
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

local user, pass = parse_authorization_header(ngx.req.get_headers()["Proxy-Authorization"])
ngx.log(ngx.INFO, "get auth request with user: ", user, ", pass: ", pass)

if not user then
    reject("username not exist")
end

-- password check
if pass == nil or pass ~= redis_hget_user_field(user, "Cleartext-Password") then
    reject("wrong password")
end

-- expiration check
local expire = tonumber(redis_hget_user_field(user, "Expiration"))
ngx.log(ngx.INFO, "expire: ", expire)
ngx.log(ngx.INFO, "ngx.time(): ", ngx.time())
if expire ~= nil and expire < ngx.time() then
    reject("account have expired")
end

-- auth success, trigger log phrase
ngx.ctx.user = user
