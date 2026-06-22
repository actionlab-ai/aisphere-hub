# Agent Skill Proposal / Overlay 设计

## 目标

Agent 可以沉淀、修正、增强 Skill，但不能直接污染线上 `stable/latest/gray` 路由。Agent 写入必须进入低信任区：Proposal、Overlay、Candidate。正式 Skill 版本仍然由 Admin 或审核流水线发布。

## API 分层

- `/v3/client/ai/skills`：运行时只读下载正式 Skill。
- `/registry/{namespace}/.well-known/agent-skills`：外部发现正式 Skill。
- `/v3/admin/ai/skills`：管理员管理正式 Skill。
- `/v3/agent/ai/skill-proposals`：Agent 提交 Skill 修改建议。
- `/v3/agent/ai/skill-overlays`：Agent 读取临时 overlay。
- `/v3/admin/ai/skill-proposals`：管理员/流水线验证、批准、拒绝 proposal。

## 生命周期

```text
running agent
  -> skill_delta
  -> proposal(submitted)
  -> overlay(active)
  -> validate(validating)
  -> approve/reject
  -> promote to real skill version
  -> optional label update latest/gray/stable
```

Proposal 创建时不会清理 runtime cache，因为正式 Skill 没有变化。
只有 approve/promote 生成正式版本、修改 label 或 online 状态后，才清理：

```text
aihub:route:{namespace}:skill:{name}:*
aihub:version:{namespace}:skill:{name}:*
skillhub:index:{namespace}:agent-skills
aihub:skillset:{namespace}:*:*
```

## 提交 proposal

```http
POST /v3/agent/ai/skill-proposals
```

```json
{
  "namespaceId": "public",
  "skillName": "novel-dialogue-card",
  "baseVersion": "0.0.3",
  "proposalType": "delta",
  "source": {
    "agentId": "dialogue_skill_batch_worker",
    "sessionId": "sess_001",
    "runId": "run_001",
    "taskId": "task_001"
  },
  "reason": "发现缺少权力反转式对话规则",
  "delta": {
    "addSections": [
      {
        "title": "权力反转式对话",
        "content": "弱势方通过证据、身份或情报完成话语权反转。"
      }
    ]
  },
  "evidence": {
    "inputRefs": ["chapter_011"],
    "score": 0.82
  }
}
```

返回：

```json
{
  "proposalId": "sp_xxx",
  "status": "submitted",
  "overlayRef": "skill-overlay://public/novel-dialogue-card/sp_xxx",
  "candidateVersion": "0.0.4-candidate.1"
}
```

## 读取 overlay

```http
GET /v3/agent/ai/skill-overlays/{proposalId}
```

或者：

```http
GET /v3/agent/ai/skill-overlays?overlayRef=skill-overlay://public/novel-dialogue-card/sp_xxx
```

Overlay 给下一轮 Agent 作为临时上下文增强，不是正式 Skill。

## 审核/验证

```http
POST /v3/admin/ai/skill-proposals/{proposalId}/validate
POST /v3/admin/ai/skill-proposals/{proposalId}/approve
POST /v3/admin/ai/skill-proposals/{proposalId}/reject
```

Approve 示例：

```json
{
  "targetVersion": "0.0.4",
  "label": "gray",
  "publish": true,
  "online": true,
  "reviewer": "admin",
  "comment": "灰度验证通过"
}
```

## 数据库表

新增：

- `ai_skill_proposal`
- `ai_skill_overlay`
- `ai_skill_proposal_validation`

Migration 已放入 `migrations/001_init_resource_group.sql`，启动时会自动建库建表。
