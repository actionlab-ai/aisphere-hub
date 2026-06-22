# SkillHub Governance Enhancement v2

本版本继续补齐生产治理能力，并修复 MySQL migration 兼容问题。

## 修复：MySQL migration 003 报错

旧版本使用了：

```sql
ALTER TABLE iam_subject ADD COLUMN IF NOT EXISTS ...
```

部分 MySQL 版本不支持 `ADD COLUMN IF NOT EXISTS`，会报：

```text
Error 1064 (42000)
```

本版改为：

1. migration SQL 使用普通 `ALTER TABLE ADD COLUMN`。
2. Go migration runner 对 MySQL `1060 duplicate column`、`1061 duplicate key`、`1091 missing column/key` 做可忽略处理。
3. 因此同一套 migration 可以兼容 MySQL 5.7 / 8.x 的常见部署场景。

## 新增/增强能力

### 1. Redis 全局 Rate Limit

原来写请求限流是本地内存版，多副本部署时每个副本各自计数。本版新增 `DistributedRateLimiter`：

- 当 `cache.provider=redis` 时，写请求限流走 Redis `INCR + EXPIRE`。
- 支持 Redis Single 和 Redis Cluster，因为底层使用统一 `ports.Cache`。
- 当 `cache.provider!=redis` 时自动回退本地内存限流。

配置：

```yaml
ops:
  rateLimit:
    enabled: true
    writeLimitPerMinute: 300
```

### 2. DB Token / API Key Provider

Token 管理表现在真正可以参与鉴权：

- 创建 token 时只返回一次明文 token。
- 数据库存储 `token_hash`，不保存明文。
- `dbapikey.Provider` 会从 `skillhub_token` 表按 token hash 查询 active token。
- 支持 `X-API-Key` 和 `Authorization: Bearer`。

相关接口：

```http
GET    /v3/admin/iam/tokens
POST   /v3/admin/iam/tokens
DELETE /v3/admin/iam/tokens/{keyId}
```

### 3. 全量写请求 Audit Middleware

新增 `AuditMiddleware`：

- 对非 GET 请求，在响应 2xx 后记录审计日志。
- 记录 action、operator、path、status、namespace、resource 等信息。
- 旧 Skill 写接口也会进入统一审计链路。

配置：

```yaml
ops:
  audit:
    enabled: true
```

### 4. Markdown Renderer + Rich Diff Viewer

前端 Skill 文件预览增强：

- `SKILL.md` / `.md` / `.markdown` 文件使用 `react-markdown` 渲染。
- 普通文件继续用安全代码预览。
- 版本对比改为 `react-diff-viewer-continued`，支持 split view 和词级差异。

新增前端依赖：

```json
"react-markdown": "^10.1.0",
"react-diff-viewer-continued": "^4.0.5"
```

### 5. Subscribe Notification + SSE

Subscribe 不再只是保存订阅关系。本版新增通知表和通知接口：

```http
GET  /v3/admin/notifications
GET  /v3/admin/notifications/stream
POST /v3/admin/notifications/{notificationId}/read
```

当前实现：

- Skill 发布时通知订阅该 Skill 的 subject。
- 前端 Ops 页面新增 Notifications 面板。
- SSE 采用轻量 polling SSE；后续可以替换成 Redis Pub/Sub 或 NATS。

### 6. Namespace 成员 ABAC 强化

新增 `StoreAuthorizer`：

- 先走原有 permission / role / namespace 权限。
- 如果 principal 没有显式 namespace 权限，则查询 `skillhub_namespace_member`。
- namespace 成员角色可授权：
  - `owner/admin`：管理权限。
  - `developer`：写入、proposal、开发类权限。
  - `reviewer`：proposal review。
  - `viewer`：只读。

这让 namespace 成员体系不只是存储字段，而能参与路由级授权判断。

## 新增 migration

```text
006_notifications_and_token_hash.sql
```

新增/调整：

- `skillhub_token.token_hash`
- `skillhub_notification`

## 新增接口

### Notifications

```http
GET /v3/admin/notifications?subjectId=&unreadOnly=true&pageNo=1&pageSize=50
GET /v3/admin/notifications/stream
POST /v3/admin/notifications/{notificationId}/read
```

### Token 鉴权能力

原接口不变，但 token 现在可用于真实 API 鉴权：

```http
X-API-Key: skh_xxx
Authorization: Bearer skh_xxx
```

## 当前仍建议后续增强

1. SSE 当前是 polling SSE，可升级 Redis Pub/Sub。
2. Metrics 仍是内置计数器，可继续接 Prometheus histogram。
3. ABAC 当前覆盖 namespace 成员角色，后续可扩展到 skill owner / group owner 级策略。
4. Rate Limit 当前是固定窗口，可升级滑动窗口或令牌桶 Lua。
