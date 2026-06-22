# SkillHub 前端整合说明（Next Console）

## 1. 本次整合结论

本版采用用户提供的 Next.js + Tailwind + shadcn 风格前端作为主 Console，并与 group-first / namespace-free 后端接口对齐。

项目结构：

```text
skillhub/
  backend/   Gin 后端，提供 SkillHub API、鉴权、缓存、存储、/ui 静态托管
  front/     Next.js 前端 Console，纯管理台，不再内置 mock API
  docs/      接口、设计、整合说明
```

## 2. Mock 数据清理

已清理：

- 删除 `front/src/app/api/route.ts`，不再提供 `Hello world` mock API。
- 删除 `front/src/lib/db.ts`、`prisma/`、`db/`、`upload/`、`tool-results/` 等和 Console 无关的本地 mock/临时目录。
- 删除页面里的 `preview-user` fallback。现在 `/v3/auth/me` 失败会清理 token 并回到登录页，不再伪造用户。
- Skill / Group / Governance / Ops 等页面均调用真实后端 `/v3/**` 接口。

## 3. 接口适配

前端已经使用新的 group-first canonical API：

```text
GET    /v3/aihub/skills
POST   /v3/aihub/skills/upload
GET    /v3/aihub/skill/{skillName}
GET    /v3/aihub/skill/{skillName}/versions/{version}/files
GET    /v3/aihub/skill/{skillName}/versions/{version}/file
GET    /v3/aihub/skill/{skillName}/compare

GET    /v3/aihub/groups
POST   /v3/aihub/groups
GET    /v3/aihub/group/{groupName}
POST   /v3/aihub/group/{groupName}/skills
DELETE /v3/aihub/group/{groupName}/skills/{skillName}
```

Skill 页面只按 `groupName` 过滤，不再把 namespace 当 Skill 分类维度。

Access Space 页面仍然调用 legacy endpoint：

```text
/v3/admin/namespaces
```

但 UI 语义已经改成 `Access Spaces`，只用于 IAM/RBAC，不参与 Skill 分组。

## 4. 前端路由转发 / 网关怎么处理

### 4.1 删除了不必要的前端 API 网关

本版没有保留 Next `src/app/api/**` 作为业务网关。原因：

- SkillHub 后端已经是 API 服务，不需要前端再包一层 API route。
- 前端 API route 容易产生 mock、鉴权、错误处理两套逻辑。
- 生产部署更推荐同域反向代理或 Gin 直接托管静态前端。

### 4.2 保留 Next rewrites 作为“开发期代理”

`front/next.config.ts` 里保留了 rewrites：

```text
/v3/**              -> SKILLHUB_BACKEND_URL/v3/**
/registry-global/** -> SKILLHUB_BACKEND_URL/registry-global/**
/registry/**        -> SKILLHUB_BACKEND_URL/registry/**
/metrics            -> SKILLHUB_BACKEND_URL/metrics
/healthz            -> SKILLHUB_BACKEND_URL/healthz
```

这不是业务网关，只是开发期避免 CORS 的代理。开发时：

```powershell
cd backend
go run .\cmd\skillhub\main.go --config .\config.example.yaml

cd ..\front
$env:SKILLHUB_BACKEND_URL="http://127.0.0.1:8848"
npm install
npm run dev
```

访问：

```text
http://127.0.0.1:3000
```

### 4.3 生产推荐两种方式

方式 A：Gin 托管静态前端：

```powershell
cd front
npm install
npm run build

cd ..\backend
go run .\cmd\skillhub\main.go --config .\config.example.yaml
```

`next build` 生成 `front/out`，后端配置：

```yaml
web:
  enabled: true
  dir: "../front/out"
  routePrefix: "/ui"
  indexFallback: true
```

访问：

```text
http://127.0.0.1:8848/ui/
```

方式 B：前端和后端独立部署，由 Nginx / Caddy / Ingress 做同域转发：

```text
/           -> front static/next
/v3         -> backend:8848
/registry   -> backend:8848
/registry-global -> backend:8848
/metrics    -> backend:8848
```

## 5. Namespace / Access Space 说明

本版不再把 namespace 作为 Skill 维度。

- `Group`：Skill 业务分组、能力包、Agent 加载单元。
- `Skill`：具体技能，全局唯一 name。
- `Access Space`：账号权限域，底层仍复用 `/v3/admin/namespaces` legacy API。

前端里所有 Skill 查询都不传 `namespaceId`。只有 Access/IAM、Token、成员管理场景会出现 `Access Space`。

## 6. 当前边界

- Next 前端已改为静态 export，生产不需要 Next Server。
- rewrites 只在 `next dev` 或 Next server 模式生效；静态 export 后由 Gin/Nginx 处理同域路由。
- 前端没有 mock API；如果后端未启动，请求会直接失败并提示。
- Access Space endpoint 名字仍是 `/namespaces`，这是后端兼容层遗留，UI 层不再叫 Namespace。
