# AIHub Group-First / Namespace-Free Registry Design

This version intentionally moves away from the Nacos namespace-first model.

## Final concept model

AIHub resource management is now organized as:

```text
Group -> Skill -> Version -> Label
```

Identity and authorization are separate:

```text
Subject/User/Agent/Service -> Access Space/RBAC/ABAC -> Permission on group/skill/proposal
```

## What changed

### Skill layer

The skill registry no longer uses `namespaceId` as a skill identity dimension.

Canonical APIs do not require `namespaceId`:

```http
GET  /v3/aihub/skills?groupName=novel-writing-suite
GET  /v3/aihub/skill/{skillName}
POST /v3/aihub/skills/upload
GET  /v3/client/ai/skills/{skillName}?label=stable
GET  /v3/client/ai/groups/{groupName}?label=stable
GET  /registry-global/.well-known/agent-skills/index.json
```

### Group layer

Group is the primary business grouping concept. It is used for:

- skill classification
- capability suites
- Agent loading packs
- version/label assembly for multiple skills

Example:

```text
group=novel-writing-suite
  skill=dialogue-card
  skill=scene-splitter
  skill=style-rewriter
```

### Access Space / RBAC layer

The old namespace management UI/API is retained as **Access Space** management only.
It is not used to separate or locate skills.

Access Spaces can still be used for:

- user/team/agent membership
- RBAC/ABAC authorization
- owner and reviewer boundaries
- external IAM mapping

## Legacy compatibility

Old Nacos-compatible endpoints are retained:

```http
GET /v3/admin/ai/skills/list?namespaceId=public
GET /v3/client/ai/skills?namespaceId=public&name=xxx
GET /registry/{namespaceId}/.well-known/agent-skills/index.json
```

However, `namespaceId` is deprecated and ignored by the skill registry. These endpoints set:

```http
X-AIHub-Warning: namespaceId is deprecated and ignored by AIHub registry
```

## Internal fields involving namespace

A code scan was performed. Namespace fields now fall into three categories:

### Retained for IAM / Access Space

These remain intentionally:

- `NamespaceInfo`
- `NamespaceMember`
- `NamespaceMemberQuery`
- token `namespaces` claim, meaning access spaces
- auth/ABAC `ResourceRef.NamespaceID`, meaning access space
- tables `aihub_namespace` and `aihub_namespace_member`

### Retained as legacy storage compatibility, hidden from canonical JSON

These are internal compatibility fields and should always contain `_global` for skill resources:

- `SkillRecord.NamespaceID`
- `SkillBase.NamespaceID`
- `SkillGroup.NamespaceID`
- `SkillProposal.NamespaceID`
- `SkillOverlay.NamespaceID`
- `SkillVersionFileList.NamespaceID`
- `SkillVersionFileContent.NamespaceID`
- `SkillVersionCompare.NamespaceID`
- runtime cache interface `namespaceID` arguments
- legacy MySQL columns `namespace_id` in skill/resource tables

The canonical API hides these fields and ignores incoming `namespaceId`.

### Deprecated in documentation/API examples

All new examples use namespace-free endpoints. Old Nacos-style examples are marked deprecated.

## Cache keys

Skill runtime cache is namespace-free:

```text
aihub:route:skill:{skillName}:{label}
aihub:version:skill:{skillName}:{version}
aihub:skillset:{groupName}:{label}
aihub:download:skill:{skillName}:{version}
```

Lock keys are also namespace-free:

```text
aihub:lock:skill:{skillName}
aihub:lock:skillset:{groupName}
```

## Object store paths

Object storage paths are namespace-free:

```text
skills/{skillName}/versions/{version}/skill.zip
skills/{skillName}/versions/{version}/SKILL.md
skills/{skillName}/versions/{version}/manifest.json
```

## Migration note

Existing installations may have old rows with different `namespace_id` values. The runtime now ignores incoming namespace values and reads/writes the canonical `_global` registry scope. If you previously created duplicate skill names under different namespaces, manually merge them before migrating to a single registry.
