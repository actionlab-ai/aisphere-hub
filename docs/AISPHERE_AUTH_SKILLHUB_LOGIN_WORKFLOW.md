# SkillHub 接入 aisphere-auth 登录链路排障与正确工作流

本文记录一次从 SkillHub 本地前端接入 `aisphere-auth` + Casdoor 的完整排障过程，并沉淀当前可用的本地联调配置、真实请求链路和后续工程化建议。

适用范围：

- SkillHub 前端：`aisphere-hub/front`
- SkillHub 后端：`aisphere-hub/backend`
- 统一认证服务：`aisphere-auth`
- 身份源和权限中心：Casdoor

## 1. 最终结论

本次链路已经验证通过：

```text
http://localhost:3000/
  -> Sign in with AI Sphere
  -> SkillHub /v3/auth/aisphere/login
  -> aisphere-auth /auth/login
  -> Casdoor 登录
  -> aisphere-auth /auth/callback/casdoor
  -> Set-Cookie: aisphere_session
  -> redirect 回 http://localhost:3000/
  -> SkillHub /v3/auth/me 返回 test1
  -> /v3/admin/notifications 返回 200
  -> /v3/aihub/skills 返回 200
```

最终验证结果：

```text
/v3/auth/me                         200
/v3/admin/notifications?pageNo=1... 200
/v3/aihub/skills?pageNo=1...     200
aisphere_session cookie domain      localhost
```

核心结论：

1. SkillHub 不应该直接拼 Casdoor `/login/oauth/authorize`。
2. SkillHub 应该先跳到 `aisphere-auth /auth/login`，由 `aisphere-auth` 生成 OAuth `state`。
3. 本地联调时前端、SkillHub 后端代理、`aisphere-auth` 回调域名必须统一使用 `localhost`，不要混用 `127.0.0.1` 和 `localhost`。
4. `aisphere-auth` 调 Casdoor `/api/enforce` 必须带 `client_credentials` 换来的 Bearer token。
5. 前端 AI Sphere 登录模式不能依赖 localStorage token，必须允许 HttpOnly cookie 模式下调用 `/v3/auth/me`。

## 2. 本次遇到的问题

### 2.1 导入 SQL 后新增用户页面空白

现象：

```text
Casdoor 用户管理 -> 新增用户 页面空白或无法保存
```

原因：

初始化 SQL 中组织、用户、表单字段等默认值不完整，Casdoor 页面读取字段配置时遇到空值。

处理：

- 修复 `aisphere-auth/scripts/casdoor/render-casdoor-seed.py`
- 修复 `aisphere-auth/scripts/casdoor/bootstrap-casdoor-mysql.py`
- 补充测试 `aisphere-auth/tests/casdoor/test_casdoor_seed.py`

原则：

初始化脚本必须生成一套能直接打开 Casdoor 控制台、能新增用户、能登录的基线配置，不能依赖页面手工补字段。

### 2.2 `GetOwnerAndNameFromId() error, wrong token count`

现象：

```text
GetOwnerAndNameFromId() error, wrong token count for ID: aisphere-auth-model
```

原因：

Casdoor 多数字段要求 `owner/name` 格式。之前 permission 的 model 写成了：

```text
aisphere-auth-model
```

正确值应该是：

```text
aisphere/aisphere-auth-model
```

处理：

- 修复初始化脚本里 permission.model 的 owner 前缀。
- 修复线上 Casdoor DB 中已有 permission 的 model 值。

### 2.3 `policy_definition must be permissionId`

现象：

```text
when adding policies with permissions, the sixth field of policy_definition must be permissionId, got permission
```

原因：

Casdoor 对 permission policy 的第六个字段名有硬编码要求，必须叫 `permissionId`。之前模型写成：

```text
p = sub, obj, act, eft, unused, permission
```

正确写法：

```text
p = sub, obj, act, eft, unused, permissionId
```

处理：

- 修复 `MODEL_TEXT`
- 修复 bootstrap 校验
- 修复 repair SQL
- 更新线上 Casdoor model

当前模型关键片段：

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft, unused, permissionId

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub) || r.sub == p.sub || keyMatch(r.sub, p.sub)) && (p.obj == "*" || keyMatch(r.obj, p.obj)) && (p.act == "*" || r.act == p.act)
```

### 2.4 直接打开 Casdoor authorize URL 导致 callback 失败

现象：

```json
{
  "error": {
    "code": "auth_callback_failed",
    "message": "登录回调校验失败"
  }
}
```

原因：

`aisphere-auth` 的 callback 第一件事是消费它自己生成并写入 Redis 的 `state`：

```text
aisphere:auth:state:<state>
```

如果直接打开 Casdoor URL：

```text
http://CHANGE_ME_HOST:8008/login/oauth/authorize?...&state=xxx
```

这个 `state` 可能不是 `aisphere-auth` 生成的，或者已经被消费/过期，callback 就会失败。

正确入口：

```text
http://localhost:18080/auth/login?app=skillhub&redirect=http%3A%2F%2Flocalhost%3A3000%2F
```

SkillHub 提供包装入口：

```text
http://127.0.0.1:8848/v3/auth/aisphere/login?redirect=http%3A%2F%2Flocalhost%3A3000%2F
```

### 2.5 SkillHub 回跳到 `/`，没有回到 `localhost:3000`

现象：

登录成功后 callback 返回：

```text
Location: /
```

导致浏览器落到：

```text
http://127.0.0.1:18080/
```

然后看到：

```text
404 page not found
```

原因：

`aisphere-auth` 原来的 `normalizeRedirect()` 只允许相对路径，传入的：

```text
http://localhost:3000/
```

会被当作不安全绝对 URL 丢弃，回退成 `/`。

处理：

本地开发场景允许以下绝对地址：

- `localhost`
- `127.0.0.1`
- `::1`
- 私有网段 IP

外部公网绝对地址仍然拒绝。

对应测试：

```go
TestNormalizeRedirectAllowsLocalhostDevelopmentURL
TestNormalizeRedirectRejectsNonLocalAbsoluteURL
```

### 2.6 Cookie 域名混用导致前端仍 401

现象：

callback 成功，响应里也有：

```text
Set-Cookie: aisphere_session=...
```

但前端请求仍然：

```text
/v3/admin/notifications 401
```

原因：

callback 原来发生在：

```text
http://127.0.0.1:18080/auth/callback/casdoor
```

前端访问的是：

```text
http://localhost:3000/
```

浏览器认为 `127.0.0.1` 和 `localhost` 是两个不同 host。`127.0.0.1` 下发的 HttpOnly cookie 不会发给 `localhost:3000`。

处理：

本地联调统一使用 `localhost`：

```yaml
# aisphere-auth/config.yaml
server:
  publicBaseURL: "http://localhost:18080"

casdoor:
  redirectURL: "http://localhost:18080/auth/callback/casdoor"
```

```yaml
# aisphere-hub/backend/config.yaml
aisphereAuth:
  endpoint: "http://localhost:18080"
```

```env
# aisphere-hub/front/.env.local
SKILLHUB_BACKEND_URL=http://localhost:8848
NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true
```

同时 Casdoor Application 需要允许：

```text
http://localhost:18080/auth/callback/casdoor
```

可以保留旧的：

```text
http://127.0.0.1:18080/auth/callback/casdoor
```

但一次登录链路中不要混用两个 host。

### 2.7 登录后从 401 变成 403

现象：

统一 `localhost` 后，cookie 已经带上，错误从 401 变为 403：

```text
/v3/admin/notifications 403
```

含义：

- 401 表示没有识别出登录态。
- 403 表示已经识别出用户，但权限判断拒绝。

原因：

`test1` 已在 Casdoor 角色：

```text
aisphere/role_skillhub_admin
```

但 `aisphere-auth/config.yaml` 当时使用的是：

```yaml
casdoor:
  permissionId: "aisphere/perm_platform_admin"
```

该 permission 只认：

```text
aisphere/role_platform_admin
```

不认 `role_skillhub_admin`。

处理：

本地 SkillHub 联调用：

```yaml
casdoor:
  permissionId: "aisphere/perm_skillhub_admin"
```

后续如果一个 `aisphere-auth` 实例要同时服务多个业务系统，应演进为按 `app` 选择不同 `permissionId`，而不是全局只有一个 `casdoor.permissionId`。

### 2.8 Casdoor `/api/enforce` 返回 `Unauthorized operation`

现象：

auth 模块调用 Casdoor：

```text
POST /api/enforce?permissionId=aisphere/perm_skillhub_admin
```

返回：

```json
{
  "status": "error",
  "msg": "Unauthorized operation"
}
```

原因：

Casdoor `/api/enforce` 需要带访问凭证。裸调会被 Casdoor 认为是未授权操作。

验证：

先用 `client_credentials` 换 token：

```text
POST /api/login/oauth/access_token
grant_type=client_credentials
client_id=aisphere-auth
client_secret=<client secret>
```

再带 Bearer token 调 enforce：

```text
Authorization: Bearer <access_token>
```

返回：

```json
{
  "status": "ok",
  "data": [true]
}
```

处理：

`aisphere-auth/internal/casdoor/http_client.go` 已修复：

- enforce 前自动获取并缓存 `client_credentials` token。
- enforce 请求带 `Authorization: Bearer <token>`。
- 兼容 Casdoor 返回 `data: [true]` 的结构。

对应测试：

```go
TestEnforceUsesClientCredentialsBearerToken
```

### 2.9 后端已登录，前端仍显示登录页

现象：

接口已经通：

```text
/v3/auth/me             200
/v3/aihub/skills     200
```

但页面仍停在登录页。

原因：

前端没有启用：

```env
NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true
```

AI Sphere 登录是 HttpOnly cookie 模式，不会把 token 写进 localStorage。如果前端仍然只根据 localStorage token 判断登录态，就会认为未登录。

处理：

新增：

```env
# aisphere-hub/front/.env.local
NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true
```

并重启 Next dev server。

## 3. 当前真实工作流程

### 3.1 登录流程

```text
用户打开 http://localhost:3000/
  |
  | 点击 Sign in with AI Sphere
  v
SkillHub 前端请求:
  GET /v3/auth/aisphere/login?redirect=http%3A%2F%2Flocalhost%3A3000%2F
  |
  v
SkillHub 后端 302:
  http://localhost:18080/auth/login?app=skillhub&redirect=http%3A%2F%2Flocalhost%3A3000%2F
  |
  v
aisphere-auth:
  1. 生成 state
  2. 写 Redis: aisphere:auth:state:<state>
  3. 302 到 Casdoor /login/oauth/authorize
  |
  v
Casdoor:
  用户输入 <test-user> / <test-password>
  登录成功后携带 code + state 回调
  |
  v
aisphere-auth /auth/callback/casdoor:
  1. 消费 Redis state
  2. 用 code 换 token
  3. 调 /api/userinfo
  4. 创建 aisphere-auth session
  5. Set-Cookie: aisphere_session=...
  6. 302 回 http://localhost:3000/
  |
  v
SkillHub 前端重新加载:
  1. 调 /v3/auth/me
  2. 后端读取 aisphere_session
  3. 后端调 aisphere-auth /auth/sessions/introspect
  4. 得到 Principal
  5. 进入控制台
```

### 3.2 鉴权流程

以请求技能列表为例：

```text
GET http://localhost:3000/v3/aihub/skills?pageNo=1&pageSize=80
  |
  v
Next dev rewrite:
  http://localhost:8848/v3/aihub/skills?pageNo=1&pageSize=80
  |
  v
SkillHub middleware:
  1. 从请求 cookie 读 aisphere_session
  2. 调 aisphere-auth /auth/sessions/introspect
  3. 得到 Principal:
     - provider: aisphere-auth
     - username: test1
     - casdoorSubject: aisphere/test1
     - roles: role_skillhub_admin
  4. RequiredPermission 映射为 skill:admin:read
  5. aisphere-auth authorizer 映射为:
     - sub: aisphere/test1
     - obj: skillhub:aihub:skill:*
     - act: read
  6. 调 aisphere-auth /authz/check
  |
  v
aisphere-auth /authz/check:
  1. 使用 casdoor.permissionId = aisphere/perm_skillhub_admin
  2. 获取/复用 client_credentials token
  3. 调 Casdoor /api/enforce
  |
  v
Casdoor:
  role_skillhub_admin -> perm_skillhub_admin -> allow
  |
  v
SkillHub handler 返回业务数据
```

## 4. 当前本地正确配置

### 4.1 aisphere-auth

文件：

```text
aisphere-auth/config.yaml
```

关键配置：

```yaml
server:
  addr: ":18080"
  publicBaseURL: "http://localhost:18080"

gateway:
  cookieDomain: ""
  cookieSecure: false
  cookieSameSite: "Lax"

casdoor:
  endpoint: "http://CHANGE_ME_HOST:8008"
  owner: "aisphere"
  application: "aisphere-auth"
  clientId: "aisphere-auth"
  redirectURL: "http://localhost:18080/auth/callback/casdoor"
  permissionId: "aisphere/perm_skillhub_admin"

session:
  provider: "redis"
  redis:
    addrs:
      - "CHANGE_ME_HOST:30011"
    prefix: "aisphere"

internal:
  serviceTokenRequired: true
```

注意：

- `clientSecret` 必须与 Casdoor Application 中的 client secret 一致。
- `serviceToken` 必须与 SkillHub `aisphereAuth.serviceToken` 一致。
- 本地联调使用 `localhost`，不要在同一条链路中混用 `127.0.0.1`。

### 4.2 SkillHub 后端

文件：

```text
aisphere-hub/backend/config.yaml
```

关键配置：

```yaml
auth:
  enabled: true
  mode: "external"
  allowAnonymous: false
  allowPublicRead: true
  providers: []
  roleMappings: []

authz:
  provider: "aisphere-auth"

aisphereAuth:
  enabled: true
  endpoint: "http://localhost:18080"
  serviceToken: "dev-aisphere-service-token-please-change-32chars"
  cookieName: "aisphere_session"
  app: "skillhub"
  cacheTTLSeconds: 5
  failClosed: true
```

说明：

- `auth.providers: []` 表示不再走 SkillHub 自己的 OIDC provider。
- `aisphereAuth.enabled: true` 会把 `aisphere-auth` provider 插入 SkillHub auth chain。
- `authz.provider: aisphere-auth` 会把权限判断委托给 `aisphere-auth /authz/check`。

### 4.3 SkillHub 前端

文件：

```text
aisphere-hub/front/.env.local
```

内容：

```env
SKILLHUB_BACKEND_URL=http://localhost:8848
NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true
NEXT_PUBLIC_AUTH_CALLBACK_PATH=/auth/callback
NEXT_PUBLIC_AUTH_REDIRECT_AFTER_LOGIN=/
```

说明：

- `SKILLHUB_BACKEND_URL` 控制 Next dev rewrites。
- `NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true` 允许前端用 HttpOnly cookie 模式识别登录态。
- 修改 `.env.local` 后必须重启 `npm run dev`。

### 4.4 Casdoor

Application：

```text
owner: admin
name: aisphere-auth
client_id: aisphere-auth
organization: aisphere
```

Redirect URIs 至少包含：

```text
http://localhost:18080/auth/callback/casdoor
```

可同时保留：

```text
http://127.0.0.1:18080/auth/callback/casdoor
```

但前端联调推荐全链路使用 `localhost`。

权限相关：

```text
permission: aisphere/perm_skillhub_admin
role:       aisphere/role_skillhub_admin
user:       aisphere/test1
```

`test1` 应属于：

```text
role_skillhub_admin
```

`perm_skillhub_admin` 应覆盖：

```text
resources: ["skillhub:*"]
actions:   ["*"]
roles:     ["aisphere/role_skillhub_admin"]
```

## 5. 正确启动顺序

### 5.1 启动 aisphere-auth

```powershell
cd E:\coding\adk\aisphere\aisphere-auth
go run .\cmd\server\main.go --config .\config.yaml
```

验证：

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:18080/readyz
```

期望：

```json
{
  "status": "ok",
  "checks": {
    "casdoor": "ok",
    "session": "redis",
    "state": "redis"
  }
}
```

### 5.2 启动 SkillHub 后端

```powershell
cd E:\coding\adk\aisphere\aisphere-hub\backend
go run .\cmd\skillhub\main.go --config .\config.yaml
```

启动日志应包含：

```text
aisphere-auth integration enabled: endpoint=http://localhost:18080 app=skillhub cookie=aisphere_session
gin skillhub listening on :8848
```

### 5.3 启动 SkillHub 前端

```powershell
cd E:\coding\adk\aisphere\aisphere-hub\front
npm run dev
```

启动日志应包含：

```text
Environments: .env.local
Local: http://localhost:3000
```

访问：

```text
http://localhost:3000/
```

## 6. 快速验证命令

### 6.1 验证 auth ready

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:18080/readyz |
  Select-Object -ExpandProperty Content
```

### 6.2 验证 SkillHub 登录入口第一跳

```powershell
@'
import urllib.request

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

opener = urllib.request.build_opener(NoRedirect)
url = "http://127.0.0.1:8848/v3/auth/aisphere/login?redirect=http%3A%2F%2Flocalhost%3A3000%2F"
try:
    opener.open(url, timeout=5)
except urllib.error.HTTPError as e:
    print(e.code)
    print(e.headers.get("Location"))
'@ | python -
```

期望 Location：

```text
http://localhost:18080/auth/login?app=skillhub&redirect=http%3A%2F%2Flocalhost%3A3000%2F
```

### 6.3 验证 authz/check

```powershell
$body = @{
  subject = "aisphere/test1"
  object = "skillhub:admin:*"
  action = "read"
  app = "skillhub"
} | ConvertTo-Json -Compress

Invoke-WebRequest `
  -UseBasicParsing `
  -Uri http://localhost:18080/authz/check `
  -Method POST `
  -Headers @{"X-Aisphere-Service-Token"="dev-aisphere-service-token-please-change-32chars"} `
  -ContentType "application/json" `
  -Body $body |
  Select-Object -ExpandProperty Content
```

期望：

```json
{
  "allow": true,
  "source": "casdoor",
  "subject": "aisphere/test1",
  "object": "skillhub:admin:*",
  "action": "read"
}
```

### 6.4 浏览器验证

1. 打开：

   ```text
   http://localhost:3000/
   ```

2. 点击：

   ```text
   Sign in with AI Sphere
   ```

3. 使用 Casdoor 用户：

   ```text
   username: <test-user>
   password: <test-password>
   ```

4. 登录后应进入 SkillHub 控制台。

5. Network 中应看到：

   ```text
   /v3/auth/me                         200
   /v3/admin/notifications?pageNo=...  200
   /v3/aihub/skills?pageNo=...      200
   ```

6. Cookie 中应有：

   ```text
   name:   aisphere_session
   domain: localhost
   ```

## 7. 状态码定位

| 现象 | 含义 | 优先检查 |
| --- | --- | --- |
| callback 返回 `auth_callback_failed` | OAuth callback 校验失败 | state 是否由 `aisphere-auth /auth/login` 生成；code 是否重复使用；Redis state 是否过期 |
| SkillHub API 返回 401 | SkillHub 没识别出登录态 | cookie host 是否一致；前端是否走 `localhost`；`aisphereAuth.endpoint` 是否是 `localhost`；`NEXT_PUBLIC_AISPHERE_AUTH_ENABLED` 是否启用 |
| SkillHub API 返回 403 | 已登录但权限拒绝 | `casdoor.permissionId` 是否匹配业务；用户是否在对应角色；Casdoor enforce 是否带 Bearer token |
| 登录成功后落到 18080 `/` | redirect 被丢弃 | `aisphere-auth` 是否允许 localhost absolute redirect |
| 页面仍显示登录页但接口 200 | 前端状态没识别 cookie 登录 | `.env.local` 是否有 `NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true`，Next dev server 是否重启 |

## 8. 工程化建议

### 8.1 permissionId 不应长期全局唯一

当前 `aisphere-auth` 使用：

```yaml
casdoor:
  permissionId: "aisphere/perm_skillhub_admin"
```

这适合当前 SkillHub 本地联调，但如果同一个 `aisphere-auth` 实例服务多个业务，例如：

- SkillHub
- Portal
- AgentRuntime
- SQLHub
- ModelGateway

就不应该只有一个全局 `permissionId`。

建议演进为：

```yaml
casdoor:
  permissionId: "aisphere/perm_platform_admin"
  appPermissions:
    skillhub: "aisphere/perm_skillhub_admin"
    portal: "aisphere/perm_portal_admin"
    agentruntime: "aisphere/perm_agentruntime_admin"
```

`/authz/check` 根据请求里的 `app` 选择 permissionId。

### 8.2 本地开发统一 host

推荐：

```text
front:          http://localhost:3000
skillhub api:   http://localhost:8848
aisphere-auth:  http://localhost:18080
casdoor:        http://CHANGE_ME_HOST:8008
```

不要混用：

```text
127.0.0.1
localhost
```

这是本地 OAuth + HttpOnly cookie 最容易踩坑的点。

### 8.3 初始化脚本必须生成可运行基线

`aisphere-auth/scripts/casdoor/bootstrap-casdoor-mysql.py` 和 `render-casdoor-seed.py` 应持续保证：

- Application 可登录。
- User 页面可新增用户。
- Model 使用 `permissionId` 第六字段。
- permission.model 使用 `owner/name`。
- 登录 permission 覆盖 `aisphere/* -> aisphere-auth -> Read`。
- 业务 permission 覆盖对应业务资源，例如 `skillhub:*`。

### 8.4 前端应区分 token 登录和 cookie 登录

SkillHub 同时支持：

- local JWT token：存在 localStorage
- AI Sphere 平台登录：存在 HttpOnly `aisphere_session` cookie

因此前端不能只根据 `getToken()` 判断登录态。

当 `NEXT_PUBLIC_AISPHERE_AUTH_ENABLED=true` 时，应允许：

```text
无 localStorage token -> 调 /v3/auth/me -> 由后端通过 cookie introspect
```

## 9. 当前已修改的关键文件

`aisphere-auth`：

```text
internal/authn/service_impl.go
internal/authn/service_impl_test.go
internal/casdoor/http_client.go
internal/casdoor/http_client_test.go
scripts/casdoor/render-casdoor-seed.py
scripts/casdoor/bootstrap-casdoor-mysql.py
tests/casdoor/test_casdoor_seed.py
config.yaml
```

`aisphere-hub`：

```text
backend/config.yaml
front/.env.local
```

线上 Casdoor DB 已调整：

- Application redirect URIs 新增 `http://localhost:18080/auth/callback/casdoor`
- model policy_definition 第六字段为 `permissionId`
- permission.model 修正为 `aisphere/aisphere-auth-model`
- 登录 permission 和 SkillHub permission 均已可用

相关备份：

```text
C:\Users\Administrator\AppData\Local\Temp\aisphere-casdoor-auth-backup-20260617234629.json
C:\Users\Administrator\AppData\Local\Temp\aisphere-casdoor-app-backup-20260618000606.json
```

## 10. 一句话排障口诀

```text
callback 失败先查 state；
401 先查 cookie host；
403 先查 permissionId 和角色；
Casdoor Unauthorized operation 先查 enforce Bearer token；
前端仍显示登录页先查 NEXT_PUBLIC_AISPHERE_AUTH_ENABLED。
```
