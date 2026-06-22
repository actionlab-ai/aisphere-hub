# AI Sphere Hub / SkillHub

本仓库是 AI Sphere 平台中的 SkillHub 子系统，提供 Agent Skill 的注册、治理、版本、发布、权限和审计能力。

- 作为 AI Sphere 平台子应用，可启用 aisphere-auth 集成（authn + authz + audit）。详见 [`docs/AISPHERE_AUTH_INTEGRATION.md`](docs/AISPHERE_AUTH_INTEGRATION.md)。
- Casdoor 负责账号、组织、登录、本地用户/外部用户。
- Casbin 作为 Go 库内嵌在 SkillHub 后端，负责 Skill / Group / Proposal 资源级权限。
- Next 前端登录页新增 `Sign in with Casdoor`。
- 开发期通过 Next rewrites 转发 `/v3/**` 到 SkillHub 后端。

重点文档：

- `docs/CASDOOR_CURRENT_INTEGRATION.md`
- `docs/FRONTEND_CASDOOR_UPDATE.md`
- `docs/CASDOOR_CASBIN_ENGINEERING.md`

---

# SkillHub — Group-First Agent Skill Registry

This is an independent SkillHub project with separated frontend and backend:

```text
backend/   Gin API server
front/     Next.js console
docs/      API and architecture documents
```

## Core design

This version is **not namespace-first**. The skill registry is now:

```text
Group -> Skill -> Version -> Label
```

`namespace` is no longer a skill dimension. Existing `/v3/admin/namespaces` APIs are retained as **Access Space / IAM** APIs only.

Read:

- `backend/docs/GROUP_FIRST_NAMESPACE_FREE_DESIGN.md`
- `docs/API_SPEC_GROUP_FIRST_V2.md`
- `docs/NAMESPACE_FIELD_AUDIT.md`
- `docs/openapi-group-first-v2.yaml`

## Start backend

```bash
cd backend
go mod tidy
go run ./cmd/skillhub/main.go --config ./config.example.yaml
```

## Start frontend

```bash
cd front
npm install
npm run dev
```

Or build frontend and serve from backend `/ui`:

```bash
cd front
npm run build
cd ../backend
go run ./cmd/skillhub/main.go --config ./config.example.yaml
```

## Canonical APIs

```http
GET  /v3/aihub/skills
POST /v3/aihub/skills/upload
GET  /v3/aihub/skill/{skillName}
GET  /v3/aihub/groups
GET  /v3/client/ai/skills/{skillName}?label=stable
GET  /v3/client/ai/groups/{groupName}?label=stable
GET  /registry-global/.well-known/agent-skills/index.json
```

Legacy Nacos-compatible APIs still exist, but `namespaceId` is deprecated and ignored by the registry.

## 前端整合说明

本版已整合用户提供的 Next.js Console 前端，并清理 mock API。前端位于 `front/`，后端位于 `backend/`。

- Skill 管理采用 group-first API，不再使用 namespace 作为 Skill 维度。
- Access Space 仅用于 IAM/RBAC，底层 legacy endpoint 仍为 `/v3/admin/namespaces`。
- 开发期可用 Next rewrites 代理 `/v3` 到后端；生产推荐 Gin 托管 `front/out` 或由外部网关统一转发。

详细见：`docs/FRONTEND_INTEGRATION_AND_PROXY.md`。

## AI Sphere Auth integration

推荐生产路径：

```yaml
aisphereAuth:
  enabled: true
  endpoint: "${AISPHERE_AUTH_ENDPOINT}"
  serviceToken: "${AISPHERE_SERVICE_TOKEN}"
  app: "skillhub"

authz:
  provider: "aisphere-auth"

ops:
  audit:
    enabled: true
```

开启后 SkillHub 将：

- 通过 aisphere-auth introspect 统一 session cookie；
- 通过 aisphere-auth `/authz/check` 做资源授权；
- 本地记录 audit，同时镜像写入 aisphere-auth `/audit/events`。

## Casdoor + Casbin IAM/Authz integration

This package includes an engineering integration for external IAM and embedded authorization:

- Casdoor is deployed as an external IAM / SSO / local account service.
- SkillHub validates Casdoor OIDC/JWT tokens.
- Casbin is imported into the Go backend and used for SkillHub resource authorization.

See:

```text
docs/CASDOOR_CASBIN_ENGINEERING.md
docs/ARCHITECTURE_CASDOOR_CASBIN.md
backend/configs/casdoor-casbin.yaml
backend/configs/casbin/model.conf
backend/configs/casbin/policy.csv
deployments/casdoor-casbin/docker-compose.yml
```

## Casdoor Remote Authorization

Current recommended production authorization mode:

```yaml
authz:
  provider: "casdoor-remote"
```

In this mode Casdoor is treated as an internal IAM and permission-center component. SkillHub no longer maintains its own policy editor. Users, roles, role bindings, Casbin model, permission and adapter are managed in Casdoor. SkillHub only calls Casdoor's exposed Casbin `/api/enforce` API for allow/deny decisions, and the Console Access page provides diagnostics, resource/action templates and quick links back to Casdoor.

See:

- `docs/CASDOOR_REMOTE_AUTHZ_ENGINEERING.md`
- `docs/CASDOOR_REMOTE_OPENAPI.md`
- `deployments/casdoor-remote/README.md`
