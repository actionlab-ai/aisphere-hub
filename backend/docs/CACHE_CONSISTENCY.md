# 缓存一致性设计

本项目采用 Cache-Aside 模式，事实来源永远是 MySQL + ObjectStore，Redis 只负责加速运行时读取。

## 读路径

Runtime 下载 Skill：

```text
1. 如果请求是 label=latest/stable/gray：先读 Redis route key
   aihub:route:{namespace}:skill:{name}:{label} -> version

2. route miss 时查 MySQL ai_resource.labels / online version，解析出 version，并回填 Redis route。

3. 拿到 version 后读 Redis version key
   aihub:version:{namespace}:skill:{name}:{version} -> VersionRecord(md5/storage/status)

4. version miss 时查 MySQL ai_resource_version，回填 Redis version。

5. 如果 client md5 == current md5，返回 304。

6. 否则从 MinIO/S3 读取 skill.zip 返回。
```

Runtime 解析 Skill Group：

```text
1. 先读 Redis group manifest
   aihub:skillset:{namespace}:{groupName}:{label}

2. miss 时查 MySQL group + group_member，再解析每个成员的 skill label/version。

3. 回填 group manifest 缓存。
```

## 写路径

所有管理端写操作遵循：

```text
先写 MySQL / S3 成功
再删除 Redis 缓存
失败则返回错误，不更新缓存
```

不采用“先删缓存再写数据库”，避免并发读把旧数据重新回填到缓存。

## 哪些操作会失效缓存

Skill 写操作：

```text
upload / draft / update draft / submit / publish / labels / metadata / scope / online / offline / delete
```

会删除：

```text
aihub:route:{namespace}:skill:{skillName}:*
aihub:version:{namespace}:skill:{skillName}:*
```

同时会扫描 namespace 内的 Skill Group，如果 group member 引用了这个 skill，则删除：

```text
aihub:skillset:{namespace}:{groupName}:*
```

Group 写操作：

```text
save group / delete group / bind member / unbind member
```

会删除：

```text
aihub:skillset:{namespace}:{groupName}:*
```

## Redis Single 和 Redis Cluster

Redis 通过 go-redis UniversalClient 适配。

Single：

```yaml
cache:
  provider: redis
  redis:
    mode: single
    addrs: ["127.0.0.1:6379"]
```

Cluster：

```yaml
cache:
  provider: redis
  redis:
    mode: cluster
    addrs:
      - "10.0.0.1:6379"
      - "10.0.0.2:6379"
      - "10.0.0.3:6379"
```

通配删除在 Cluster 模式下会 `ForEachShard + SCAN + DEL`，避免只扫到单个分片。

## 为什么还需要 TTL

即使写后失效做得很完整，也需要短 TTL 兜底，防止：

```text
1. Redis 删除失败
2. 服务发布中断
3. 多实例之间短暂并发读写
4. 手工改库导致缓存未知
```

推荐：

```yaml
runtime:
  routeCacheTTLSeconds: 300
  versionCacheTTLSeconds: 600
  groupCacheTTLSeconds: 60
```

## 下载计数

Runtime 下载计数走 Redis INCR：

```text
aihub:download:{namespace}:skill:{name}:{version}
```

后续生产版建议增加一个后台 flusher，定时把 Redis 计数刷回 MySQL。当前版本已经把计数入口抽象在 RuntimeCache.IncrementDownload，后续加 flusher 不影响 API。

## Agent Proposal 与缓存

Agent Proposal / Overlay 属于低信任候选区，不会改变正式 Skill 的 runtime 路由，因此：

- `POST /v3/agent/ai/skill-proposals` 不清理 route/version/index/group 缓存。
- `GET /v3/agent/ai/skill-overlays` 只读取 proposal/overlay，不参与正式 Skill 下载缓存。
- 只有 `approve` 将 proposal 提升为正式版本，或同时更新 label / online 状态时，才执行正式 Skill 的缓存失效。

这样可以避免 Agent 高频提交候选 skill_delta 导致线上运行时缓存抖动。
