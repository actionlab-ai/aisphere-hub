# SkillHub Review and Hardening Notes

This document records the first engineering review for the uploaded `aisphere-hub` package.

## Fixed in this pass

1. **Module identity**
   - Changed backend module from `github.com/example/gin-skillhub` to `github.com/actionlab-ai/aisphere-hub/backend`.
   - Updated all internal imports.
   - Removed the local `replace ../../aisphere-auth` so the repo can be consumed outside the original local workspace.

2. **aisphere-auth SDK integration**
   - The SkillHub wrapper now uses the configured HTTP timeout.
   - The wrapper no longer passes `nil` context into the SDK; it uses `context.Background()` when the caller has no explicit context.
   - Added `WriteAudit` and `ListAudit` forwarding methods to the wrapper.

3. **Authorization mapping**
   - `authz.provider=aisphere-auth` now maps SkillHub resources to AI Sphere policy objects such as:
     - `skillhub:aihub:skill:*`
     - `skillhub:skill:{name}`
     - `skillhub:aihub:skillset:*`
     - `skillhub:aihub:proposal:*`
     - `skillhub:audit:*`
     - `skillhub:admin:*`
   - It also maps SkillHub internal actions such as `skill:admin:read` and `skill:publish` to platform action verbs such as `read`, `create`, `update`, `delete`, `publish`, `approve`, `reject`, `admin:read`, `admin:write`.

4. **Audit forwarding**
   - Existing local audit is preserved.
   - When `aisphereAuth.enabled=true`, successful write operations are mirrored to aisphere-auth `/audit/events` using the public SDK.
   - Mirroring is best-effort and does not block the business operation.

5. **Security cleanup**
   - Active example configs no longer contain the previously embedded public IP, Redis password, MinIO secret, or Casdoor client secret.
   - Examples now use environment placeholders.
   - The OIDC redirect development allowlist no longer contains a hardcoded public IP.

## Current limitations

1. The backend depends on `github.com/actionlab-ai/aisphere-auth v0.1.0`. Make sure that tag exists in the auth repo before running CI from a clean environment.
2. The execution environment used for this review cannot download Go or npm dependencies, so full `go test ./...` and `npm run lint` could not be completed here.
3. Frontend lint/build should be run after `npm ci` in a network-enabled environment.
4. The old Casdoor/Casbin compatibility mode is retained. Production should prefer the `aisphere-auth` route after the platform seed is imported.

## Suggested next PRs

1. Add GitHub Actions CI for:
   - `go test ./...`
   - `go vet ./...`
   - `npm ci && npm run lint && npm run build`
2. Add deployment manifests for:
   - SkillHub backend
   - SkillHub frontend static console
   - ConfigMap/Secret example
   - readiness/liveness probes
3. Add a typed audit abstraction inside SkillHub:
   - `internal/audit.Service`
   - local sink
   - aisphere-auth sink
   - async queue with retry/backoff
