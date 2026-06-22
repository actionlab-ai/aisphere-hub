# AIHub Backend

Gin-based backend for the independent AIHub.

## Current architecture

- Skill registry: namespace-free, group-first.
- Group: business/capability package.
- Access Space: IAM/RBAC boundary only.
- Storage: MySQL + Redis/Cluster + MinIO/S3, with local fallbacks.
- Auth: local / external / mixed.

## Canonical APIs

```http
GET    /v3/aihub/skills
POST   /v3/aihub/skills/upload
GET    /v3/aihub/skill/{skillName}
DELETE /v3/aihub/skill/{skillName}
GET    /v3/aihub/skill/{skillName}/versions/{version}
GET    /v3/aihub/skill/{skillName}/versions/{version}/download
GET    /v3/aihub/groups
POST   /v3/aihub/groups
GET    /v3/aihub/group/{groupName}
GET    /v3/client/ai/skills/{skillName}
GET    /v3/client/ai/groups/{groupName}
GET    /registry-global/.well-known/agent-skills/index.json
```

## Legacy API

Nacos-style APIs under `/v3/admin/ai/skills` and `/registry/{namespaceId}` remain as compatibility shims. Incoming `namespaceId` is ignored for skill lookup.

## Casdoor + Casbin mode

Use Casdoor as the IAM service and Casbin as the embedded authorization engine:

```powershell
cd ..\deployments\casdoor-casbin
docker compose up -d

cd ..\..\backend
$env:SKILLHUB_CASDOOR_CLIENT_SECRET="<casdoor-client-secret>"
go run .\cmd\skillhub\main.go --config .\configs\casdoor-casbin.yaml
```

Casbin policies are loaded from `configs/casbin/policy.csv` in file mode, or persisted in MySQL table `aihub_casbin_rule` in mysql mode.
