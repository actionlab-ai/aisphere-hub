# SkillHub 快速使用指南

## 1. 本地首次初始化模式

启动后端：

```bash
cd backend
go mod tidy
go run ./cmd/skillhub --config configs/local-first-setup.yaml
```

启动前端：

```bash
cd front
npm install
npm run dev
```

查看是否需要初始化：

```bash
curl http://127.0.0.1:8848/v3/auth/setup/status
```

初始化管理员：

```bash
curl -X POST http://127.0.0.1:8848/v3/auth/setup \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"ChangeMe_123!","displayName":"Platform Admin"}'
```

登录：

```bash
curl -X POST http://127.0.0.1:8848/v3/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"ChangeMe_123!"}'
```

## 2. 生产模式

准备 MySQL、Redis、MinIO/S3 后：

```bash
cd backend
go run ./cmd/skillhub --config configs/mysql-redis-minio.yaml
```

服务会自动：

```text
1. 解析 MySQL DSN
2. 自动建库
3. 执行 migrations/*.sql
4. 启动 API
```

## 3. 构建前端

```bash
cd front
npm install
npm run build
```

后端配置：

```yaml
web:
  enabled: true
  dir: "../front/dist"
  routePrefix: "/ui"
```

访问：

```text
http://127.0.0.1:8848/ui/
```

## 4. 上传 Skill

Skill ZIP 结构：

```text
novel-dialogue-card/
  SKILL.md
  prompts/dialogue.md
  examples/case1.md
```

上传：

```bash
curl -F 'namespaceId=public' \
  -F 'overwrite=true' \
  -F 'targetVersion=0.0.1' \
  -F 'commitMsg=init' \
  -F 'file=@novel-dialogue-card.zip' \
  http://127.0.0.1:8848/v3/admin/ai/skills/upload
```

发布上线：

```bash
curl -X POST -d 'namespaceId=public&skillName=novel-dialogue-card&version=0.0.1' \
  http://127.0.0.1:8848/v3/admin/ai/skills/publish

curl -X POST -d 'namespaceId=public&skillName=novel-dialogue-card&version=0.0.1' \
  http://127.0.0.1:8848/v3/admin/ai/skills/online
```

Runtime 下载：

```bash
curl -i -o novel-dialogue-card.zip \
  'http://127.0.0.1:8848/v3/client/ai/skills?namespaceId=public&name=novel-dialogue-card&label=latest'
```

## 5. Group 示例

创建能力包：

```bash
curl -X POST http://127.0.0.1:8848/v3/admin/ai/skill-groups \
  -H 'Content-Type: application/json' \
  -d '{
    "namespaceId":"public",
    "groupName":"novel-suite",
    "displayName":"小说写作技能组",
    "members":[{"skillName":"novel-dialogue-card","label":"latest","required":true,"order":1}]
  }'
```

Runtime 拉取 group：

```bash
curl 'http://127.0.0.1:8848/v3/client/ai/skill-groups?namespaceId=public&groupName=novel-suite&label=latest'
```

## 6. Agent Proposal 示例

```bash
curl -X POST http://127.0.0.1:8848/v3/agent/ai/skill-proposals \
  -H 'Content-Type: application/json' \
  -d '{
    "namespaceId":"public",
    "skillName":"novel-dialogue-card",
    "baseVersion":"0.0.1",
    "proposalType":"delta",
    "source":{"agentId":"dialogue_skill_batch_worker","runId":"run_001"},
    "reason":"发现缺少权力反转式对话规则",
    "delta":{"addSections":[{"title":"权力反转式对话","content":"弱势方通过证据完成话语权反转。"}]},
    "evidence":{"score":0.82}
  }'
```

Admin 审核：

```bash
curl -X POST http://127.0.0.1:8848/v3/admin/ai/skill-proposals/{proposalId}/validate
curl -X POST http://127.0.0.1:8848/v3/admin/ai/skill-proposals/{proposalId}/approve \
  -H 'Content-Type: application/json' \
  -d '{"publish":true,"online":true,"label":"gray","operator":"admin"}'
```
