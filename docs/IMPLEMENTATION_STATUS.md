# SkillHub 当前实现状态

版本：`skillhub-governance-complete+docs-v1`

## 1. 项目结构

```text
skillhub/
  backend/   Go Gin 服务
  front/     React + Vite 管理台
  docs/      顶层设计和接口文档
```

## 2. 已完成模块

| 模块 | 状态 | 说明 |
|---|---:|---|
| 前后端分离 | 已完成 | `front/` + `backend/` |
| Nacos Skill Admin API | 已完成 | 覆盖核心 CRUD、上传、版本、发布、上下线 |
| Runtime Client API | 已完成 | 按 name/version/label 下载 ZIP，支持 md5/304 |
| Well-known Registry | 已完成 | index、zip、SKILL.md、资源访问 |
| MySQL migration | 已完成 | 自动建库、建表、schema_migrations |
| Redis cache | 已完成 | 支持 single / cluster 配置和基础封装 |
| MinIO/S3 object store | 已完成 | Skill ZIP / SKILL.md / manifest 存储路径 |
| Resource Lock | 已完成 | skill/group 粒度写锁，读侧短等待 |
| Group | 已完成 | 一组 Skill 能力包管理和 runtime manifest |
| Agent Proposal / Overlay | 已完成 | Agent 自迭代提交候选，Admin 审核 |
| Local Auth | 已完成 | 首次初始化、登录、刷新、本地账号文件 |
| External Auth | 已完成 | JWT / OIDC / Introspection / API Key Provider 架构 |
| Namespace 成员体系 | 已完成基础 | namespace、member、role、owner |
| Star / Rating / Subscribe | 已完成基础 | 社交数据保存与查询 |
| Audit Log | 已完成基础 | 审计表和查询接口 |
| Token 管理 | 已完成基础 | 创建/列表/删除 API Key，明文只返回一次 |
| Idempotency | 已完成基础 | `Idempotency-Key` 中间件 |
| Rate Limit | 已完成基础 | 本地内存写请求限流 |
| Metrics | 已完成基础 | `/metrics` 和 `/v3/admin/metrics` |
| 前端 Console | 已完成基础 | Skills、Groups、Namespaces、Governance、Ops、IAM |

## 3. 未完成 / 待增强模块

| 模块 | 当前边界 | 建议下一步 |
|---|---|---|
| Redis 全局限流 | 当前主要是本地内存限流 | 改为 Redis Lua/token bucket |
| DB Token 认证 | Token 表和管理 API 已有，Provider 仍需完全接 DB | APIKeyProvider 增加 DB/Cache 查询 |
| 全量 Audit | 部分接口已接入，老接口未必全覆盖 | 增加统一 audit middleware / service hook |
| ABAC 强校验 | Namespace 成员表已建，路由级强校验仍需增强 | Authorizer 增加 resource scope 判断 |
| Proposal Validate | 轻量校验 | 接 eval pipeline、回归测试、质量评分 |
| Markdown 渲染 | 前端基础预览 | react-markdown + rehype-sanitize |
| Diff Viewer | 轻量行级 diff | 引入 hunk/inline diff |
| Subscribe 通知 | 仅保存订阅关系 | 通知中心、SSE、邮件/消息回调 |
| OIDC 前端登录 | 后端 Provider 架构有，前端授权码回调未完善 | Authorization Code + PKCE |
| OpenAPI SDK | 已有手写规范 | 后续自动生成 OpenAPI / TS Client |
| 测试 | 仅局部包测试 | 单测、集成测试、E2E、压测 |

## 4. 关键设计边界

### 4.1 Client API 只读

`/v3/client/ai/skills` 只用于 Agent runtime 下载正式 Skill，不允许写入。

### 4.2 Agent 自迭代走 Proposal

Agent 修改 Skill 不能直接走 Admin API，也不能直接发布。正确路径：

```text
Agent -> Proposal -> Overlay -> Validate -> Approve -> Candidate/Gray -> Stable
```

### 4.3 Well-known 不是 Redis

`.well-known` 是对外发现协议。Redis 是服务内部生成 index/route/version 响应的缓存层。

### 4.4 Namespace 与 Group

```text
Namespace = 环境/租户/隔离边界
Group = namespace 内的一组 Skill 能力包
```

### 4.5 缓存一致性

采用：

```text
写锁 + 写后删缓存 + 读侧短等待 + Redis miss 回源 + TTL 兜底
```

## 5. 推荐下一版优先级

1. APIKeyProvider 读取 DB token 表，并加入 Redis token cache。
2. Rate Limit 改 Redis 全局限流。
3. 全量 Audit Hook 覆盖所有写接口。
4. Authorizer 强化 namespace member / role / owner 校验。
5. Governance Inbox 接真实 eval pipeline。
6. 前端 Markdown/Diff 组件增强。
7. 生成 OpenAPI YAML 和 TypeScript Client。
