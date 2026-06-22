# Changelog

## [Unreleased] — aisphere-auth integration

### Added — 后端

- **`backend/internal/aisphereclient/`** (新包)
  - `client.go`：封装 `github.com/actionlab-ai/aisphere-auth/pkg/aisphereauth/client.HTTPClient`，集中配置 `endpoint` / `serviceToken` / `cookieName` / `app` / 超时 / 缓存 TTL，统一暴露 `Introspect` / `Check` / `BatchCheck`。
  - `cache.go`：5s TTL 的 introspect 缓存，避免每次请求都打 aisphere-auth。
- **`backend/internal/auth/providers/aisphereauth/`** (新包)
  - `aisphereauth.go`：新的 `core.AuthProvider`，读 `aisphere_session` cookie → 调 `/auth/sessions/introspect` → 映射为 SkillHub 的 `auth.Principal`。失败时返回 `ok=false`，让 composite authenticator 透明地走下一个 provider。
- **`backend/internal/authz/aisphereauth/`** (新包)
  - `authorizer.go`：新的 `core.Authorizer`，把 SkillHub 的 `(action, ResourceRef)` 映射为 `(sub, obj, act)` 调 aisphere-auth `/authz/check`。`objectFor` 复用 casdoor-remote 的命名约定，policy 可移植。
  - `access_routes.go`：诊断页路由 `/v3/admin/access/aisphere/{overview,evaluate}`，与既有 casdoor-remote Access 页风格一致。
- **`backend/configs/aisphere-auth.yaml`** (新文件)：完整启用 aisphere-auth 的示例配置。
- **`backend/internal/authhttp/oidc_routes.go`**：新增 `RegisterAISphereRoutes(r, client)`，挂载：
  - `GET /v3/auth/aisphere/login?redirect=...` → 302 到 aisphere-auth `/auth/login?app=skillhub&redirect=...`
  - `GET /v3/auth/aisphere/callback` → 简单 landing 页，刷新回 SkillHub
- **`backend/internal/authhttp/middleware.go`**：新增 `MiddlewareWithAISphere(cfg, localMgr, st, authz, aisphereClient)`，把 `aisphereauth.Provider` **前置**插入 auth chain；`RequiredPermission` 识别 `/v3/auth/aisphere/*` 为 `auth:login`。

### Added — 前端

- **`front/src/components/auth/login-page.tsx`**：新增 "Sign in with AI Sphere" 按钮（蓝色 indigo→blue 渐变，位于 Casdoor 按钮之上）。点击后跳转 `/v3/auth/aisphere/login?redirect=<current>`，由后端 302 到 aisphere-auth。
- **`front/src/lib/api/client.ts`**：新增 `AISPHERE_AUTH_ENABLED` 环境开关 + `redirectToAISphereLogin()` helper。当 API 返回 401 且本地无 bearer token 且开关开启时，自动跳转到 aisphere-auth 重新建立 session cookie。
- **`front/src/components/layout/app-shell.tsx`**：`authed` 初始化时同时考虑 aisphere-auth 开关；`/v3/auth/me` 失败时若开关开启且无本地 token，自动跳转 aisphere-auth 登录。

### Added — 文档

- **`docs/AISPHERE_AUTH_INTEGRATION.md`**：完整集成说明，包括设计原则、改造点一览、配置示例、请求流转、共存矩阵、迁移路径、已知限制。
- **`CHANGELOG.md`**：本文件。
- **`.gitignore`**：覆盖 Go / Node / IDE / OS / 本地数据等忽略项。

### Changed — 后端

- **`backend/internal/config/config.go`**
  - 新增 `AISphereConfig` 结构体（`Enabled` / `Endpoint` / `ServiceToken` / `ServiceTokenHeader` / `CookieName` / `App` / `HTTPTimeoutSeconds` / `CacheTTLSeconds` / `FailClosed`）。
  - `Config` 根结构新增 `AISphere AISphereConfig yaml:"aisphereAuth"` 字段。
  - `Default()` 加入 aisphere-auth 的默认值（`enabled=false`，`endpoint=http://aisphere-auth:18080`，`app=skillhub`，`cacheTTLSeconds=5`，`failClosed=true`）。
  - `applyEnvOverrides()` 加入 `AISPHERE_AUTH_*` 环境变量绑定。
  - `AuthzConfig.Provider` 注释加入 `aisphere-auth` 选项。
- **`backend/internal/authhttp/middleware.go`**
  - `MiddlewareWithLocalStoreAuthorizer` 改为委托给 `MiddlewareWithAISphere(..., nil)`，保持向后兼容。
  - `BuildProviders` 对 `aisphere-auth` / `aisphere_auth` / `aisphereauth` 类型显式跳过（避免重复注册），真正的 provider 由 `MiddlewareWithAISphere` 注入。
- **`backend/cmd/skillhub/main.go`**
  - import `aisphereclient` 和 `aisphereauthz`。
  - main() 中构造 `aisphereclient.Client`，注入 `MiddlewareWithAISphere`。
  - 当 `authz.provider=aisphere-auth` 时，用 `initAISphereAuthorizer` 替换默认 authorizer。
  - 启动日志加入 `aisphereAuth=%v` 字段。
  - `initAuthorizer` switch 新增 `aisphere-auth` case（返回 nil，由 main() 后续替换）。
- **`backend/go.mod`**
  - `require` 加入 `github.com/actionlab-ai/aisphere-auth v0.1.0`。
  - 末尾加入 `replace github.com/actionlab-ai/aisphere-auth => ../../aisphere-auth`（本地开发用，发布前删除或换成真实 tag）。

### NOT removed (重要：保留的既有能力)

为避免破坏既有部署，下列能力**全部保留**，未做任何删除：

- `backend/internal/auth/providers/local/` — 本地账号 + JWT 签发
- `backend/internal/auth/providers/oidc/` — Casdoor / 通用 OIDC 登录
- `backend/internal/auth/providers/jwt/` — 外部 JWT 验证（JWKS / 公钥）
- `backend/internal/auth/providers/apikey/` — 静态 API Key
- `backend/internal/auth/providers/dbapikey/` — 数据库 API Key
- `backend/internal/auth/providers/introspection/` — OAuth2 token introspection
- `backend/internal/authz/casbin/` — 内嵌 Casbin 授权
- `backend/internal/authz/casdoorremote/` — Casdoor /api/enforce 远程授权
- `backend/internal/auth/abac.go` — store-backed ABAC
- `backend/internal/auth/authorizer.go` — StaticAuthorizer
- `backend/internal/auth/composite.go` — CompositeAuthenticator
- `backend/internal/auth/mapper.go` — StaticMapper (roleMappings)
- `backend/internal/authhttp/routes.go` — local auth 路由（/v3/auth/login、/v3/auth/setup 等）
- `backend/internal/authhttp/oidc_routes.go` 既有 OIDC login/callback 路由
- `backend/internal/ops/middleware.go` 既有 `AuditMiddleware` / 限流 / 幂等
- 前端 `LoginPage` 的 "Sign in with Casdoor" 按钮和 local fallback 表单
- 前端 `SetupPage` 首次初始化流程
- 前端 `AuthCallbackPage` Casdoor 回调页

### 配置开关矩阵

| `aisphereAuth.enabled` | `authz.provider` | AuthN 来源 | AuthZ 来源 |
| --- | --- | --- | --- |
| `false` | `casdoor-remote` / `casbin` / `static` | 既有 chain | 既有 chain（**与改造前完全一致**） |
| `true` | `casdoor-remote` / `casbin` / `static` | aisphere-auth 优先，失败回退既有 chain | 既有 chain |
| `true` | `aisphere-auth` | aisphere-auth 优先，失败回退既有 chain | aisphere-auth `/authz/check` |
| `false` | `aisphere-auth` | 既有 chain | static fallback（启动时 warn） |

### 部署影响

- **零侵入**：现有部署不需要修改任何配置即可继续运行（`aisphereAuth.enabled` 默认 `false`）。
- **可灰度**：先开 AuthN 验证登录链路，再切 AuthZ，可分两步上线。
- **可回退**：任何阶段把 `aisphereAuth.enabled` 改回 `false` 即恢复原状，不需要回滚代码。

### 已知限制

- aisphere-auth 当前未打 git tag，`go.mod` 用 `replace` 指向本地路径。aisphere-auth 发 v0.1.0 后删 `replace` 行即可。
- aisphere-auth 的 audit 服务目前是接口预留，SkillHub 这边 audit 仍走本地 `ops.AuditMiddleware`。等 aisphere-auth 暴露 audit ingest API 后再增加 forward sink。
- introspect cache 默认 5s，意味着 aisphere-auth 上 session 注销后 SkillHub 最长 5s 内仍可能放行请求。需要更严格可设 `cacheTTLSeconds: 0` 关闭缓存。
- 前端 `AISPHERE_AUTH_ENABLED` 是构建期 env，切换需要重新构建前端。
