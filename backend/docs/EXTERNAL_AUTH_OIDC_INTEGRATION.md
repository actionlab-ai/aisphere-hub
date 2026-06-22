# External Auth / OIDC / OAuth2 Integration

AIHub should not be the primary identity provider. It is designed as a Resource Server:

```text
External IdP / Admin / Agent Platform
  -> OIDC / OAuth2 JWT / JWT / introspection / API key
  -> AIHub AuthProvider
  -> SubjectMapper
  -> AIHub RBAC/ABAC authorization
```

## Supported provider types

### 1. `oidc`

Use this for Keycloak, Casdoor, Authentik, Dex, or any standard OIDC provider.

```yaml
auth:
  providers:
    - name: company-oidc
      type: oidc
      issuer: https://idp.example.com/realms/aihub
      audience: aihub-api
      clientId: aihub-admin
      clientSecret: ${SKILLHUB_OIDC_CLIENT_SECRET}
      redirectUrl: https://aihub.example.com/oauth/callback
      scopes: [openid, profile, email, groups]
```

If `jwksUrl` is omitted, AIHub discovers it from:

```text
{issuer}/.well-known/openid-configuration
```

### 2. `jwt`

Use this when another application, such as your Go Admin or Agent Platform, signs JWTs.

```yaml
auth:
  providers:
    - name: main-admin
      type: jwt
      issuer: go-admin
      audience: skillhub
      jwksUrl: http://admin.example.com/.well-known/jwks.json
```

AIHub validates:

```text
RS256 signature
issuer
audience
exp / nbf
```

### 3. `introspection`

Use this when the external system returns opaque bearer tokens that cannot be locally decoded.

```yaml
auth:
  providers:
    - name: external-sso
      type: introspection
      introspectionUrl: https://sso.example.com/oauth/introspect
      clientId: skillhub
      clientSecret: ${SKILLHUB_INTROSPECTION_SECRET}
```

AIHub calls the external endpoint and expects `active=true` plus user/role/group fields.

### 4. `api_key`

Use this for bootstrap, offline mode, service accounts, or simple Agent integration.

```yaml
auth:
  providers:
    - name: bootstrap
      type: api_key
      header: X-API-Key
      keys:
        - name: admin
          subjectId: user:admin
          subjectType: human
          token: CHANGE_ME_ADMIN_TOKEN
          permissions: ['*']
          namespaces: ['*']
```

## Subject mapping

External tokens are mapped into AIHub subjects:

```text
human          user:peng
organization   org:novel-team
agent          agent:dialogue_skill_batch_worker
service        service:ci-publisher
```

AIHub maps claims using `claimMapping`:

```yaml
claimMapping:
  subject: sub
  subjectType: typ
  username: preferred_username
  email: email
  groups: groups
  roles: roles
  organization: org
```

If the external subject does not already include a prefix, AIHub creates one from `subjectType`, for example:

```text
sub=dialogue_skill_batch_worker, typ=agent
  -> agent:dialogue_skill_batch_worker
```

## Role mapping

External groups/roles should not directly become AIHub permissions. They should be mapped.

```yaml
roleMappings:
  - provider: main-admin
    externalGroup: ai-platform-admin
    internalRoles: [aihub-admin]
    permissions: ['*']
    namespaces: ['*']

  - provider: agent-platform
    externalRole: agent
    subjectType: agent
    internalRoles: [skill-agent]
    permissions:
      - skill:read
      - skill:proposal:create
      - skill:proposal:read
      - skill:overlay:read
    namespaces: [dev, public]
```

## Permission boundaries

Client API remains read-only:

```text
/v3/client/ai/skills -> skill:read
```

Agent proposal API allows low-risk write into candidate/proposal space only:

```text
/v3/agent/ai/skill-proposals -> skill:proposal:create
```

Admin APIs require stronger permissions:

```text
/v3/admin/ai/skills       -> skill:admin:read / skill:admin:write
/v3/admin/ai/skill-groups -> skill:group:read / skill:group:write
/v3/admin/ai/skill-proposals -> skill:proposal:review
```

## Runtime behavior

The middleware attempts providers in order:

```text
Bearer JWT/OIDC
Bearer opaque token introspection
X-API-Key
legacy bootstrap apiKeys
```

On success, it injects `principal` into the Gin context. You can inspect it with:

```http
GET /v3/admin/iam/whoami
```

## Important design rule

AIHub trusts external systems for identity, but AIHub remains the authority for AIHub resources:

```text
External Auth: who are you?
AIHub AuthZ: can you operate this namespace/group/skill/proposal?
```
