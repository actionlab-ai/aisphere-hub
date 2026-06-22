# AIHub Console 前端适配说明

## 背景

本前端参考 Nacos `console-ui-next` 中 Skill 管理模块的功能组织方式，包括：

- Skill 列表 / 搜索 / 上传
- Skill 详情 / 版本 / label / 上下线 / 下载
- Skill Group 能力包管理
- Agent Proposal / Overlay 审核入口
- 本地账号 / Agent 账号管理
- 首次初始化管理员

改造目标不是保留 Nacos 控制台，而是把 Skill 相关管理能力迁移为一个独立的 Go AIHub 管理台。

## 已去除的 Nacos 依赖

- 去掉 Nacos Logo、Nacos 文案、Nacos 控制台菜单。
- 不再调用 `/v3/console/ai/skills`，统一调用 Go AIHub 的 `/v3/admin/ai/skills`。
- 不依赖 Nacos namespace 页面、用户页面、配置管理页面、服务发现页面。
- 不依赖 Nacos console cookie/session，改为 AIHub 本地 JWT / 外部 Bearer Token。

## 前端目录

```text
../front/
  package.json
  vite.config.ts
  src/
    main.tsx
    api/
      client.ts
      index.ts
      types.ts
    pages/
      SkillsPage.tsx
      GroupsPage.tsx
      ProposalsPage.tsx
      IamPage.tsx
      DocsPage.tsx
    styles/app.css
```

## 后端静态资源配置

后端新增 `web` 配置：

```yaml
web:
  enabled: true
  dir: "../front/dist"
  routePrefix: "/ui"
  indexFallback: true
```

构建后访问：

```text
http://127.0.0.1:8848/ui/
```

## 开发启动

前端开发：

```bash
cd ../front
npm install
npm run dev
```

Vite 会把 `/v3`、`/registry`、`/healthz` 代理到 `http://127.0.0.1:8848`。

后端开发：

```bash
go run ./cmd/skillhub --config configs/local-first-setup.yaml
```

## 生产构建

```bash
cd ../front
npm install
npm run build
cd ../..
go run ./cmd/skillhub --config configs/mysql-redis-minio.yaml
```

## 页面能力

### 1. 首次初始化

前端启动后会先调用：

```http
GET /v3/auth/setup/status
```

如果 `setupRequired=true`，展示首次初始化管理员页面，并调用：

```http
POST /v3/auth/setup
```

### 2. 登录

内置账号模式下调用：

```http
POST /v3/auth/login
```

登录成功后把 `accessToken` 保存到 `localStorage`，后续请求带：

```http
Authorization: Bearer <token>
```

### 3. Skill 管理

主要接口：

```http
GET    /v3/admin/ai/skills/list
GET    /v3/admin/ai/skills
POST   /v3/admin/ai/skills/upload
POST   /v3/admin/ai/skills/submit
POST   /v3/admin/ai/skills/publish
POST   /v3/admin/ai/skills/online
POST   /v3/admin/ai/skills/offline
PUT    /v3/admin/ai/skills/labels
DELETE /v3/admin/ai/skills
```

### 4. Skill Group

主要接口：

```http
GET    /v3/admin/ai/skill-groups/list
POST   /v3/admin/ai/skill-groups
POST   /v3/admin/ai/skill-groups/bind
DELETE /v3/admin/ai/skill-groups/bind
```

### 5. Agent Proposal

主要接口：

```http
GET  /v3/admin/ai/skill-proposals/list
POST /v3/admin/ai/skill-proposals/{proposalId}/validate
POST /v3/admin/ai/skill-proposals/{proposalId}/approve
POST /v3/admin/ai/skill-proposals/{proposalId}/reject
```

### 6. IAM 本地账号

主要接口：

```http
GET    /v3/admin/iam/local-users/list
POST   /v3/admin/iam/local-users
PUT    /v3/admin/iam/local-users/{username}
DELETE /v3/admin/iam/local-users/{username}
```

## 当前边界

已实现：

- 独立 AIHub Console。
- Nacos 品牌清理。
- 首次初始化、登录、Token 保存。
- Skill、Group、Proposal、IAM 页面。
- Go 后端 `/ui` 静态资源服务。

未实现 / 后续增强：

- 复杂 Markdown 编辑器、Monaco Diff、Nacos 原版高级交互未完整迁移。
- OIDC Authorization Code + Callback 前端页面未做；目前支持外部 Bearer Token 透传和本地登录。
- 权限按钮级隐藏还比较基础，后续可根据 `/v3/auth/me` 的 permissions 动态控制。
- 尚未生成 OpenAPI 驱动的 API Client。
- 尚未做前端单元测试和 E2E。
