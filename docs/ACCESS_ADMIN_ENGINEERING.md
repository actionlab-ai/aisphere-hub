# SkillHub Access Admin 工程化说明

本版把 Casbin 权限管理从手写 `policy.csv` / 手工插 SQL，升级成平台化后台能力。

## 分工边界

- Casdoor：统一账号、组织、登录、基础角色，例如 `aihub-admin`、`aihub-developer`。
- SkillHub：Skill / Group / Proposal 等业务资源。
- Casbin：嵌入 SkillHub 后端，执行资源级权限判断。
- MySQL：保存 Casbin policy 和 Casdoor role 到 SkillHub role 的映射。

## 默认配置建议

你的当前配置可以继续使用：

```yaml
auth:
  enabled: true
  mode: "external"
  providers:
    - name: "casdoor"
      type: "oidc"
      issuer: "${CASDOOR_ENDPOINT}"
      audience: "<Casdoor Application Client ID>"
      clientId: "<Casdoor Application Client ID>"
      clientSecret: "${SKILLHUB_CASDOOR_CLIENT_SECRET}"
      redirectUrl: "http://127.0.0.1:8848/v3/auth/oidc/callback"

authz:
  provider: "casbin"
  model: "./configs/casbin/model.conf"
  policyStore: "file"   # 本地调试可用 file；生产推荐 mysql
  policyFile: "./configs/casbin/policy.csv"
  autoSave: true
```

生产推荐：

```yaml
authz:
  provider: "casbin"
  model: "./configs/casbin/model.conf"
  policyStore: "mysql"
  policyFile: "./configs/casbin/policy.csv"
  autoSave: true
```

当 `policyStore=mysql` 时，第一次启动如果 `skillhub_casbin_rule` 为空，会从 `policy.csv` 种子导入；后续平台页面增删策略会落 MySQL。

## 新增数据库表

新增迁移：

```text
backend/migrations/008_access_admin.sql
```

它创建：

```text
skillhub_role_mapping
```

用于维护 Casdoor 外部角色到 SkillHub 内部角色的映射。

默认种子包括：

```text
platform-admin      -> role:admin
aihub-admin      -> role:admin
admin               -> role:admin
aihub-developer  -> role:developer
developer           -> role:developer
aihub-reviewer   -> role:reviewer
reviewer            -> role:reviewer
aihub-agent      -> role:agent
agent               -> role:agent
aihub-viewer     -> role:viewer
viewer              -> role:viewer
```

## 新增后端接口

所有接口都在：

```text
/v3/admin/access/**
```

菜单和接口需要权限：

```text
access:admin:read  on access:*
access:admin:write on access:*
```

默认 `role:admin` 拥有所有权限。

### Overview

```http
GET /v3/admin/access/overview
```

返回 policyStore、策略数量、当前 principal、当前角色、菜单权限模型。

### Permission Policies

```http
GET  /v3/admin/access/policies
POST /v3/admin/access/policies
POST /v3/admin/access/policies/remove
```

对应 Casbin `p` 规则：

```text
p, subject, object, action, allow
```

示例：

```json
{
  "ptype": "p",
  "subject": "role:developer",
  "object": "aihub:skill:*",
  "action": "skill:admin:read",
  "effect": "allow"
}
```

### Role Bindings

```http
GET    /v3/admin/access/role-bindings
POST   /v3/admin/access/role-bindings
DELETE /v3/admin/access/role-bindings?subject=human:xxx&role=role:admin
```

对应 Casbin `g` 规则：

```text
g, subject, role
```

示例：

```json
{
  "subject": "human:fcd6bd11-dbd5-49da-b6df-8abbff35789b",
  "role": "role:admin"
}
```

### Role Mappings

```http
GET    /v3/admin/access/role-mappings
POST   /v3/admin/access/role-mappings
DELETE /v3/admin/access/role-mappings/{id}
```

它管理 Casdoor role 到 SkillHub role 的映射。建议常用方式是在 Casdoor 给用户分配 `aihub-admin`，SkillHub 自动映射成 `role:admin`。

### Permission Test

```http
POST /v3/admin/access/evaluate
```

请求：

```json
{
  "subject": "human:fcd6bd11-dbd5-49da-b6df-8abbff35789b",
  "object": "aihub:skill:*",
  "action": "skill:admin:read"
}
```

返回是否允许和当前 subject 的 Casbin roles。

### Reload

```http
POST /v3/admin/access/reload
```

从 policyStore 重新加载 Casbin 策略。

## 前端变化

新增菜单：

```text
Access -> Access Admin
```

只有当前 principal 具备以下任一角色/权限时显示：

```text
admin
role:admin
aihub-admin
platform-admin
access:admin:read
*
```

页面包含：

- Overview
- Policies
- Role Bindings
- Role Mappings
- Permission Test

## 推荐使用方式

### 初期调试

1. 在 Casdoor 给用户设置角色 `admin` 或 `aihub-admin`。
2. SkillHub 通过 `auth.roleMappings` 或 `skillhub_role_mapping` 映射为 `role:admin`。
3. 进入 Access Admin 页面管理策略。

### 生产

1. Casdoor 管用户、组织、登录、基础角色。
2. SkillHub Access Admin 管本系统的资源权限。
3. `authz.policyStore` 使用 `mysql`。
4. 不再手写 SQL 和 `policy.csv`，只把 `policy.csv` 当初始种子。
