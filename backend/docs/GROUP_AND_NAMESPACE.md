# Namespace 与 Skill Group 的边界

## Namespace 是隔离边界

Namespace 用来做租户、环境、项目或组织隔离。它解决的是“这批资源属于谁、在哪个环境里生效”的问题。

典型用法：

- `dev`、`test`、`prod`：环境隔离。
- `tenant-a`、`tenant-b`：多租户隔离。
- `novel-platform`、`ops-platform`：业务域隔离。

同名 Skill 可以存在于不同 namespace，互不影响：

```text
namespace=dev   skill=dialogue-card version=0.0.3
namespace=prod  skill=dialogue-card version=0.0.1
```

Runtime 下载 Skill 时必须带 `namespaceId`，服务端只会在这个 namespace 内解析 name、version、label。

## Skill Group 是能力包/装配单元

Skill Group 用来把同一个 namespace 内的一组相关 Skill 组合成一个“能力包”。它解决的是“Agent 这次要一次性加载哪些 Skill”的问题。

典型用法：

- `novel-writing-suite`：小说写作能力包。
- `ops-troubleshooting-suite`：运维排障能力包。
- `dialogue-analysis-suite`：对话拆解能力包。

Group 里的成员可以按 version 固定，也可以按 label 动态解析：

```json
{
  "groupName": "novel-writing-suite",
  "members": [
    {"skillName": "dialogue-card", "label": "stable", "required": true, "order": 1},
    {"skillName": "scene-splitter", "version": "0.0.2", "required": true, "order": 2},
    {"skillName": "style-checker", "label": "gray", "required": false, "order": 3}
  ]
}
```

## 它们的关系

```text
Namespace = 边界 / 租户 / 环境
Group     = namespace 内的 Skill 编排集合
Skill     = 具体能力
Version   = Skill 的不可变版本
Label     = latest/stable/gray 等运行时路由
```

一个 Group 必须属于某个 namespace。Group 不能跨 namespace 引用 Skill。这样可以避免 dev 环境的 Skill 被 prod 的 Group 意外加载。

## 为什么 Group 不是 Namespace

Namespace 是隔离。Group 是组合。

错误理解：

```text
用 namespace=novel-writing 存小说技能组
```

这样会把“业务隔离”和“能力组合”混在一起。后面要做 prod/test、多租户、权限和灰度时会很乱。

推荐理解：

```text
namespace=prod
  group=novel-writing-suite
    skill=dialogue-card@stable
    skill=scene-splitter@stable

namespace=dev
  group=novel-writing-suite
    skill=dialogue-card@latest
    skill=scene-splitter@gray
```

这样同一个 group 名称可以在不同 namespace 里有不同装配结果。

## Agent Runtime 如何使用 Group

Agent 不需要逐个知道 Skill 名称，可以只配置：

```yaml
skill_group:
  namespaceId: prod
  groupName: novel-writing-suite
  label: stable
```

启动时请求：

```http
GET /v3/client/ai/skill-groups?namespaceId=prod&groupName=novel-writing-suite&label=stable
```

返回结果会包含已经解析好的成员 Skill、版本、md5 和加载顺序。
