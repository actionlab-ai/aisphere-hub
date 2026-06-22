# SkillHub 最终版功能说明与对接文档

版本：`skillhub-final-rbac-group-first-v1`

本版本是一个前后端分离的独立 SkillHub 项目：

```text
skillhub/
  backend/   Gin 后端，负责 API、认证、权限、存储、缓存、运行时下载
  front/     React + Vite 管理台，负责 Skill/Group/Governance/Ops/IAM 管理
  docs/      架构、接口、部署和能力边界文档
```

## 1. 核心概念重新定义

### 1.1 Namespace 不是 Skill 分类

本版明确调整设计：

```text
Namespace = 账号体系 / 租户 / 环境 / RBAC 授权边界
Group     = Skill 业务分组 / 能力包 / 分类 / Agent 加载单元
Skill     = 具体技能资源，真正按 Group 组织和复用
```

也就是说：

- 不再把 Namespace 当成 Skill 的业务分类。
- Namespace 只用于访问控制、租户隔离、环境隔离，例如 `dev`、`prod`、`team-a`。
- Skill 的业务归属、能力分类、Agent 一次加载哪些技能，应该通过 Group 管理。
- 一个 Group 可以包含多个 Skill。
- 一个 Skill 可以被多个 Group 引用。
- Group 仍然位于某个 Namespace 下，这是为了权限和租户隔离，不是为了业务分类。

推荐使用方式：

```text
namespace = prod
  group = novel-writing-suite
    skill = dialogue-card
    skill = scene-splitter
    skill = chapter-reviewer

namespace = dev
  group = novel-writing-suite
    skill = dialogue-card@latest
    skill = scene-splitter@gray
```

这里 `prod/dev` 是环境和权限边界；`novel-writing-suite` 才是业务能力包。

### 1.2 Group 是 Skill 的主组织方式

Group 用于解决：

```text
1. 一组相关 Skill 怎么管理
2. Agent Runtime 一次应该加载哪些 Skill
3. 不同环境下同名能力包怎么选择 stable/latest/gray
4. 前端怎么按业务能力查看 Skill
```

Group 成员支持：

```json
{
  "skillName": "dialogue-card",
  "label": "stable",
  "version": "",
  "required": true,
  "order": 1
}
```

如果指定 `version`，优先使用固定版本；如果指定 `label`，运行时解析 `stable/latest/gray`。

## 2. 已实现能力清单

### 2.1 Skill 管理

已实现：

```text
Skill 列表 / 搜索
按 Group 过滤 Skill
Skill ZIP 上传 / 批量上传
SKILL.md frontmatter 解析
Skill 详情
版本列表
版本文件树
文件内容预览
Markdown 渲染
版本对比 Diff
版本下载
draft / submit / publish / online / offline / redraft
latest / stable / gray 标签路由
scope / metadata / bizTags 管理
Star / Rating / Subscribe
下载计数
```

### 2.2 Skill Group 管理

已实现：

```text
Group 列表
Group 创建 / 更新 / 删除
Group 绑定 Skill
Group 解绑 Skill
Group Runtime Manifest
按 Group 过滤 Skill 列表
group manifest 缓存
group 资源级写锁
```

### 2.3 Runtime Client API

已实现：

```text
Agent 按 name + version 精确下载 Skill
Agent 按 name + label 下载 Skill
Agent 按 groupName + label 获取能力包 manifest
md5 / ETag / 304 Not Modified
Redis route cache
Redis version cache
Redis group cache
```

### 2.4 .well-known 发现协议

已实现：

```text
/registry/{namespace}/.well-known/agent-skills/index.json
/registry/{namespace}/.well-known/skills/index.json
/registry/{namespace}/.well-known/agent-skills/{skill}.zip
/registry/{namespace}/.well-known/agent-skills/{skill}/SKILL.md
/registry/{namespace}/api/search
```

`.well-known` 是外部发现协议，不是 Redis，不是缓存。Redis 只是后端生成 `.well-known` 响应时的内部加速层。

### 2.5 Agent Proposal / Governance Inbox

已实现：

```text
Agent 提交 Skill Proposal
Agent 读取 Overlay
Admin 查看 Proposal Inbox
Admin Validate / Approve / Reject
Approve 后生成候选正式版本
可绑定 gray/latest/stable
Proposal 不污染正式 Skill 缓存
```

设计边界：Agent 不允许直接改正式 Skill，不允许直接改 stable/latest/gray；Agent 只能提交 proposal 和 overlay。

### 2.6 Auth / IAM / RBAC / ABAC

已实现：

```text
local / external / mixed 三种认证模式
首次初始化管理员
本地账号登录
本地账号 PostgreSQL 存储
外部 JWT / OIDC / Introspection / API Key Provider 架构
DB Token / API Key Provider
Subject 统一模型：human / organization / agent / service
Namespace 成员管理
Namespace 角色：owner / admin / developer / reviewer / viewer
路由级 ABAC 基础校验
whoami
```

### 2.7 存储、缓存、锁

已实现：

```text
PostgreSQL 自动建库
PostgreSQL migration 自动执行
本地 JSON fallback
MinIO / S3 ObjectStore
Local ObjectStore fallback
Redis Single / Redis Cluster
Memory Cache fallback
Redis 全局限流
资源级 Redis/Memory Lock
写后删缓存
读侧短等待写锁
TTL 兜底
```

### 2.8 运维治理

已实现：

```text
Audit Log
全量写请求 Audit Middleware
Idempotency-Key 幂等
Rate Limit
Metrics: /metrics 和 /v3/admin/metrics
Subscribe Notification
SSE 通知流
Token / API Key 管理
```

## 3. 前端怎么用

### 3.1 启动前端开发模式

```bash
cd front
npm install
npm run dev
```

Vite 会代理：

```text
/v3
/registry
/healthz
```

### 3.2 生产构建

```bash
cd front
npm install
npm run build
```

构建产物：

```text
front/dist
```

后端配置：

```yaml
web:
  enabled: true
  dir: "../front/dist"
  routePrefix: "/ui"
  indexFallback: true
```

访问：

```text
http://127.0.0.1:8848/ui/
```

### 3.3 前端主要页面

| 页面 | 作用 |
|---|---|
| Skills | Skill CRUD、上传、版本、文件树、Markdown、Diff、Star、Rating、Subscribe |
| Groups | 业务能力包管理，绑定一组 Skill |
| Namespaces | 账号/RBAC 隔离空间，管理成员和角色 |
| Governance | Proposal 审核工作台 |
| Ops | Audit、Token、Metrics、通知、限流、幂等说明 |
| IAM | 本地账号、Agent 账号、Service 账号 |
| Docs | 接口和能力边界说明 |

### 3.4 前端操作流程

首次部署：

```text
1. 打开 /ui/
2. 如果 setupRequired=true，进入首次初始化页面
3. 设置 admin 用户和密码
4. 登录后进入 Console
```

上传 Skill：

```text
1. 进入 Skills
2. 选择当前 Namespace，通常是 public/dev/prod
3. 上传 Skill ZIP
4. 查看版本、文件树、SKILL.md
5. Submit / Publish / Online
6. 设置 stable/latest/gray labels
```

创建 Group：

```text
1. 进入 Groups
2. 创建 group，例如 novel-writing-suite
3. 绑定 dialogue-card、scene-splitter 等 Skill
4. Runtime 通过 groupName 拉取 manifest
```

治理 Proposal：

```text
1. Agent 调用 /v3/agent/ai/skill-proposals 提交 delta
2. 前端 Governance 页面查看
3. Validate
4. Approve 到 gray
5. 验证后再切 stable
```

## 4. 后端怎么用

### 4.1 本地开发启动

```powershell
cd backend
go mod tidy
go run .\cmd\skillhub\main.go --config .\config.example.yaml
```

`config.example.yaml` 已提供开发用本地签名密钥，可以直接跑。生产必须更换。

### 4.2 生产模式

```powershell
cd backend
$env:SKILLHUB_LOCAL_AUTH_SIGNING_SECRET = "your-long-random-secret-at-least-32-chars"
go run .\cmd\skillhub\main.go --config .\configs\postgres-redis-minio.yaml
```

首次启动会自动：

```text
1. 解析 PostgreSQL DSN
2. 自动 CREATE DATABASE IF NOT EXISTS
3. 执行 migrations/*.sql
4. 创建 schema_migrations
5. 启动 Gin 服务
```

### 4.3 配置模块

核心配置：

```yaml
server:
  addr: ":8848"

database:
  provider: "postgres"
  dsn: "root:CHANGE_ME_PASSWORD@tcp(127.0.0.1:3306)/skillhub?charset=utf8mb4&parseTime=true&loc=Local"
  autoCreate: true

migration:
  enabled: true
  dir: "./migrations"

cache:
  provider: "redis"
  redis:
    mode: "single" # single / cluster
    addrs: ["127.0.0.1:6379"]

objectStore:
  provider: "minio"
  s3:
    endpoint: "127.0.0.1:9000"
    bucket: "skillhub"

auth:
  enabled: true
  mode: "mixed" # local / external / mixed
```

## 5. 核心 API 规范摘要

### 5.1 Auth

```http
GET  /v3/auth/setup/status
POST /v3/auth/setup
POST /v3/auth/login
POST /v3/auth/refresh
GET  /v3/auth/me
GET  /v3/admin/iam/whoami
```

### 5.2 Namespace / RBAC

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

### 5.3 Skill Admin

```http
GET    /v3/admin/ai/skills/list?namespaceId=public&groupName=novel-suite
GET    /v3/admin/ai/skills
GET    /v3/admin/ai/skills/version
GET    /v3/admin/ai/skills/version/files
GET    /v3/admin/ai/skills/version/file
GET    /v3/admin/ai/skills/version/compare
GET    /v3/admin/ai/skills/version/download
POST   /v3/admin/ai/skills/upload
POST   /v3/admin/ai/skills/upload/batch
POST   /v3/admin/ai/skills/draft
PUT    /v3/admin/ai/skills/draft
DELETE /v3/admin/ai/skills/draft
POST   /v3/admin/ai/skills/submit
POST   /v3/admin/ai/skills/publish
POST   /v3/admin/ai/skills/online
POST   /v3/admin/ai/skills/offline
PUT    /v3/admin/ai/skills/labels
PUT    /v3/admin/ai/skills/biz-tags
PUT    /v3/admin/ai/skills/metadata
PUT    /v3/admin/ai/skills/scope
DELETE /v3/admin/ai/skills
```

### 5.4 Group

```http
GET    /v3/admin/ai/skill-groups/list
GET    /v3/admin/ai/skill-groups
POST   /v3/admin/ai/skill-groups
PUT    /v3/admin/ai/skill-groups
DELETE /v3/admin/ai/skill-groups
POST   /v3/admin/ai/skill-groups/bind
DELETE /v3/admin/ai/skill-groups/bind
GET    /v3/client/ai/skill-groups
```

### 5.5 Runtime Client

```http
GET /v3/client/ai/skills?namespaceId=public&name=dialogue-card&label=stable
GET /v3/client/ai/skills?namespaceId=public&name=dialogue-card&version=0.0.1
GET /v3/client/ai/skill-groups?namespaceId=public&groupName=novel-suite&label=stable
```

### 5.6 Governance Proposal

```http
POST /v3/agent/ai/skill-proposals
GET  /v3/agent/ai/skill-proposals/{proposalId}
GET  /v3/agent/ai/skill-overlays/{proposalId}
GET  /v3/admin/ai/skill-proposals/list
GET  /v3/admin/ai/skill-proposals/{proposalId}
POST /v3/admin/ai/skill-proposals/{proposalId}/validate
POST /v3/admin/ai/skill-proposals/{proposalId}/approve
POST /v3/admin/ai/skill-proposals/{proposalId}/reject
```

### 5.7 Ops

```http
GET    /v3/admin/audit/logs
GET    /v3/admin/iam/tokens
POST   /v3/admin/iam/tokens
DELETE /v3/admin/iam/tokens/{keyId}
GET    /v3/admin/notifications
GET    /v3/admin/notifications/stream
POST   /v3/admin/notifications/{notificationId}/read
GET    /metrics
GET    /v3/admin/metrics
```

## 6. 对接后端的关键点

### 6.1 所有写请求建议带 Idempotency-Key

```http
Idempotency-Key: upload-skill-20260616-001
```

同 key 同请求会复用响应；同 key 不同请求返回 409。

### 6.2 Token 使用

本地登录返回：

```json
{
  "accessToken": "...",
  "refreshToken": "...",
  "tokenType": "Bearer"
}
```

请求：

```http
Authorization: Bearer <accessToken>
```

DB Token / API Key：

```http
X-API-Key: skh_xxx
```

### 6.3 Agent 账号权限建议

拆书/提炼 Skill 的 Agent：

```text
skill:read
skill:proposal:create
skill:proposal:read
skill:overlay:read
```

审核 Agent：

```text
skill:read
skill:proposal:read
skill:proposal:review
```

不建议给 Agent：

```text
skill:publish
skill:delete
skill:label:update:stable
system:admin
```

## 7. 能力边界与未完成项

已完成工程主链路，但仍建议后续加强：

```text
1. 完整 OpenAPI schema 生成 SDK
2. 更严格的字段级 ABAC
3. Proposal 接真实自动 Eval Pipeline
4. Subscribe SSE 改 Redis Pub/Sub 或 NATS
5. Rate Limit 增加按 subjectId / namespace / token 维度限流
6. 前端 OIDC Authorization Code Callback 页面
7. 前端按钮级权限隐藏
8. 更完整的 E2E 测试和压测
```

## 8. 常见问题

### 8.1 为什么 Namespace 还出现在 Skill API 参数里？

因为 Namespace 是权限和租户边界。Skill 在数据库里仍然需要属于某个 Namespace，方便：

```text
1. 区分 dev/prod/tenant/team
2. 控制谁能读写
3. 避免不同团队同名 Skill 冲突
```

但业务组织方式不是 Namespace，而是 Group。

### 8.2 怎么按业务分类管理 Skill？

使用 Group。

```text
Groups -> 创建 novel-writing-suite -> 绑定多个 Skill
Skills -> 输入 groupName 过滤
Runtime -> /v3/client/ai/skill-groups?groupName=novel-writing-suite
```

### 8.3 删除本地 data 目录会影响初始化吗？

如果 `database.provider=postgres`，不会。账号保存在 PostgreSQL-backed local account store。只有 `database.provider=local/file` 时才使用 `data/iam/local_accounts.json`。

### 8.4 local signingSecret 报错怎么办？

生产环境必须设置：

```powershell
$env:SKILLHUB_LOCAL_AUTH_SIGNING_SECRET = "your-long-random-secret-at-least-32-chars"
```

开发用 `config.example.yaml` 已内置一个 dev-only secret，方便本地启动；生产不能使用。
