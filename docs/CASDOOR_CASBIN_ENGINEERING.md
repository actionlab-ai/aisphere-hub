# SkillHub + Casdoor + Casbin 工程化组合方案

## 1. 分工

SkillHub 不再自研完整账号体系。

```text
Casdoor = 独立服务化部署的 IAM / SSO / 本地账号中心
Casbin  = Go 库，import 到 SkillHub 后端，负责 SkillHub 资源授权
SkillHub = Skill / Group / Version / Proposal / Runtime / Governance
```

### Casdoor 做什么

- 用户、组织、应用、登录页。
- 本地账号密码体系。
- 外部身份源接入，例如 OIDC / OAuth2 / SAML / LDAP。
- 给 SkillHub 颁发 OIDC / JWT access token。
- 给 Agent / Service / CI 创建身份或应用。

### Casbin 做什么

Casbin 不是单独服务，它是 SkillHub 后端里的 Go 依赖：

```go
import "github.com/casbin/casbin/v2"
```

SkillHub 在每个写接口、发布接口、下载接口、Proposal 审核接口前执行：

```go
enforcer.Enforce(subject, object, action)
```

例如：

```text
subject = user:alice
object  = skill:dialogue-card
action  = skill:publish
```

或者：

```text
subject = agent:dialogue_worker
object  = group:novel-writing-suite
action  = skill:read
```

## 2. 本次代码改动

### 后端新增

```text
backend/internal/authz/casbin/authorizer.go
backend/internal/authz/casbin/adapter.go
backend/configs/casbin/model.conf
backend/configs/casbin/policy.csv
backend/configs/casdoor-casbin.yaml
backend/migrations/007_casbin_policy.sql
```

### 前端新增/调整

- Login 页面新增 `Sign in with Casdoor / OIDC` 按钮。
- 点击后跳转 `/v3/auth/oidc/login`。
- Casdoor 登录完成后回调 `/v3/auth/oidc/callback`。
- 后端用 code 换 token，再写入前端 localStorage，进入 Console。

### 部署新增

```text
deployments/casdoor-casbin/docker-compose.yml
deployments/casdoor-casbin/casdoor/app.conf
deployments/casdoor-casbin/mysql/init.sql
```

该 compose 启动：

```text
MySQL + Redis + MinIO + Casdoor
```

注意：Casbin 不在 compose 里，因为 Casbin 是 SkillHub 后端内嵌库。

## 3. 运行方式

### 3.1 启动依赖

```powershell
cd deployments\casdoor-casbin
docker compose up -d
```

访问 Casdoor：

```text
http://127.0.0.1:8000
```

### 3.2 在 Casdoor 里创建 SkillHub 应用

建议：

```text
Organization: skillhub
Application: skillhub
Client ID: skillhub
Redirect URI: http://127.0.0.1:8848/v3/auth/oidc/callback
```

然后复制 Application 的 Client Secret：

```powershell
$env:SKILLHUB_CASDOOR_CLIENT_SECRET="<client-secret>"
```

### 3.3 启动后端

```powershell
cd backend
go mod tidy
go run .\cmd\skillhub\main.go --config .\configs\casdoor-casbin.yaml
```

### 3.4 启动前端开发模式

```powershell
cd front
$env:SKILLHUB_BACKEND_URL="http://127.0.0.1:8848"
npm install
npm run dev
```

访问：

```text
http://127.0.0.1:3000
```

### 3.5 后端托管静态前端

```powershell
cd front
npm run build:static

cd ..\backend
go run .\cmd\skillhub\main.go --config .\configs\casdoor-casbin.yaml
```

访问：

```text
http://127.0.0.1:8848/ui/
```

## 4. Casbin 权限模型

模型文件：

```text
backend/configs/casbin/model.conf
```

策略文件：

```text
backend/configs/casbin/policy.csv
```

MySQL 策略表：

```text
skillhub_casbin_rule
```

`authz.policyStore=mysql` 时，SkillHub 启动后会使用 `skillhub_casbin_rule`。如果表为空，会从 `policy.csv` 自动种子初始化。

## 5. 资源命名规范

```text
skill:{skillName}
group:{groupName}
proposal:{proposalId}
overlay:{overlayRef}
audit:*
metrics:*
iam:*
system:*
```

权限动作：

```text
skill:read
skill:create
skill:update
skill:delete
skill:publish
skill:online
skill:offline
skill:label:update

group:read
group:create
group:update
group:delete
group:bind

proposal:create
proposal:read
proposal:validate
proposal:approve
proposal:reject

audit:read
metrics:read
iam:admin
system:admin
```

## 6. 推荐策略

```text
p, role:admin, *, *, allow
p, role:developer, aihub:skill:*, skill:read, allow
p, role:developer, aihub:skill:*, skill:update, allow
p, role:reviewer, aihub:proposal:*, proposal:approve, allow
p, role:agent, aihub:skill:*, skill:read, allow
p, role:agent, aihub:proposal:*, proposal:create, allow

g, user:admin, role:admin
g, agent:dialogue_worker, role:agent
```

## 7. 为什么 Casdoor 服务化，Casbin import

Casdoor 是账号中心，需要管理 UI、数据库、登录会话、外部身份源、OAuth/OIDC 协议，所以应该独立服务化部署。

Casbin 是授权决策库，判断的是 SkillHub 内部业务资源。它必须贴近 SkillHub 的接口和资源模型，所以直接 import 到后端代码里最合适。

```text
Casdoor 服务化部署
Casbin 作为 Go library import
```

## 8. 后续建议

下一步可以继续补：

```text
1. /v3/authz/policies 策略管理 API
2. 前端 Access Policy 页面接 Casbin policy API
3. Casdoor 角色同步任务
4. Agent Client Credentials 模式示例
5. HttpOnly Cookie 版本的 OIDC callback
```
