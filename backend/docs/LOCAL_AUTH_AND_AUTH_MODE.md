# Local Auth, External Auth, and Mixed Auth Modes

AIHub supports three authentication modes:

```yaml
auth:
  mode: local    # only AIHub built-in accounts
  mode: external # only external OIDC/JWT/Introspection/API-key providers
  mode: mixed    # local accounts + external providers, recommended during migration
```

## Why local auth exists

AIHub should normally integrate with an existing identity system, such as Keycloak, Casdoor, an existing Go Admin JWT, or an Agent platform JWT. But in offline deployments, demos, or private environments, there may be no external IdP. In that case AIHub can run with its own local account system.

Local auth is not a separate authorization model. It still maps into the same internal `Principal` model:

```text
human / organization / agent / service
  -> roles
  -> permissions
  -> namespaces
```

So switching from `local` to `external` does not change Skill/Group/Proposal permission checks.

## Local login flow

```text
POST /v3/auth/login
  username + password
  -> AIHub verifies local account password
  -> AIHub signs local HS256 JWT
  -> client uses Authorization: Bearer <accessToken>
```

Example:

```bash
curl -X POST http://127.0.0.1:8848/v3/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"CHANGE_ME_PASSWORD"}'
```

Then:

```bash
curl http://127.0.0.1:8848/v3/admin/iam/whoami \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Local account storage

The current implementation supports config/bootstrap + account file storage:

```yaml
auth:
  local:
    accountFile: ./data/iam/local_accounts.json
    autoCreateBootstrap: true
    users:
      - username: admin
        password: CHANGE_ME_PASSWORD
        subjectId: user:admin
        subjectType: human
        roles: [admin]
        permissions: ['*']
        namespaces: ['*']
```

When `autoCreateBootstrap` is enabled, bootstrap users are created into the account file if they do not already exist. Passwords are saved as PBKDF2-SHA256 hashes, not plaintext.

The migration file `004_local_account_auth.sql` also creates DB tables reserved for a future DB-backed local account store:

```text
iam_local_account
iam_service_account
```

## Local user APIs

```http
POST /v3/auth/login
POST /v3/auth/refresh
GET  /v3/auth/me

GET    /v3/admin/iam/local-users/list
POST   /v3/admin/iam/local-users
PUT    /v3/admin/iam/local-users/{username}
DELETE /v3/admin/iam/local-users/{username}
```

Admin local user APIs require `iam:admin`, `admin`, or `*` permissions.

## External mode

When AIHub is attached to another account system:

```yaml
auth:
  mode: external
  local:
    enabled: false
  providers:
    - name: main-admin
      type: jwt
      issuer: go-admin
      audience: skillhub
      jwksUrl: http://admin.example.com/.well-known/jwks.json
```

In this mode `/v3/auth/login` is disabled for local login, and AIHub only accepts external identities.

## Mixed mode

During migration, or when both humans and agents come from different sources:

```yaml
auth:
  mode: mixed
  local:
    enabled: true
  providers:
    - name: main-admin
      type: jwt
    - name: agent-platform
      type: jwt
```

This allows:

```text
local admin account
external enterprise users
external agent tokens
bootstrap API keys
```

all to map into the same authorization model.

## Security notes

- Change `auth.local.signingSecret` before production.
- Prefer `passwordHash` instead of `password` in production configs.
- Do not give Agent subjects `skill:publish`, `skill:label:update:stable`, or `system:admin` unless explicitly trusted.
- Local auth is a fallback and embedded IAM option; enterprise deployment should still prefer OIDC/JWT integration.
