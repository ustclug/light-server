local stat = ngx.shared.stat

local function stat_rpop(k)
    return stat:rpop(k)
end
for line in stat_rpop, "influx" do
    ngx.say(line)
end
