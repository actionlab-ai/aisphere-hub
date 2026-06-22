# SkillHub API Spec v2 — Group-First / Namespace-Free

## Canonical Skill APIs

### List skills

```http
GET /v3/aihub/skills?search=&groupName=&pageNo=1&pageSize=20
```

### Upload skill ZIP

```http
POST /v3/aihub/skills/upload
Content-Type: multipart/form-data

file=<skill.zip>
overwrite=true
targetVersion=0.0.1
commitMsg=init
```

### Skill detail

```http
GET /v3/aihub/skill/{skillName}
DELETE /v3/aihub/skill/{skillName}
```

### Versions

```http
GET  /v3/aihub/skill/{skillName}/versions/{version}
GET  /v3/aihub/skill/{skillName}/versions/{version}/download
GET  /v3/aihub/skill/{skillName}/versions/{version}/files
GET  /v3/aihub/skill/{skillName}/versions/{version}/file?path=SKILL.md
GET  /v3/aihub/skill/{skillName}/compare?baseVersion=0.0.1&targetVersion=0.0.2
```

### Lifecycle

```http
POST /v3/aihub/skill/{skillName}/submit
POST /v3/aihub/skill/{skillName}/publish
POST /v3/aihub/skill/{skillName}/online
POST /v3/aihub/skill/{skillName}/offline
PUT  /v3/aihub/skill/{skillName}/labels
```

## Canonical Group APIs

```http
GET    /v3/aihub/groups
POST   /v3/aihub/groups
GET    /v3/aihub/group/{groupName}
PUT    /v3/aihub/group/{groupName}
DELETE /v3/aihub/group/{groupName}
GET    /v3/aihub/group/{groupName}/skills
POST   /v3/aihub/group/{groupName}/skills
DELETE /v3/aihub/group/{groupName}/skills/{skillName}
```

## Runtime APIs

```http
GET /v3/client/ai/skills/{skillName}?label=stable
GET /v3/client/ai/skills/{skillName}?version=0.0.1&md5=...
GET /v3/client/ai/groups/{groupName}?label=stable
```

## Registry discovery

```http
GET /registry-global/api/search?q=dialogue
GET /registry-global/.well-known/agent-skills/index.json
GET /registry-global/.well-known/agent-skills/{skillName}.zip
GET /registry-global/.well-known/agent-skills/{skillName}/SKILL.md
```

## Access Space / IAM APIs

These replace the old conceptual use of namespace. They are retained for RBAC/ABAC only.

```http
GET    /v3/admin/namespaces
POST   /v3/admin/namespaces
GET    /v3/admin/namespaces/{accessSpaceId}
PUT    /v3/admin/namespaces/{accessSpaceId}
GET    /v3/admin/namespaces/{accessSpaceId}/members
POST   /v3/admin/namespaces/{accessSpaceId}/members
DELETE /v3/admin/namespaces/{accessSpaceId}/members/{subjectId}
```

> Path name remains `/namespaces` for backward compatibility, but the UI and docs call it Access Space.

## Legacy Nacos-compatible APIs

Still available but deprecated. `namespaceId` is ignored for skill registry lookup.

```http
GET /v3/admin/ai/skills/list?namespaceId=public
GET /v3/client/ai/skills?namespaceId=public&name={skillName}
GET /registry/{namespaceId}/.well-known/agent-skills/index.json
```
