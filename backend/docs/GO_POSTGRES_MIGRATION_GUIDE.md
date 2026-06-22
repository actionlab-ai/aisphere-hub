# Go + PostgreSQL migration 规范

本规范用于需要由 Go 服务管理 PostgreSQL 数据库和 schema 的模块。目标是：首次部署可自动初始化，重复启动幂等，多副本启动不互相破坏，并能审计每一次 schema 变更。

## 1. 启动分层

将启动过程固定为三个阶段，禁止混在一个连接或一个事务中处理。

1. **建库（cluster 级）**：从业务 DSN 解析目标 `dbname`，用同一认证参数连接维护库 `postgres`，检查 `pg_database`，必要时执行 `CREATE DATABASE`。
2. **连接业务库（database 级）**：使用原始业务 DSN 建立连接池并 `Ping`。
3. **迁移 schema（schema 级）**：在业务库中获取迁移锁、确认迁移记录、按顺序执行未应用的 migration、记录 checksum 和时间。

`CREATE DATABASE` 不能在事务块中执行，因此它必须独立于表结构 migration。建库账户必须有 `CREATEDB` 权限；服务不应试图提升权限或吞掉权限错误。

## 2. 配置契约

```yaml
database:
  provider: postgres
  dsn: "host=db.example port=5432 user=app password=*** dbname=aisphere_hub sslmode=require"
  autoCreate: true

migration:
  enabled: true
  dir: ./migrations/postgres
```

- `database.autoCreate: true`：允许创建缺失的数据库，并运行 schema migration。
- `database.autoCreate: false`：不创建数据库，也不隐式变更 schema；数据库或表缺失应显式失败。
- `migration.enabled` 应用于已有数据库上的版本化 schema 迁移；生产环境可关闭并改由发布 job 执行。
- DSN 必须显式包含目标数据库名。不要依赖 PostgreSQL 或驱动的默认数据库名。

## 3. 建库规则

```go
cfg, err := pgxpool.ParseConfig(dsn)
if err != nil {
	return fmt.Errorf("parse postgres dsn: %w", err)
}
target := strings.TrimSpace(cfg.ConnConfig.Database)
if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(target) {
	return fmt.Errorf("invalid PostgreSQL database name %q", target)
}

adminCfg := cfg.ConnConfig.Copy()
adminCfg.Database = "postgres"
adminConn, err := pgx.ConnectConfig(ctx, adminCfg)
if err != nil {
	return fmt.Errorf("connect postgres server for database bootstrap: %w", err)
}
defer adminConn.Close(ctx)

_, err = adminConn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{target}.Sanitize())
```

要求：

- 库名只能来自 DSN，先校验，再通过 `pgx.Identifier{target}.Sanitize()` 作为标识符拼接；不能把标识符当 SQL 参数传递。
- `CREATE DATABASE` 前先用参数化查询检查 `pg_database`：`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`。
- 若创建失败，重新检查一次目录；另一实例已创建则继续，其他错误返回并保留上下文。
- 使用短超时（例如 5 到 15 秒）；维护连接必须 `Close`。

## 4. 版本化 schema migration

每个 PostgreSQL 模块应使用独立的 migration 目录：

```text
migrations/postgres/
  202606221530_create_aihub_document.up.sql
  202606221540_add_document_updated_index.up.sql
```

在业务库中维护记录表：

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

执行规则：

1. 按文件名字典序读取 `.up.sql`。
2. 计算每个文件 SHA-256；已记录版本的 checksum 不一致时立即失败，禁止静默重写历史 migration。
3. 对每个未应用 migration 开启独立事务，执行 SQL 后插入 `schema_migrations`，最后提交。
4. 在执行前取得 PostgreSQL advisory lock，确保多副本不会并行执行：`SELECT pg_advisory_lock(hashtext('module:migrations'))`；退出时释放锁。
5. `CREATE INDEX CONCURRENTLY`、`VACUUM` 等禁止在事务中的语句，必须单独标记为 non-transactional migration，并在文件头说明原因和回滚方式。

不要依赖 `CREATE TABLE IF NOT EXISTS` 作为长期 migration 策略。它可用于 migration 元数据表或首次 bootstrap，但无法记录字段类型、约束和索引的演进。

## 5. SQL 编写要求

- 每个 migration 只表达一个可审计的 schema 变化；不混入业务数据修复。
- DDL 使用明确的约束、默认值和 `NOT NULL`；新增非空字段必须给出在线升级策略，避免大表长时间锁定。
- 大表索引优先使用 `CREATE INDEX CONCURRENTLY`，并将它放入 non-transactional migration。
- 所有对象名使用稳定的小写 snake_case，避免依赖 PostgreSQL 的大小写折叠。
- 提供对应 `.down.sql` 仅用于可安全回滚的变更；涉及数据删除或不可逆转换时，明确标记不可逆，不伪造回滚脚本。

## 6. Go 代码边界

- `bootstrap.go`：只负责 DSN 解析、维护库连接、幂等建库。
- `migrator.go`：只负责 migration 文件发现、锁、事务、checksum 与版本表。
- `store.go`：只建立业务连接池，并在配置允许时调用 bootstrap 与 migrator；不内嵌多段 SQL 历史。
- 单元测试使用窄接口模拟数据库目录检查和建库竞争；集成测试通过 `HUB_POSTGRES_DSN` 在真实 PostgreSQL 上执行完整迁移。

## 7. 上线检查清单

- [ ] 业务 DSN 含 `dbname`，并使用目标环境要求的 TLS 配置。
- [ ] 自动建库角色只在开发/首次部署具备 `CREATEDB`；生产通常由 migration job 使用受控角色执行。
- [ ] migration job 或服务启动日志输出当前版本、已跳过版本和失败文件名，但不输出 DSN 密码。
- [ ] 多副本环境验证 advisory lock 行为。
- [ ] 对大表 migration 做锁时长和回滚演练。
- [ ] 发布前在与生产 PostgreSQL 主版本一致的环境运行 `go test ./...` 和 migration 集成测试。
