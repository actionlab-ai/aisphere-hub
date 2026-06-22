# 前后端分离与参考 SkillHub 复用说明

## 1. 目录结构

本项目已经调整为前后端分离：

```text
backend/
  cmd/
  internal/
  configs/
  migrations/
  docs/
  go.mod

front/
  src/
  package.json
  vite.config.ts
  index.html
```

后端只负责 API 和可选静态资源托管；前端是独立 Vite 项目。

## 2. 参考项目中值得复用的前端能力

上传的 `skillhub-main` 前端有几个非常值得借鉴的设计：

| 能力 | 参考项目表现 | 本次处理 |
|---|---|---|
| Skill 卡片 | 卡片干净、有 namespace badge、版本和下载统计 | 已适配到 `front/src/pages/SkillsPage.tsx` |
| 文件目录 | FileTree、FileTreeNode 展示 ZIP 内文件 | 已增加文件树和预览 |
| 版本管理 | 版本状态、生命周期、版本列表 | 已改成版本时间线 |
| Markdown/代码预览 | markdown renderer、code renderer | 当前先做纯文本/代码预览，后续可接 Markdown renderer |
| 版本对比 | compare hunks | 当前先返回两版 SKILL.md，前端并排显示；后续可接 diff viewer |
| Governance | 审核 inbox、通知、活动 | 我们已有 Proposal/Overlay 基础，后续可做成工作台 |
| Namespace | namespace 成员、owner、transfer | 我们当前 namespace 只是隔离字段，后端还没做完整 namespace 成员体系 |
| Token | token 创建、过期、撤销 | 我们已有 Auth 设计，API Key 管理还未完整落地 |
| Security Audit | 扫描发现、严重级别、结论 | 我们目前只有轻量 Proposal validate，未接安全扫描 |

## 3. 本次已实现的前端优化

### 3.1 视觉层

已将原来的简单后台风格升级为：

- indigo/violet 渐变品牌色。
- 玻璃态侧边栏和顶栏。
- 卡片悬浮阴影和 glow。
- 粘性搜索工具条。
- 更清楚的详情抽屉。
- 版本状态 badge。

### 3.2 Skill 详情

详情抽屉新增 tab：

```text
概览
版本管理
文件目录
版本对比
运行时
```

### 3.3 文件树

新增：

```http
GET /v3/admin/ai/skills/version/files?namespaceId=public&skillName=x&version=0.0.1
GET /v3/admin/ai/skills/version/file?namespaceId=public&skillName=x&version=0.0.1&path=SKILL.md
```

前端会把平铺文件列表构造成目录树，点击文件后显示内容。

### 3.4 版本对比

新增：

```http
GET /v3/admin/ai/skills/version/compare?namespaceId=public&skillName=x&baseVersion=0.0.1&targetVersion=0.0.2
```

当前返回两版 `SKILL.md` 和文件列表，前端做并排预览。下一步可以引入 `react-diff-viewer-continued` 或自行实现 hunk diff。

## 4. 本次后端新增接口规范

### 4.1 获取版本文件列表

```http
GET /v3/admin/ai/skills/version/files
```

参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| namespaceId | 否 | 默认 public |
| skillName/name | 是 | Skill 名称 |
| version | 是 | 版本号 |

响应：

```json
{
  "namespaceId": "public",
  "skillName": "dialogue-card",
  "version": "0.0.1",
  "files": [
    {"path":"SKILL.md","name":"SKILL.md","type":"markdown","size":1234,"binary":false},
    {"path":"prompts/main.md","name":"main.md","type":"prompts","size":456,"binary":false}
  ]
}
```

### 4.2 获取版本文件内容

```http
GET /v3/admin/ai/skills/version/file
```

参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| namespaceId | 否 | 默认 public |
| skillName/name | 是 | Skill 名称 |
| version | 是 | 版本号 |
| path | 是 | 文件路径 |

响应：

```json
{
  "namespaceId": "public",
  "skillName": "dialogue-card",
  "version": "0.0.1",
  "path": "SKILL.md",
  "content": "---\nname: ...",
  "binary": false
}
```

### 4.3 对比两个版本

```http
GET /v3/admin/ai/skills/version/compare
```

参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| namespaceId | 否 | 默认 public |
| skillName/name | 是 | Skill 名称 |
| baseVersion | 是 | 基准版本 |
| targetVersion | 是 | 目标版本 |

响应：

```json
{
  "namespaceId": "public",
  "skillName": "dialogue-card",
  "baseVersion": "0.0.1",
  "targetVersion": "0.0.2",
  "baseSkillMd": "...",
  "targetSkillMd": "...",
  "baseFiles": [],
  "targetFiles": []
}
```

## 5. 参考项目后端中我们还没完全覆盖的能力

参考项目后端能力比我们当前 Go 版更偏完整产品化，以下能力值得后续继续补：

| 能力 | 说明 | 我们当前状态 |
|---|---|---|
| Namespace 成员体系 | 成员、角色、owner transfer、批量导入 | 还没做完整 namespace membership |
| Search 架构 | 搜索索引、标签搜索、语义向量 | 当前只有轻量 list/search |
| Social | star、rating、subscription | 未做 |
| Notification | 通知中心、SSE、偏好设置 | 未做 |
| Governance Workbench | 审核 inbox、活动、分页 | 当前只有 Proposal 基础接口 |
| Security Audit | 扫描 finding、严重级别、结论 | 未做 |
| Report | Skill 举报/处理 | 未做 |
| Token 管理 | 创建、过期、撤销 token | Auth 有设计，管理 API 未完整实现 |
| Idempotency | 幂等拦截、防重复提交 | 未做 |
| Rate Limit | 下载限流、匿名身份识别 | 未做 |
| Metrics | 业务指标、下载统计 | 下载计数有基础，指标未完整暴露 |
| Builtin Skills | 内置 Skill 初始化、远端包下载 | 未做完整内置 Skill 管理 |
| CLI 兼容 | publish/search/resolve/delete CLI API | 我们有 Runtime/Registry，但 CLI 专用 API 未补齐 |

## 6. 下一步建议

优先顺序建议：

1. 完成 namespace 成员/角色体系，让 namespace 不只是字符串。
2. 把 Agent Proposal 做成 Governance Inbox 工作台。
3. 引入 Markdown renderer 和 Diff viewer。
4. 补 token/API key 管理页面和后端 API。
5. 增加审计日志、幂等 key、下载限流。
6. 做 OpenAPI 生成 TypeScript Client，减少手写 API 漂移。
