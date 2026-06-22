# Access Admin API

## GET /v3/admin/access/overview

返回 Casbin 策略概览、当前 principal 和当前角色。

## GET /v3/admin/access/policies

查询 p 策略。

Query:

- `subject`
- `object`
- `action`

Response item:

```json
{
  "id": "sha256",
  "ptype": "p",
  "subject": "role:developer",
  "object": "aihub:skill:*",
  "action": "skill:admin:read",
  "effect": "allow",
  "raw": ["role:developer", "aihub:skill:*", "skill:admin:read", "allow"]
}
```

## POST /v3/admin/access/policies

新增 p 或 g 策略。

新增 p：

```json
{
  "ptype": "p",
  "subject": "role:developer",
  "object": "aihub:skill:*",
  "action": "skill:admin:read",
  "effect": "allow"
}
```

新增 g：

```json
{
  "ptype": "g",
  "subject": "human:xxx",
  "role": "role:admin"
}
```

## POST /v3/admin/access/policies/remove

删除 p 或 g 策略。参数同新增。

## GET /v3/admin/access/role-bindings

查询 g 角色绑定。

## POST /v3/admin/access/role-bindings

```json
{
  "subject": "human:xxx",
  "role": "role:admin"
}
```

## DELETE /v3/admin/access/role-bindings

Query:

- `subject`
- `role`

## GET /v3/admin/access/role-mappings

查询 Casdoor role -> SkillHub role 映射。包含 config 映射和 DB 映射。

## POST /v3/admin/access/role-mappings

```json
{
  "provider": "casdoor",
  "externalRole": "aihub-admin",
  "internalRole": "role:admin",
  "description": "SkillHub admin from Casdoor"
}
```

## DELETE /v3/admin/access/role-mappings/{id}

删除 DB 角色映射。配置文件来源的映射不能通过 API 删除。

## POST /v3/admin/access/evaluate

```json
{
  "subject": "human:xxx",
  "object": "aihub:skill:*",
  "action": "skill:admin:read"
}
```

## POST /v3/admin/access/reload

重新加载 Casbin 策略。
