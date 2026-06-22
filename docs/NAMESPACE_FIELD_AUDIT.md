# Namespace Field Audit

Result after group-first refactor:

## Removed from canonical API

- Skill list/detail/upload/lifecycle no longer require `namespaceId`.
- Runtime download no longer requires `namespaceId`.
- Registry discovery no longer uses `/registry/{namespaceId}`; use `/registry-global`.
- Frontend skill/group pages no longer expose namespace as a business dimension.

## Still present intentionally

### IAM / Access Space

- `NamespaceInfo.NamespaceID` — access space ID.
- `NamespaceMember.NamespaceID` — access space membership.
- `TokenInfo.Namespaces` — access-space claim for compatibility with existing auth config.
- `ResourceRef.NamespaceID` — access-space scope in the authorizer.

### Legacy compatibility/internal storage

- Skill/resource MySQL tables still include `namespace_id` for compatibility with previous migrations and Nacos-compatible APIs.
- Service/store method signatures still accept `namespaceID` but canonical handlers always pass `model.DefaultNamespace`.
- Incoming `namespaceId` on legacy endpoints is ignored by `httputil.Namespace`.

## Next optional hard migration

A future major migration can physically rename/drop columns:

- `ai_resource.namespace_id` -> `registry_scope` or remove
- `ai_resource_version.namespace_id` -> `registry_scope` or remove
- `ai_resource_group.namespace_id` -> remove
- `skillhub_star.namespace_id` -> remove
- `skillhub_rating.namespace_id` -> remove
- `skillhub_subscription.namespace_id` -> remove

This version avoids destructive DDL to prevent data loss during upgrade.
