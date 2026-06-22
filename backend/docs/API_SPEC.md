
## Notifications / Subscribe Events

```http
GET  /v3/admin/notifications
GET  /v3/admin/notifications/stream
POST /v3/admin/notifications/{notificationId}/read
```

### GET /v3/admin/notifications

Query:

| 参数 | 说明 |
|---|---|
| subjectId | 可选，默认当前登录主体 |
| unreadOnly | true/false |
| pageNo | 默认 1 |
| pageSize | 默认 50 |

### GET /v3/admin/notifications/stream

轻量 SSE 订阅接口，当前实现为 polling SSE，后续可替换 Redis Pub/Sub。

### POST /v3/admin/notifications/{notificationId}/read

标记通知已读。

## DB Token 鉴权

`POST /v3/admin/iam/tokens` 创建的 token 现在会存储 `token_hash`，明文只返回一次。之后可以通过：

```http
X-API-Key: skh_xxx
```

或：

```http
Authorization: Bearer skh_xxx
```

调用接口。权限来自 token 绑定的 `subjectId / roles / permissions / namespaces`。

## Redis 全局 Rate Limit

当 `cache.provider=redis` 时，写请求限流使用 Redis 全局计数；当使用 memory/local 时回退本地内存限流。

## Namespace ABAC

除 token/JWT 中显式 `namespaces` 外，系统还会查询 namespace member：

| Role | 能力 |
|---|---|
| owner/admin | namespace 内管理权限 |
| developer | 写入、开发、proposal |
| reviewer | proposal review |
| viewer | 只读 |
