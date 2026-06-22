# SkillHub + Casdoor Remote Authorization 工程化说明

## 1. 最终定位

本版本把 Casdoor 作为平台内部 IAM 与权限中心：

- Casdoor 管用户、组织、应用、角色、角色绑定、Casbin Model、Permission、Adapter、Enforcer。
- SkillHub 管业务资源：Skill、Group、Version、Proposal、Runtime。
- SkillHub 不再本地维护 Casbin policy / role binding / role mapping。
- SkillHub 每次访问受保护接口时，把 `subject / object / action` 发送给 Casdoor `/api/enforce` 做最终授权判断。

Casdoor 的 Exposed Casbin APIs 要从后端调用，并使用应用的 Client ID / Client Secret 做 HTTP Basic Auth。

## 2. 后端配置

推荐配置：

```yaml
authz:
  provider: "casdoor-remote"
  casdoor:
    endpoint: "${CASDOOR_ENDPOINT}"
    owner: "built-in"              # Casdoor organization
    application: "skillhub"
    permission: "skillhub_permission"
    permissionId: ""               # 可选，优先级高于 owner/permission
    model: ""                      # 可选
    modelId: ""                    # 可选
    clientId: "${SKILLHUB_CASDOOR_CLIENT_ID}"
    clientSecret: "${SKILLHUB_CASDOOR_CLIENT_SECRET}"
    subjectFormat: "casdoor"       # casdoor -> org/username
    cacheTTLSeconds: 30
    failClosed: true
```

`subjectFormat` 可选：

- `casdoor`：优先使用 `organization/username`，推荐用于 Casdoor Role / Permission 绑定。
- `principal`：使用 SkillHub 内部 subject，例如 `human:uuid`。
- `external`：使用 JWT 原始 `sub`。

## 3. Casdoor 里怎么配置

### 3.1 创建 Model

建议模型：

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub) || r.sub == p.sub) &&
    (p.obj == "*" || keyMatch(r.obj, p.obj)) &&
    (p.act == "*" || r.act == p.act)
```

### 3.2 创建 Permission

建议：

- Owner: `built-in` 或你的组织名
- Name: `skillhub_permission`
- Model: 上面创建的 model
- Adapter: Casdoor 默认数据库 Adapter

### 3.3 创建角色和策略

建议角色：

- `aihub-admin`
- `aihub-developer`
- `aihub-reviewer`
- `aihub-agent`
- `aihub-viewer`

如果你希望少一层映射，可以直接在 Permission 里写：

```text
p, aihub-admin, *, *, allow
p, aihub-developer, aihub:skill:*, skill:admin:read, allow
p, aihub-developer, aihub:skill:*, skill:read, allow
p, aihub-developer, aihub:skill:*, skill:create, allow
p, aihub-developer, aihub:skill:*, skill:update, allow
p, aihub-developer, aihub:skillset:*, skill:group:read, allow
p, aihub-developer, aihub:skillset:*, skill:group:write, allow
p, aihub-reviewer, aihub:proposal:*, skill:proposal:review, allow
p, aihub-agent, aihub:skill:*, skill:read, allow
p, aihub-agent, aihub:skillset:*, skill:group:read, allow
p, aihub-agent, aihub:proposal:*, skill:proposal:create, allow
p, aihub-viewer, aihub:skill:*, skill:admin:read, allow
p, aihub-viewer, aihub:skill:*, skill:read, allow
p, aihub-viewer, aihub:skillset:*, skill:group:read, allow
```

然后在 Casdoor Role 页面把用户绑定到对应角色。

## 4. SkillHub 资源动作表

| Object | Action | 说明 |
|---|---|---|
| `aihub:skill:*` | `skill:admin:read` | 后台读取 Skill |
| `aihub:skill:*` | `skill:admin:write` | 后台写 Skill |
| `aihub:skill:*` | `skill:publish` | 发布 Skill |
| `aihub:skill:*` | `skill:read` | Runtime 读 Skill |
| `aihub:skillset:*` | `skill:group:read` | 读取 Group |
| `aihub:skillset:*` | `skill:group:write` | 维护 Group |
| `aihub:proposal:*` | `skill:proposal:review` | 审核 Proposal |
| `aihub:proposal:*` | `skill:proposal:create` | Agent 提交 Proposal |
| `access:*` | `access:admin:read` | 查看权限诊断页 |
| `notification:*` | `notification:read` | 查看通知 |
| `system:*` | `system:admin` | 系统管理 |

## 5. SkillHub Access 页面变化

现在 Access 页面不再管理本地 policy，只保留：

- 当前 Casdoor authz provider 信息
- 当前登录用户解析后的 Casdoor subject
- SkillHub 支持的 object/action 模板
- 权限测试：调用 SkillHub 后端，再由后端调用 Casdoor `/api/enforce`
- 快捷跳转 Casdoor 用户、角色、权限、模型页面

## 6. 故障排查

### 401 unauthorized

认证失败，通常是 OIDC token 问题：issuer、audience、JWKS、token 过期。

### 403 forbidden

认证成功，Casdoor 权限拒绝。进入 Casdoor Permission 或 SkillHub Access -> Permission Test 验证：

```text
subject = built-in/test01
object  = aihub:skill:*
action  = skill:admin:read
```

### Casdoor enforce error

检查：

- `authz.casdoor.endpoint`
- `authz.casdoor.owner`
- `authz.casdoor.permission`
- `authz.casdoor.clientId/clientSecret`
- Casdoor Application 的 Client ID / Secret 是否一致
- Casdoor Permission 名称是否存在
