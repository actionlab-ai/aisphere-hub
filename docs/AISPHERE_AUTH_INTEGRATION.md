# SkillHub × aisphere-auth 集成说明

本文档说明 SkillHub 如何作为 AI Sphere 平台的一个子应用接入 `aisphere-auth`，
以及如何在不删除既有 local / Casdoor / Casbin 能力的前提下，**新增**一条
平台统一的 auth / authz / audit 通道。

## 设计原则

1. **不删除既有能力**：`local`、`oidc`、`jwt`、`api_key`、`introspection`、
   `casbin`、`casdoor-remote` 全部保留。`aisphereAuth.enabled=false` 时
   SkillHub 行为与改造前完全一致。
2. **纯加法**：新增 1 个 `auth provider`、1 个 `authz provider`、1 个
   `aisphereclient` 包、1 套 `/v3/auth/aisphere/*` 跳转路由、1 个前端按钮。
3. **可灰度**：可以只开 AuthN 不开 AuthZ，也可以同时开。两条通道互相不依赖。

## 改造点一览

### 后端新增

| 路径 | 作用 |
| --- | --- |
| `backend/internal/aisphereclient/client.go` | 封装 `pkg/aisphereauth/client.HTTPClient`，集中配置 endpoint / service token / cookie name，提供 introspect cache |
| `backend/internal/aisphereclient/cache.go` | introspect 结果的短 TTL 缓存，默认 5s |
| `backend/internal/auth/providers/aisphereauth/aisphereauth.go` | 新 `AuthProvider`，读取 `aisphere_session` cookie 并调 `/auth/sessions/introspect` |
| `backend/internal/authz/aisphereauth/authorizer.go` | 新 `Authorizer`，把 `(action, ResourceRef)` 映射为 `(sub, obj, act)` 调 aisphere-auth `/authz/check` |
| `backend/internal/authz/aisphereauth/access_routes.go` | Access 诊断页路由 `/v3/admin/access/aisphere/{overview,evaluate}` |

### 后端修改

| 路径 | 修改 |
| --- | --- |
| `backend/internal/config/config.go` | 新增 `AISphereConfig` 结构、默认值、env override (`AISPHERE_AUTH_*`)；`AuthzConfig.Provider` 注释加入 `aisphere-auth` |
| `backend/internal/authhttp/middleware.go` | 新增 `MiddlewareWithAISphere(...)`，把 `aisphereauth.Provider` 前置插入 auth chain；`RequiredPermission` 识别 `/v3/auth/aisphere/*` 为 `auth:login` |
| `backend/internal/authhttp/oidc_routes.go` | 新增 `RegisterAISphereRoutes`，挂载 `/v3/auth/aisphere/login` 和 `/v3/auth/aisphere/callback` |
| `backend/cmd/skillhub/main.go` | 构造 `aisphereclient.Client`，注入 middleware 和 authorizer；当 `authz.provider=aisphere-auth` 时切换 authorizer |
| `backend/go.mod` | 新增 `require github.com/actionlab-ai/aisphere-auth v0.1.0` 和本地 `replace` |
| `backend/configs/aisphere-auth.yaml` | 新示例配置，展示完整启用方式 |

### 前端修改

| 路径 | 修改 |
| --- | --- |
| `front/src/components/auth/login-page.tsx` | 新增 "Sign in with AI Sphere" 按钮（蓝色），位于 Casdoor 按钮之上，local fallback 表单保留 |
| `front/src/lib/api/client.ts` | 新增 `AISPHERE_AUTH_ENABLED` 开关 + `redirectToAISphereLogin()`；401 且无本地 token 时自动跳转 |
| `front/src/components/layout/app-shell.tsx` | `authed` 初始化时同时考虑 aisphere-auth 开关；`/v3/auth/me` 失败时若开启了 aisphere-auth 自动跳转登录 |

## 配置示例

最小启用方式见 `backend/configs/aisphere-auth.yaml`，关键片段：

```yaml
aisphereAuth:
  enabled: true
  endpoint: "http://127.0.0.1:18080"
  serviceToken: "${AISPHERE_SERVICE_TOKEN}"
  cookieName: "aisphere_session"
  app: "skillhub"
  cacheTTLSeconds: 5
  failClosed: true

authz:
  provider: "aisphere-auth"   # 想保留旧路径就改回 casdoor-remote / casbin / static
```

环境变量同样支持：

```bash
export AISPHERE_AUTH_ENABLED=true
export AISPHERE_AUTH_ENDPOINT=http://aisphere-auth:18080
export AISPHERE_SERVICE_TOKEN=<与 aisphere-auth 一致的长随机串>
# 可选：
export AISPHERE_SESSION_COOKIE_NAME=aisphere_session
export AISPHERE_AUTH_APP=skillhub
export AISPHERE_AUTH_FAIL_CLOSED=true
```

前端开关：

```bash
# front/.env.local
NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true
```

## 请求流转

### 登录

```
浏览器 → SkillHub 前端 "Sign in with AI Sphere"
  → /v3/auth/aisphere/login?redirect=<current>
  → (SkillHub 后端 302)
  → aisphere-auth /auth/login?app=skillhub&redirect=<current>
  → Casdoor 登录页 → 回调 aisphere-auth
  → aisphere-auth 下发 aisphere_session cookie，302 回 SkillHub
  → SkillHub 前端重新加载，AppShell 调 /v3/auth/me
  → SkillHub 后端 aisphereauth.Provider 读 cookie 调 introspect
  → 返回 Principal，AppShell 渲染主界面
```

### 鉴权

```
浏览器 → SkillHub 前端 → /v3/aihub/skills
  → SkillHub 后端 middleware:
    1. aisphereauth.Provider 读 aisphere_session cookie 调 introspect → Principal
    2. aisphereauth.Authorizer 把 (skill:admin:read, aihub:skill:*) 调 /authz/check
       → aisphere-auth 内部走 Casdoor enforce → 返回 allow
  → SkillHub 业务 handler 返回数据
```

### 失败回退

- aisphere-auth 不可达：`failClosed=true` 时拒绝；`false` 时回退到下一个
  authorizer（`static` / `casdoor-remote` / `casbin`）。
- aisphere_session cookie 不存在：`aisphereauth.Provider` 返回 `ok=false`，
  composite authenticator 透明地尝试下一个 provider（local / oidc / …）。
- 前端 401 且无本地 token：`AISPHERE_AUTH_ENABLED=true` 时自动跳转
  `/v3/auth/aisphere/login`；否则保持原行为（显示登录页）。

## 与既有能力的共存矩阵

| `aisphereAuth.enabled` | `authz.provider` | 行为 |
| --- | --- | --- |
| `false` | `casdoor-remote` / `casbin` / `static` | 与改造前完全一致 |
| `true` | `casdoor-remote` / `casbin` / `static` | AuthN 走 aisphere-auth，AuthZ 走原路径（适合 aisphere-auth 还没接 enforce 的过渡期） |
| `true` | `aisphere-auth` | AuthN + AuthZ 全走 aisphere-auth（推荐生产配置） |
| `false` | `aisphere-auth` | SkillHub 启动时打印 warn，回退到 `static` |

## 迁移路径

1. **第 0 步**：部署 aisphere-auth，跑通其自己的 Casdoor 链路。
2. **第 1 步**：在 SkillHub 配置 `aisphereAuth.enabled=true` 但保持
   `authz.provider=casdoor-remote`。验证 "Sign in with AI Sphere" 按钮可登录，
   `/v3/auth/me` 能拿到 Principal，权限仍由 Casdoor 直接决定。
3. **第 2 步**：在 aisphere-auth 的 Casdoor 里配置与 SkillHub 相同的 permission
   / model / policy。在 SkillHub 上把 `authz.provider` 改成 `aisphere-auth`，
   观察 `/v3/admin/access/aisphere/overview` 诊断页。
4. **第 3 步**：稳定后可考虑关掉 SkillHub 的 Casdoor OIDC provider
   (`auth.providers=[]`)，只保留 aisphere-auth 通道。

## 已知限制

- SkillHub 当前会同时保留本地 audit log，并在 `aisphereAuth.enabled=true` 时把成功的写操作镜像到 aisphere-auth `/audit/events`。如果 aisphere-auth 暂时不可用，审计镜像为 best-effort，不阻断业务请求。
- `go.mod` 直接依赖 `github.com/actionlab-ai/aisphere-auth v0.1.0`。发布前需要确保 aisphere-auth 已经打对应 tag；否则请先在 aisphere-auth 仓库发布 `v0.1.0`。
- introspect cache 默认 5s，意味着 aisphere-auth 上 session 注销后 SkillHub
  最长 5s 内仍可能放行请求。需要更严格可设为 0 关闭缓存。
