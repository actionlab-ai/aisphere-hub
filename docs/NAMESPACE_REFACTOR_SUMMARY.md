# Namespace Refactor Summary

## Decision

SkillHub is now a **group-first, namespace-free skill registry**.

Namespace is not a skill dimension. It is retained only as an IAM/Access-Space compatibility concept.

## Canonical model

```text
Group -> Skill -> Version -> Label
```

## Authorization model

```text
Subject/User/Agent/Service -> Access Space -> Role/Policy -> Group/Skill/Proposal permission
```

## Removed from skill-facing APIs

The following canonical endpoints do not accept `namespaceId`:

- `/v3/aihub/skills`
- `/v3/aihub/skills/upload`
- `/v3/aihub/skill/{skillName}`
- `/v3/aihub/skill/{skillName}/versions/{version}`
- `/v3/aihub/groups`
- `/v3/client/ai/skills/{skillName}`
- `/v3/client/ai/groups/{groupName}`
- `/registry-global/.well-known/agent-skills/index.json`

## Kept only for compatibility

Old Nacos-like APIs still exist, but `namespaceId` is ignored for skill lookup.

## Field scan result

Namespace-related fields remain only in:

1. Access Space / IAM structs and APIs.
2. Deprecated Nacos-compatible API handlers.
3. Internal storage compatibility fields, hidden from canonical JSON and forced to `_global`.

See `docs/NAMESPACE_FIELD_AUDIT.md` for details.
