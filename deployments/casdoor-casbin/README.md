# SkillHub + Casdoor + Casbin local stack

This stack deploys external dependencies only:

- MySQL: SkillHub metadata and Casbin policy DB table
- Redis: runtime cache / distributed lock / rate limit
- MinIO: Skill package object store
- Casdoor: IAM / login / organization / application / local accounts

Casbin is **not** a separate service. It is imported as a Go library and embedded in SkillHub backend.

## Start dependencies

```powershell
cd deployments\casdoor-casbin
docker compose up -d
```

Open Casdoor:

```text
http://127.0.0.1:8000
```

Then create a Casdoor organization and application:

```text
Organization: skillhub
Application: skillhub
Redirect URI: http://127.0.0.1:8848/v3/auth/oidc/callback
```

Copy the Casdoor application client secret into:

```powershell
$env:SKILLHUB_CASDOOR_CLIENT_SECRET="<client-secret>"
```

Start SkillHub:

```powershell
cd backend
go run .\cmd\skillhub\main.go --config .\configs\casdoor-casbin.yaml
```
