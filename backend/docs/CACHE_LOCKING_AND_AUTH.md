# Cache locking and account / permission design

## 1. Redis is not globally locked

Do not lock the whole Redis instance. The service uses resource-level locks:

- `aihub:lock:skill:{namespace}:{skillName}`
- `aihub:lock:skillset:{namespace}:{groupName}`

A write operation acquires the corresponding lock, updates the source of truth
(MySQL/S3), then evicts runtime caches. Runtime reads briefly wait for the same
resource lock before trusting Redis. This avoids the race where a reader loads
old route/version metadata while a writer is publishing or changing labels.

## 2. Read path

Runtime skill download:

1. Wait briefly if `lock:skill:{ns}:{name}` exists.
2. Read Redis route cache: `label -> version`.
3. Read Redis version cache: `version -> md5/objectKey`.
4. If miss, read MySQL and refill Redis.
5. If `md5` matches client md5, return `304`.
6. Otherwise read zip from S3/MinIO.

Group manifest:

1. Wait briefly if `lock:group:{ns}:{groupName}` exists.
2. Read Redis group manifest.
3. If miss, read MySQL group and members, resolve member versions, then refill Redis.

## 3. Write path

Admin write path:

1. Acquire resource-level write lock.
2. Write MySQL/S3.
3. Delete route/version/index/group caches.
4. Release lock.

Proposal creation does not clear formal runtime caches because proposal/overlay
are not online runtime versions. Only approve/promote clears the formal Skill
runtime caches.

## 4. Redis single and cluster

The same lock contract supports Redis single and Redis Cluster. For Cluster,
pattern eviction scans all shards. Locks use `SET NX PX` and Lua compare-delete
so that one writer cannot delete another writer's lock.

## 5. Account model

Subjects are unified:

- human
- organization
- agent
- service

A Subject receives roles and permissions scoped by namespace.

Typical permissions:

- `skill:read`
- `skill:admin:read`
- `skill:admin:write`
- `skill:group:write`
- `skill:proposal:create`
- `skill:proposal:read`
- `skill:overlay:read`
- `skill:proposal:review`

Client API remains read-only. Agent API can create proposal and read overlay,
but cannot publish or change labels unless explicitly granted admin permissions.

## 6. Bootstrap auth

The current implementation supports config bootstrap API keys. MySQL tables are
created for durable account/RBAC management and can be wired into CRUD APIs next.
