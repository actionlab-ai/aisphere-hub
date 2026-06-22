# AIHub 工程化设计说明

本版本开始按“接口优先”改造，目标是外部兼容 Nacos AIHub 契约，内部支持可替换实现。

## 1. 三个核心 Port

- `ports.ObjectStore`：文件存储抽象，生产默认 MinIO/S3，开发可用 LocalFS/Memory。
- `ports.Cache` / `ports.RuntimeCache`：缓存抽象，生产默认 Redis，支持单点 Redis 和 Redis Cluster，开发可用 Memory/Noop。
- `ports.Repository`：元数据仓储抽象，生产默认 MySQL，开发可用 Memory/JSON。

上层业务只依赖接口，不直接依赖 Redis SDK、S3 SDK、MySQL Driver。

## 2. Redis 单点和 Cluster 适配

推荐使用 `go-redis` 的 `UniversalClient` 思路实现一个 adapter：

```go
if mode == "cluster" {
  client = redis.NewClusterClient(&redis.ClusterOptions{Addrs: addrs})
} else {
  client = redis.NewClient(&redis.Options{Addr: addrs[0], DB: db})
}
```

对业务层暴露同一个 `ports.Cache` 接口。

Redis Key 设计：

```text
aihub:route:{namespace}:{type}:{name}:{label}
aihub:version:{namespace}:{type}:{name}:{version}
aihub:skillset:{namespace}:{group}:{label}
aihub:download:{namespace}:{type}:{name}:{version}
```

Cluster 注意点：

- 不在业务主链路使用跨 slot 多 key 事务。
- 删除时支持按精确 key 删除；通配删除只作为本地 memory 能力，Redis Cluster 生产实现应使用索引集合或短 TTL。
- download counter 用 `INCRBY`，天然适配 Cluster。

## 3. Skill Group 能力

新增一等资源：`SkillGroup`。

一个 group 管理一组相关 skill，例如：

```json
{
  "name": "novel-writing-suite",
  "displayName": "小说写作技能组",
  "members": [
    {"skillName":"dialogue-card", "label":"stable", "required":true, "order":1},
    {"skillName":"scene-split", "version":"0.0.3", "required":true, "order":2}
  ]
}
```

新增接口：

```text
GET    /v3/admin/ai/skill-groups/list
GET    /v3/admin/ai/skill-groups
POST   /v3/admin/ai/skill-groups
PUT    /v3/admin/ai/skill-groups
DELETE /v3/admin/ai/skill-groups
POST   /v3/admin/ai/skill-groups/bind
DELETE /v3/admin/ai/skill-groups/bind
GET    /v3/client/ai/skill-groups
```

`/v3/client/ai/skill-groups` 返回解析后的 group manifest，Agent 可以一次拿到这一组 skill 的 resolved version 和 downloadUrl。

## 4. 缓存一致性策略

采用组合拳，不依赖单一强一致缓存：

### 4.1 MySQL/S3 是事实来源

Redis 永远不是事实来源。Redis 丢失时最多降级查 MySQL/S3。

### 4.2 写后失效

发布、上线、下线、更新 label、更新 group member 后，删除：

```text
aihub:route:{ns}:skill:{name}:*
aihub:version:{ns}:skill:{name}:*
aihub:skillset:{ns}:{group}:*
```

### 4.3 短 TTL

Runtime route/version/group manifest 都设置短 TTL，例如 60s ~ 10min。即使失效消息丢了，也会自动收敛。

### 4.4 不变版本内容可长缓存

版本发布后不可变，`skill.zip` 的 md5/sha256 不变；runtime 下载支持 `md5 + 304`。

### 4.5 读穿透回填

Redis miss 时查 MySQL，再回填 Redis。

### 4.6 Group 缓存独立

Group manifest 独立缓存。修改 group member 时只清 group manifest；修改 skill label/version 时清 skill route/version，同时建议异步清理引用该 skill 的 group manifest。

后续生产实现可增加 `group_member_index` 表或 Redis Set：

```text
skillhub:reverse-group:{namespace}:skill:{skillName} -> [groupName...]
```

用于快速反向清理 group 缓存。

## Production adapters added in this edition

### MySQL repository

`internal/store/mysql` implements the same `store.Backend` contract as the local JSON store. The service layer does not know whether persistence is local JSON or MySQL.

Environment switch:

```bash
export SKILLHUB_STORE=mysql
export SKILLHUB_MYSQL_DSN='root:CHANGE_ME_PASSWORD@tcp(127.0.0.1:3306)/aihub?parseTime=true&charset=utf8mb4&loc=Local'
```

Run `migrations/001_init_resource_group.sql` before startup.

### Redis cache: single and cluster

`internal/cache/redis` uses go-redis UniversalClient style wiring. In single mode it creates `redis.Client`; in cluster mode it creates `redis.ClusterClient`. Wildcard invalidation uses SCAN. In cluster mode it scans every shard with `ForEachShard`, so group/route/version invalidation does not miss keys on other slots.

Single node:

```bash
export SKILLHUB_CACHE=redis
export SKILLHUB_REDIS_MODE=single
export SKILLHUB_REDIS_ADDRS=127.0.0.1:6379
```

Cluster:

```bash
export SKILLHUB_CACHE=redis
export SKILLHUB_REDIS_MODE=cluster
export SKILLHUB_REDIS_ADDRS=10.0.0.1:6379,10.0.0.2:6379,10.0.0.3:6379
```

### MinIO/S3 object store

`internal/objectstore/s3` implements `ports.ObjectStore` using the MinIO SDK. It works with MinIO and S3-compatible storage.

```bash
export SKILLHUB_OBJECT_STORE=minio
export SKILLHUB_S3_ENDPOINT=127.0.0.1:9000
export SKILLHUB_S3_ACCESS_KEY=minioadmin
export SKILLHUB_S3_SECRET_KEY=minioadmin
export SKILLHUB_S3_BUCKET=skillhub
export SKILLHUB_S3_USE_SSL=false
```

Uploaded skill versions now write these objects:

```text
namespaces/{namespaceId}/resources/skill/{skillName}/versions/{version}/skill.zip
namespaces/{namespaceId}/resources/skill/{skillName}/versions/{version}/SKILL.md
namespaces/{namespaceId}/resources/skill/{skillName}/versions/{version}/manifest.json
```

MySQL keeps the version metadata and `storage` JSON. Object storage keeps the immutable file content.

### Cache consistency model

The project uses a conservative consistency strategy:

1. MySQL and S3/MinIO are the source of truth.
2. Redis is only a cache and counter buffer.
3. Write path saves the source of truth first, then deletes route/version/group cache keys.
4. Runtime cache keys have short TTL.
5. Version content is immutable after publish, so version cache is safe to keep longer than route cache.
6. Label and group changes always invalidate broad wildcard keys.
7. Client download keeps md5/ETag/304 semantics to protect runtime startup.
