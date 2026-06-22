# SkillHub Access Diagnostic API

当 `authz.provider=casdoor-remote` 时，SkillHub 只提供权限诊断接口，不提供 policy CRUD。

## GET /v3/admin/access/overview

返回当前远程授权配置、当前 principal、解析后的 Casdoor subject、资源动作模板和 Casdoor 快捷链接。

## GET /v3/admin/access/resources

返回 SkillHub 支持的 object/action 模板。

## GET /v3/admin/access/links

返回 Casdoor 用户、角色、权限、模型等后台入口。

## POST /v3/admin/access/evaluate

请求：

```json
{
  "subject": "built-in/test01",
  "object": "aihub:skill:*",
  "action": "skill:admin:read"
}
```

返回：

```json
{
  "data": {
    "allowed": true,
    "subject": "built-in/test01",
    "object": "aihub:skill:*",
    "action": "skill:admin:read",
    "provider": "casdoor-remote"
  }
}
```
