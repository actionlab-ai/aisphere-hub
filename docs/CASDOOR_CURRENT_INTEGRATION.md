# SkillHub 接入现有 Casdoor（${CASDOOR_ENDPOINT}）

本版本已经把你当前上传的 Next Console 前端接入到 SkillHub 后端，并把登录主路径改成 Casdoor OIDC。

## 1. 最终分工

```text
Casdoor：用户、组织、登录、SSO、应用、Client Secret、本地账号/外部账号
Casbin：SkillHub 后端内嵌 Go 库，负责 Skill / Group / Proposal 的资源级权限判断
SkillHub：Skill Registry、Group 能力包、Version、Label、Proposal、Runtime 下载
```

Casdoor 是一个独立服务，不 import 到 SkillHub 代码里；Casbin 是 Go 依赖，已经在 `backend/go.mod` 中引入：

```go
github.com/casbin/casbin/v2
```

## 2. 你现在的 Casdoor 地址

当前按你给的地址预置：

```text
${CASDOOR_ENDPOINT}
```

后端配置文件：

```text
backend/configs/casdoor-casbin.yaml
```

关键配置：

```yaml
auth:
  enabled: true
  mode: "external"
  providers:
    - name: "casdoor"
      type: "oidc"
      issuer: "${CASDOOR_ENDPOINT}"
      audience: "skillhub"
      clientId: "skillhub"
      clientSecret: "${SKILLHUB_CASDOOR_CLIENT_SECRET}"
      redirectUrl: "http://127.0.0.1:8848/v3/auth/oidc/callback"
```

注意：`redirectUrl` 必须和 Casdoor Application 里的 Redirect URLs 完全一致。

## 3. Casdoor 里要怎么配置

进入 Casdoor 控制台后，建议最小配置如下：

### 3.1 创建 Organization

可以用：

```text
skillhub
```

如果暂时不想新建，也可以先用 `built-in`，但正式建议单独建 `skillhub`。

### 3.2 创建 Application

建议：

```text
Application name: skillhub
Client ID: skillhub
Client Secret: 复制出来给 SkillHub 后端环境变量
```

### 3.3 配置 Redirect URLs

本地开发，后端在本机 8848：

```text
http://127.0.0.1:8848/v3/auth/oidc/callback
```

如果后端部署在服务器，例如：

```text
http://skillhub.example.com/v3/auth/oidc/callback
```

则需要把 `backend/configs/casdoor-casbin.yaml` 里的 `redirectUrl` 同步改成这个地址，并且 Casdoor Application 里也加这个地址。

### 3.4 给用户设置角色

SkillHub 当前默认识别这些角色：

```text
admin

developer
reviewer
agent
```

Casdoor token 里如果能带出 `roles` 或 `groups`，SkillHub 会映射到内部角色。当前内置 Casbin 策略：

```text
role:admin      允许所有操作
role:developer  允许 Skill / Group 管理
role:reviewer   允许 Proposal 审核
role:agent      允许 Runtime 读取、Proposal 创建、Overlay 读取
```

如果你的 Casdoor token 暂时没有 `roles`，可以先在 `backend/configs/casbin/policy.csv` 或 MySQL 的 `skillhub_casbin_rule` 里加 `g` 规则，例如：

```text
g, user:<casdoor-sub-or-name>, role:admin
```

## 4. 登录流程

开发期推荐：

```powershell
# 1. 后端
cd backend
$env:SKILLHUB_CASDOOR_CLIENT_SECRET="Casdoor 里的 client secret"
go run .\cmd\skillhub\main.go --config .\configs\casdoor-casbin.yaml

# 2. 前端
cd ..\front
$env:SKILLHUB_BACKEND_URL="http://127.0.0.1:8848"
npm install
npm run dev
```

浏览器访问：

```text
http://127.0.0.1:3000
```

点击：

```text
Sign in with Casdoor
```

流程：

```text
前端 /v3/auth/oidc/login
  -> Next rewrites 到 SkillHub 后端
  -> SkillHub 跳转 Casdoor
  -> Casdoor 登录
  -> 回调 SkillHub /v3/auth/oidc/callback
  -> SkillHub 换取 access_token
  -> 跳回前端 /auth/callback#access_token=...
  -> 前端写入 localStorage
  -> 前端调用 /v3/auth/me
```

## 5. 为什么做了 token fragment relay

开发期前端是：

```text
http://127.0.0.1:3000
```

后端回调是：

```text
http://127.0.0.1:8848/v3/auth/oidc/callback
```

如果后端 callback 直接在 8848 写 localStorage，3000 端口的前端读不到。因此这版后端支持把 token 通过 URL fragment relay 回：

```text
http://127.0.0.1:3000/auth/callback#access_token=xxx
```

前端 `src/app/auth/callback/page.tsx` 会读取 fragment 并保存 token。

生产环境建议把前端和后端放到同一个域名下，例如：

```text
https://skillhub.example.com/ui
https://skillhub.example.com/v3
```

这样可以减少跨端口和跨域问题。

## 6. Casbin 权限文件

模型：

```text
backend/configs/casbin/model.conf
```

初始策略：

```text
backend/configs/casbin/policy.csv
```

MySQL 模式下会持久化到：

```text
skillhub_casbin_rule
```

## 7. 常见问题

### 7.1 OIDC discovery 失败

执行：

```powershell
curl ${CASDOOR_ENDPOINT}/.well-known/openid-configuration
```

如果失败，检查 Casdoor 的 `origin`、端口、防火墙、Nginx 反代。

### 7.2 redirect_uri mismatch

Casdoor Application 里的 Redirect URLs 必须包含：

```text
http://127.0.0.1:8848/v3/auth/oidc/callback
```

如果后端部署在公网，改成公网 SkillHub 后端地址。

### 7.3 登录成功但接口 403

说明认证成功了，但 Casbin 没授权。先给用户加 admin：

```text
g, user:<subject>, role:admin
```

或者让 Casdoor token 带 `roles: ["admin"]`。

### 7.4 前端 API 访问到 3000 返回 HTML

检查：

```text
front/next.config.ts
SKILLHUB_BACKEND_URL=http://127.0.0.1:8848
```

开发期 Next rewrites 会把 `/v3/**` 转发到后端。
