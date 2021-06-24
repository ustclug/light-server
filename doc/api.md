# light-server (nginx version)

[TOC]

## 整理架构

### 认证侧

后端服务器将用户认证信息写入 Redis 数据库中。接入服务查询 Redis 数据库实现认证功能。

### 日志侧

接入节点周期性报告用户的访问数据，并存储至时序数据库 InfluxDB 中。后端从 InfluxDB 中查询详细的统计信息。

## 快速开始

**Step1: 添加用户信息**

添加用户 `test`，设置密码 `1`：

```bash
$ redis-cli -h redis-master.hostname -a bd20120115509f4fa0121190d2336a

> HSET user:test Cleartext-Password 1
(integer) 1
```

为用户 `test`设置流量限制 `1GB`：

```redis
> HSET user:test Data-Plan 1073741824
(integer) 1
```

**Step2: 测试**

```shell
$ curl -v -x https://proxy.hostname:29980/ -U "test:1" http://ipv4.ip.sb/

```

**Step4: 查询用户已用总流量**

查询用户 `test`已用的总流量:

```redis
> HGET user:test Data-Used
"2465"
```

**Step5: 查询统计信息**

通过 Homebrew/Linuxbrew 安装 InfluxDB CLI

```shell
$ brew install influxdb
```

查询过去30天 test 用户的日流量和使用时长

```shell
$ influx -database light -host xx.xx.xx.xx -port 8086 -username dbread -password XXXXX

> precision rfc3339
> SELECT sum("bytes") AS bytes , sum("duration") AS intervals FROM "light" WHERE ("user" = 'test') AND time >= now() - 30d  GROUP BY time(1d)

name: light
time                 bytes duration
----                 ----- ---------
YYYY-MM-DDT00:00:00Z 11859 80
```

## 用户认证

用户认证信息存储在 Redis 数据库的 Hash 表中，定义如下：

* key: `user:<username>`
* field: 用户属性名
* value: 属性值

目前支持的用户属性：

| 属性名                 |  属性值               |      |
| --------------------- | ------------------- | ---- |
| Cleartext-Password    | 明文表示的用户密码     | 必选 |
| Expiration            | 过期时间，Unix时间格式 | 可选 |
| Domain-List           | 用户对应域名列表       | 可选 |
| Data-Plan             | 流量限制，单位:byte   | 可选 |
| Data-Used             | 已用流量，单位:byte   | 只读 |

比如，添加一个用户 `test` ，密码为 `1`，并设置账户过期时间为 2020年1月1日0点

```redis
> HSET user:test Cleartext-Password 1
(integer) 1
> HSET user:test Expiration 1577836800
(integer) 1
```

**tips**：可以使用 Redis 提供的过期机制自动删除过期用户 `> EXPIREAT user:test 1577836800` 

## 日志与统计

日志储存于时序数据库 InfluxDB 中，用于统计。

数据的时间精度为10秒，即：每隔10秒采集一次日志。每次采集后会清空暂存区，即每次收集到的是过去10秒的日志信息。

每隔约1分钟，会将采集到的日志上传至 InfluxDB 数据库。

InfluxDB 数据定义如下

**请求相关的测量量：**

* **field:** 
  * bytes 流量
  * target 访问目标
  * duration 持续时间
* **measurement:** light
* **tag:**
  * host 接入节点主机名
  * user 用户名
  * client_addr 用户IP
