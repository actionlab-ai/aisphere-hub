# SkillHub API 接口规范

适用版本：`skillhub-governance-complete+docs-v1`

本规范描述当前 SkillHub Go Edition 的 HTTP API。项目采用前后端分离：

```text
skillhub/
  backend/   Gin API + MySQL/Redis/MinIO/S3 + Auth + Migration
  front/     React + Vite Console
  docs/      API/部署/能力边界文档
```

## 1. 通用约定

### 1.1 Base URL

本地默认：

```text
http://127.0.0.1:8848
```

前端开发环境默认代理：

```text
/v3
/registry
/healthz
/metrics
```

### 1.2 认证方式

除首次初始化、登录、公开 registry/index、健康检查外，管理接口通常需要认证。

支持模式：

```yaml
auth:
  mode: local     # local / external / mixed
```

请求头：

```http
Authorization: Bearer <accessToken>
```

也支持 API Key：

```http
X-API-Key: <apiKey>
```

### 1.3 通用 JSON 响应

大多数 JSON API 返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误示例：

```json
{
  "code": 400,
  "message": "missing namespaceId"
}
```

文件下载类接口直接返回二进制或文本，不包裹 JSON。

### 1.4 Namespace 和 Group

```text
Namespace = 租户 / 环境 / 隔离边界，例如 public、dev、prod、tenant-a
Group     = 某个 namespace 内的一组 Skill 能力包，例如 novel-writing-suite
```

Group 不跨 namespace 引用 Skill。

### 1.5 Idempotency-Key

写请求可以携带：

```http
Idempotency-Key: <unique-key>
```

语义：

```text
同 key + 同请求 hash：返回上次响应
同 key + 不同请求 hash：409 Conflict
默认 TTL：由 ops.idempotency 配置控制
```

### 1.6 Runtime 缓存一致性

事实来源：

```text
MySQL: 元数据事实来源
S3/MinIO: Skill ZIP / 文件事实来源
Redis: 只作为缓存和计数
```

写操作：

```text
1. 按 namespace + skill/group 加资源级写锁
2. 写 MySQL / S3 成功
3. 删除 Redis route/version/index/group 缓存
4. 释放锁
```

读操作：

```text
1. Runtime 读前短等资源锁
2. 先查 Redis
3. miss 回源 MySQL/S3
4. 回填 Redis
```

---

## 2. 健康检查与指标

### GET /healthz

健康检查。

响应：

```json
{"status":"ok"}
```

### GET /metrics

Prometheus 文本指标。

### GET /v3/admin/metrics

管理端 JSON 指标。

权限建议：`system:admin` 或 `metrics:read`。

---

## 3. Auth / 首次初始化 / 登录

### GET /v3/auth/setup/status

检查是否需要首次初始化管理员。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "localEnabled": true,
    "setupEnabled": true,
    "setupRequired": true,
    "mode": "local"
  }
}
```

### POST /v3/auth/setup

首次初始化管理员。仅当没有 active 本地账号时允许执行。

请求：

```json
{
  "username": "admin",
  "password": "ChangeMe_123!",
  "displayName": "Platform Admin",
  "email": "admin@example.com",
  "organization": "default",
  "setupToken": "optional-token"
}
```

响应返回账号和 token。

### POST /v3/auth/login

本地账号登录。

请求：

```json
{
  "username": "admin",
  "password": "ChangeMe_123!"
}
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "...",
    "refreshToken": "...",
    "tokenType": "Bearer",
    "expiresIn": 3600,
    "subjectId": "user:admin",
    "subjectType": "human"
  }
}
```

### POST /v3/auth/refresh

刷新本地 token。

请求：

```json
{"refreshToken":"..."}
```

### GET /v3/auth/me

返回当前登录主体。

### GET /v3/admin/iam/whoami

调试当前 Principal，包含 subject、roles、permissions、namespaces。

---

## 4. 本地账号管理

> 适用于 `auth.mode=local` 或 `mixed`。权限建议：`iam:admin`。

### GET /v3/admin/iam/local-users/list

查询本地用户。

### POST /v3/admin/iam/local-users

创建用户、Agent 账号或 Service 账号。

请求示例：

```json
{
  "username": "dialogue-agent",
  "password": "ChangeMe_123!",
  "subjectId": "agent:dialogue_skill_batch_worker",
  "subjectType": "agent",
  "roles": ["agent"],
  "permissions": ["skill:read", "skill:proposal:create", "skill:overlay:read"],
  "namespaces": ["public", "dev"]
}
```

### PUT /v3/admin/iam/local-users/{username}

更新本地用户。

### DELETE /v3/admin/iam/local-users/{username}

禁用本地用户。

---

## 5. Token / API Key 管理

> 权限建议：`iam:token:admin` 或 `iam:admin`。

### GET /v3/admin/iam/tokens

查询 token 列表。不会返回明文 token。

### POST /v3/admin/iam/tokens

创建 API Key。明文 token 只返回一次。

请求示例：

```json
{
  "name": "writer-agent-key",
  "subjectId": "agent:writer",
  "subjectType": "agent",
  "permissions": ["skill:read", "skill:proposal:create"],
  "namespaces": ["public"]
}
```

### DELETE /v3/admin/iam/tokens/{keyId}

删除 / 禁用 token。

---

## 6. Namespace 管理

Namespace 用于租户、环境或资源隔离。

### GET /v3/admin/namespaces

查询 namespace 列表。

### GET /v3/admin/namespaces/{namespaceId}

查询 namespace 详情。

### POST /v3/admin/namespaces

创建 namespace。

请求示例：

```json
{
  "namespaceId": "prod",
  "displayName": "生产环境",
  "description": "生产 Skill 空间",
  "owner": "user:admin"
}
```

### PUT /v3/admin/namespaces/{namespaceId}

更新 namespace。

### GET /v3/admin/namespaces/{namespaceId}/members

查询 namespace 成员。

### POST /v3/admin/namespaces/{namespaceId}/members

新增成员。

请求：

```json
{
  "subjectId": "user:alice",
  "subjectType": "human",
  "role": "developer"
}
```

角色建议：

```text
owner / admin / developer / reviewer / viewer
```

### PUT /v3/admin/namespaces/{namespaceId}/members/{subjectId}

更新成员角色。

### DELETE /v3/admin/namespaces/{namespaceId}/members/{subjectId}

移除成员。

---

## 7. Admin Skill API

基础路径：

```text
/v3/admin/ai/skills
```

### GET /v3/admin/ai/skills/list

Skill 列表。

Query：

```text
namespaceId  必填
skillName    可选
search       可选
owner        可选
scope        可选
bizTag       可选
pageNo       默认 1
pageSize     默认 20
```

### GET /v3/admin/ai/skills

Skill 详情。

Query：

```text
namespaceId
skillName 或 name
```

### GET /v3/admin/ai/skills/version

指定版本详情。

Query：

```text
namespaceId
skillName 或 name
version
```

### GET /v3/admin/ai/skills/version/download

下载指定版本 ZIP。

### GET /v3/admin/ai/skills/version/files

获取指定版本文件树。

Query：

```text
namespaceId
skillName 或 name
version
```

响应示例：

```json
{
  "code": 0,
  "data": {
    "namespaceId": "public",
    "skillName": "dialogue-card",
    "version": "0.0.1",
    "files": [
      {"path":"SKILL.md","name":"SKILL.md","type":"markdown","size":1234,"binary":false}
    ]
  }
}
```

### GET /v3/admin/ai/skills/version/file

获取指定版本内某个文件内容。

Query：

```text
namespaceId
skillName 或 name
version
path
```

### GET /v3/admin/ai/skills/version/compare

版本对比。

Query：

```text
namespaceId
skillName 或 name
baseVersion
targetVersion
```

当前返回两版 `SKILL.md` / 主要文本内容，前端做轻量行级 diff。

### POST /v3/admin/ai/skills/upload

上传单个 Skill ZIP。

Content-Type：`multipart/form-data`

字段：

```text
namespaceId     必填
overwrite       可选，true/false
targetVersion   可选
commitMsg       可选
file            必填，zip
```

### POST /v3/admin/ai/skills/upload/batch

批量上传 Skill ZIP。一个 ZIP 内可包含多个 Skill 子目录。

### POST /v3/admin/ai/skills/draft

创建草稿。

### PUT /v3/admin/ai/skills/draft

更新草稿。

### DELETE /v3/admin/ai/skills/draft

删除草稿。

### POST /v3/admin/ai/skills/submit

提交版本。当前轻量模式下可直接转发布链路；后续可接 Pipeline。

### POST /v3/admin/ai/skills/publish

发布版本。

### POST /v3/admin/ai/skills/force-publish

强制发布版本。权限建议：`skill:force-publish`。

### POST /v3/admin/ai/skills/redraft

把指定版本重新转成草稿。

### PUT /v3/admin/ai/skills/labels

更新 label 路由。

请求示例：

```json
{
  "namespaceId": "public",
  "skillName": "dialogue-card",
  "labels": {
    "latest": "0.0.3",
    "stable": "0.0.2",
    "gray": "0.0.3"
  }
}
```

### PUT /v3/admin/ai/skills/biz-tags

更新业务标签。

### PUT /v3/admin/ai/skills/metadata

更新运行时 metadata。

### PUT /v3/admin/ai/skills/scope

更新 scope。

### POST /v3/admin/ai/skills/online

上线 Skill 或指定版本。

### POST /v3/admin/ai/skills/offline

下线 Skill 或指定版本。

### DELETE /v3/admin/ai/skills

删除 Skill。

---

## 8. Skill Social API

### GET /v3/admin/ai/skills/social

查询当前主体对某个 Skill 的 star/rating/subscribe 状态，以及聚合数据。

Query：

```text
namespaceId
skillName 或 name
```

### POST /v3/admin/ai/skills/social/star

收藏 / 取消收藏。

请求：

```json
{"namespaceId":"public","skillName":"dialogue-card","starred":true}
```

### POST /v3/admin/ai/skills/social/rating

评分。

请求：

```json
{"namespaceId":"public","skillName":"dialogue-card","rating":5,"comment":"好用"}
```

### POST /v3/admin/ai/skills/social/subscribe

订阅 / 取消订阅。

请求：

```json
{"namespaceId":"public","skillName":"dialogue-card","subscribed":true}
```

---

## 9. Skill Group API

Group 是 namespace 内的一组相关 Skill，用于 Agent 一次性加载能力包。

### GET /v3/admin/ai/skill-groups/list

Group 列表。

### GET /v3/admin/ai/skill-groups

Group 详情。

Query：

```text
namespaceId
groupName
```

### POST /v3/admin/ai/skill-groups

创建 Group。

请求：

```json
{
  "namespaceId": "public",
  "groupName": "novel-suite",
  "displayName": "小说写作技能组",
  "description": "长篇小说写作相关技能",
  "members": [
    {"skillName":"dialogue-card","label":"stable","required":true,"order":1}
  ]
}
```

### PUT /v3/admin/ai/skill-groups

更新 Group。

### DELETE /v3/admin/ai/skill-groups

删除 Group。

### POST /v3/admin/ai/skill-groups/bind

绑定成员 Skill。

### DELETE /v3/admin/ai/skill-groups/bind

解绑成员 Skill。

---

## 10. Runtime Client API

Runtime API 面向 Agent，只读。

### GET /v3/client/ai/skills

按 name/version/label 下载 Skill ZIP。

Query：

```text
namespaceId 必填
name        必填
version     可选
label       可选，latest/stable/gray 等
md5         可选，客户端已有 md5
```

响应：

```text
200 返回 zip
304 Not Modified，当客户端 md5 与当前版本一致
```

响应头：

```http
ETag: <md5>
X-Nacos-Skill-Md5: <md5>
X-Nacos-Skill-Resolved-Version: <version>
```

### GET /v3/client/ai/skill-groups

获取 Group runtime manifest。

Query：

```text
namespaceId
groupName
label 可选
```

返回该 group 下各成员 Skill 解析后的版本、label、下载信息。

---

## 11. Registry / Well-known API

面向外部 SDK、CLI、Agent 的发现协议。

### GET /registry/{namespaceId}/api/search

搜索 Skill。

### GET /registry/{namespaceId}/.well-known/agent-skills/index.json

Agent Skills 标准索引。

### GET /registry/{namespaceId}/.well-known/skills/index.json

Legacy skills 索引。

### GET /registry/{namespaceId}/.well-known/agent-skills/{skillName}.zip

下载 Skill ZIP。

### GET /registry/{namespaceId}/.well-known/skills/{skillName}.zip

Legacy 下载路径。

### GET /registry/{namespaceId}/.well-known/agent-skills/{skillName}/SKILL.md

获取 SKILL.md。

### GET /registry/{namespaceId}/.well-known/agent-skills/{skillName}/**

获取 Skill 内部文本资源。

> `.well-known` 是外部发现协议，不是 Redis。Redis 只是服务内部生成该响应时的缓存层。

---

## 12. Agent Proposal / Governance Inbox

Agent 不能直接修改正式 Skill，只能提交 Proposal / Overlay，进入治理流程。

### POST /v3/agent/ai/skill-proposals

Agent 提交 Skill 修改建议。

请求：

```json
{
  "namespaceId": "public",
  "skillName": "dialogue-card",
  "baseVersion": "0.0.3",
  "proposalType": "delta",
  "source": {
    "agentId": "dialogue_skill_batch_worker",
    "sessionId": "sess_001",
    "runId": "run_001",
    "taskId": "task_001"
  },
  "reason": "发现缺少权力反转式对话规则",
  "delta": {
    "addSections": [
      {"title":"权力反转式对话","content":"弱势方通过证据、身份、情报完成话语权反转。"}
    ]
  },
  "evidence": {
    "inputRefs": ["chapter_011"],
    "score": 0.82
  }
}
```

### GET /v3/agent/ai/skill-proposals/{proposalId}

Agent 查询 Proposal。

### GET /v3/agent/ai/skill-overlays/{proposalId}

根据 Proposal 获取 overlay。

### GET /v3/agent/ai/skill-overlays?overlayRef=...

根据 overlayRef 获取 overlay。

### GET /v3/admin/ai/skill-proposals/list

Governance Inbox 列表。

### GET /v3/admin/ai/skill-proposals/{proposalId}

Proposal 详情。

### POST /v3/admin/ai/skill-proposals/{proposalId}/validate

验证 Proposal。当前为轻量校验，后续可接模型 eval / 回归测试。

### POST /v3/admin/ai/skill-proposals/{proposalId}/approve

批准 Proposal，并可生成 candidate/publish/online/label。

请求：

```json
{
  "publish": true,
  "online": true,
  "label": "gray",
  "operator": "admin"
}
```

### POST /v3/admin/ai/skill-proposals/{proposalId}/reject

拒绝 Proposal。

请求：

```json
{"reason":"样例不足，暂不采纳","operator":"admin"}
```

---

## 13. Audit Log

### GET /v3/admin/audit/logs

查询审计日志。

Query 可包含：

```text
namespaceId
type
name
action
operator
pageNo
pageSize
```

---

## 14. 权限建议

当前代码已经有 Principal、Auth Provider、Role Mapping、Namespace Member 等基础能力。建议权限动作如下：

```text
skill:read
skill:create
skill:update
skill:delete
skill:publish
skill:online
skill:offline
skill:label:update
skill:proposal:create
skill:proposal:read
skill:proposal:review
skill:proposal:approve
skill:proposal:reject
skill:overlay:read
skill:group:read
skill:group:create
skill:group:update
skill:group:delete
namespace:admin
namespace:member:update
iam:admin
iam:token:admin
audit:read
metrics:read
system:admin
```

Agent 默认建议只给：

```text
skill:read
skill:proposal:create
skill:proposal:read
skill:overlay:read
```

不要给 Agent：

```text
skill:publish
skill:delete
skill:label:update:stable
system:admin
```

---

## 15. 当前边界

已实现：

```text
Skill 管理、版本、文件树、版本对比、Runtime 下载、well-known、Group、Proposal、Namespace、Social、Audit、Token、Metrics、幂等、限流、Auth 模式、本地首次初始化。
```

待增强：

```text
1. Rate Limit 当前主要是本地内存，生产多副本建议改 Redis 全局限流。
2. Token 管理表已做，APIKeyProvider 读取 DB token 需要继续打通。
3. Audit 已有接口和表，但老 Skill 写接口建议统一接入 audit middleware。
4. Markdown 渲染建议引入 react-markdown + rehype-sanitize。
5. Diff 建议引入完整 hunk/inline diff 组件。
6. Subscribe 当前只保存订阅关系，通知中心/SSE 未完整实现。
7. Namespace 成员角色已存储，路由级 ABAC 强制校验还需继续加强。
8. Proposal validate 目前是轻量校验，还未接真实 eval/pipeline。
9. OIDC Authorization Code + PKCE 前端 callback 未完整实现。
10. OpenAPI 目前为手写规范，尚未自动生成 SDK。
```
