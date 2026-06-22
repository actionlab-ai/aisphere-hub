# First setup for built-in local auth

When `auth.mode` is `local` or `mixed` and no external identity provider is configured, AIHub can initialize the first administrator through a one-time setup API.

## Why this exists

A fresh deployment should not require a hard-coded default password in `config.yaml`. The recommended production mode is:

```yaml
auth:
  enabled: true
  mode: "local"
  local:
    enabled: true
    autoCreateBootstrap: false
    setupEnabled: true
    setupToken: "${SKILLHUB_SETUP_TOKEN}" # optional but recommended when the service is reachable over a network
    users: []
```

On first boot, if the local account store has no active account, AIHub reports that setup is required. After the first admin is created, setup automatically becomes unavailable.

## APIs

### Check setup status

```http
GET /v3/auth/setup/status
```

Example response before initialization:

```json
{
  "localEnabled": true,
  "setupEnabled": true,
  "setupRequired": true,
  "mode": "local"
}
```

After initialization:

```json
{
  "localEnabled": true,
  "setupEnabled": true,
  "setupRequired": false,
  "mode": "local"
}
```

### Create the first admin

```http
POST /v3/auth/setup
Content-Type: application/json
```

```json
{
  "username": "admin",
  "password": "ChangeMe_123!",
  "displayName": "Platform Admin",
  "email": "admin@example.com",
  "organization": "default",
  "setupToken": "optional-token-from-config"
}
```

Response includes a local admin account and access tokens:

```json
{
  "data": {
    "account": {
      "username": "admin",
      "subjectId": "user:admin",
      "subjectType": "human",
      "roles": ["admin"],
      "permissions": ["*"],
      "namespaces": ["*"]
    },
    "tokens": {
      "accessToken": "...",
      "refreshToken": "...",
      "tokenType": "Bearer",
      "expiresIn": 3600
    }
  }
}
```

## Security behavior

- `/v3/auth/setup` is only useful while no active local account exists.
- Once an active account exists, the endpoint returns `409 Conflict`.
- If `setupToken` is configured, requests must provide the same token.
- Passwords are stored as PBKDF2-SHA256 hashes, not plaintext.
- The first setup admin receives `roles=["admin"]`, `permissions=["*"]`, and `namespaces=["*"]`.

## Frontend flow

The frontend should call `GET /v3/auth/setup/status` before showing the normal login page.

```text
setupRequired=true
  -> show first-admin setup page
  -> POST /v3/auth/setup
  -> store returned access token
  -> enter admin console

setupRequired=false
  -> show normal login / SSO page
```

## Config examples

A ready-to-use first setup config is available at:

```text
configs/local-first-setup.yaml
```

Start with:

```bash
go run ./cmd/skillhub --config configs/local-first-setup.yaml
```
