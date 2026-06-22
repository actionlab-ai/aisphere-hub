# SkillHub Governance Feature Completion

本版本在前后端分离版基础上补齐了参考项目中的治理类能力，并保留 Go SkillHub 的工程化架构。

## 新增后端能力

### Namespace 成员 / 角色 / Owner 管理

新增接口：

```http
GET    /v3/admin/namespaces
GET    /v3/admin/namespaces/{namespaceId}
POST   /v3/admin/namespaces
PUT    /v3/admin/namespaces/{namespaceId}
GET    /v3/admin/namespaces/{namespaceId}/members
POST   /v3/admin/namespaces/{namespaceId}/members
PUT    /v3/admin/namespaces/{namespaceId}/members/{subjectId}
DELETE /v3/admin/namespaces/{namespaceId}/members/{subjectId}
```

角色建议：

```text
owner       namespace owner
admin       namespace 管理员
developer   可维护 skill
reviewer    可审核 proposal
viewer      只读
```

### Star / Rating / Subscribe

新增接口：

```http
GET  /v3/admin/ai/skills/social?namespaceId=public&skillName=xxx
POST /v3/admin/ai/skills/social/star
POST /v3/admin/ai/skills/social/rating
POST /v3/admin/ai/skills/social/subscribe
```

用途：

```text
Star       收藏 Skill
Rating     评分 Skill
Subscribe 订阅 Skill 更新
```

### Governance Inbox

原 Agent Proposal 基础接口保留，本版新增前端治理工作台，将 proposal 做成待处理收件箱。

支持：

```text
查看 proposal
查看 delta / evidence
validate
approve -> gray
reject
```

### Audit Log

新增接口：

```http
GET /v3/admin/audit/logs
```

新增后端表：

```text
skillhub_audit_log
```

当前已在 Namespace、Member、Token 等操作中写入审计，后续可继续把所有 Skill 写操作接入统一 audit middleware。

### Token / API Key 管理

新增接口：

```http
GET    /v3/admin/iam/tokens
POST   /v3/admin/iam/tokens
DELETE /v3/admin/iam/tokens/{keyId}
```

创建 token 时只返回一次明文 token，列表不返回 token 明文。

### Idempotency

新增中间件，支持写请求幂等：

```http
Idempotency-Key: <unique-key>
```

覆盖：

```text
POST / PUT / DELETE
```

策略：

```text
同 key + 同请求 hash：返回上次响应
同 key + 不同请求 hash：409
TTL 默认 3600 秒
```

### Rate Limit

新增写请求限流中间件：

```yaml
ops:
  rateLimit:
    enabled: true
    writeLimitPerMinute: 120
```

当前按 ClientIP 进行本地内存限流。生产多副本后可升级为 Redis sliding window。

### Metrics

新增接口：

```http
GET /metrics
GET /v3/admin/metrics
```

暴露：

```text
requests_total
errors_total
uptime_seconds
by_path
by_status
```

### Version File Tree / Version Compare

上版已补，本版保留并增强前端展示：

```http
GET /v3/admin/ai/skills/version/files
GET /v3/admin/ai/skills/version/file
GET /v3/admin/ai/skills/version/compare
```

前端增加文件树、文件预览、左右 diff 视图。

## 新增数据库 migration

```text
migrations/005_governance_metrics_rate_limit.sql
```

包含：

```text
skillhub_namespace
skillhub_namespace_member
skillhub_star
skillhub_rating
skillhub_subscription
skillhub_audit_log
skillhub_token
skillhub_idempotency
```

## 新增前端页面

```text
Namespaces    namespace / member / role / owner 管理
Governance    proposal 收件箱
Ops           audit / token / metrics / idempotency / rate limit 说明
```

并增强：

```text
Skills 详情页新增 star/rating/subscribe
版本对比使用简易 diff view
文件目录树继续保留
```

## 仍未完全实现

```text
1. Redis 分布式 Rate Limit 还没做，目前是本地内存限流。
2. Token 后端当前保存了 token 管理元数据，但 API Key 认证 Provider 还没有自动读取 DB token 表。
3. Audit 还没有覆盖所有老接口写操作，已提供 AppendAudit 能力和部分接口接入。
4. Markdown 渲染器当前仍以安全纯文本/代码预览为主，未引入 markdown-it/react-markdown 依赖。
5. Diff 是轻量行级对比，未实现完整 hunk/inline diff。
6. Subscribe 目前只保存订阅关系，通知中心/SSE 推送未做。
7. Namespace 成员角色已存储，路由级 ABAC 强制校验还需要继续增强。
```

## 建议下一步

```text
1. 把 DB token 接入 APIKeyProvider，实现后台创建的 token 可直接调用接口。
2. 用 Redis 实现全局 Rate Limit。
3. 所有 Skill 写接口统一接入 audit middleware。
4. 加 react-markdown + diff viewer，完善前端预览体验。
5. Governance Inbox 增加 test/eval result 可视化。
6. Subscribe 接入通知中心和 SSE。
```
