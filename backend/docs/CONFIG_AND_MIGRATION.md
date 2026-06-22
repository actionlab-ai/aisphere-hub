# Config 和 Migration

服务统一使用 YAML 配置。环境变量只作为临时覆盖，不作为主部署方式。

## 启动

```bash
go run ./cmd/skillhub --config configs/postgres-redis-minio.yaml
```

或者：

```bash
export SKILLHUB_CONFIG=configs/postgres-redis-minio.yaml
go run ./cmd/skillhub
```

## 自动建库和建表

配置：

```yaml
database:
  provider: postgres
  dsn: "root:CHANGE_ME_PASSWORD@tcp(127.0.0.1:3306)/aihub?charset=utf8mb4&parseTime=true&loc=Local"
  autoCreate: true
  charset: "utf8mb4"
  collation: "utf8mb4_unicode_ci"

migration:
  enabled: true
  dir: "./migrations"
```

启动时流程：

```text
1. 解析 PostgreSQL DSN，取出 DBName，例如 skillhub。
2. 使用不带 DBName 的 server DSN 连接 PostgreSQL。
3. 执行 CREATE DATABASE IF NOT EXISTS `skillhub`。
4. 使用原始 DSN 连接 skillhub 数据库。
5. 创建 schema_migrations。
6. 扫描 migrations/*.sql。
7. 只执行未记录过的 migration。
8. 写入 schema_migrations。
```

所以首次部署不需要手工执行建表 SQL，也不需要提前创建数据库。只需要 PostgreSQL 账号有 `CREATE DATABASE` 权限。

如果线上 DBA 不允许应用自动建库，可以关闭：

```yaml
database:
  autoCreate: false
```

这种情况下由 DBA 先创建库，应用只负责建表 migration。
