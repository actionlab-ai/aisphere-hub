import { request, toQuery } from './client';
import type {
  Page,
  Proposal,
  Skill,
  SkillGroup,
  AgentListItem,
  AgentResponse,
  AgentRuntimeSnapshot,
  AgentServiceRef,
  AgentUpsertRequest,
  RuntimeServicesSnapshot,
  SandboxEnsureRequest,
  SandboxStatus,
  SandboxToolCallRequest,
  SandboxToolCallResult,
  SandboxToolListResponse,
  ToolFailureRecord,
  ToolListItem,
  ToolResponse,
  ToolRuntimeSnapshot,
  ToolUpsertRequest,
  LocalUser,
  SkillFileList,
  SkillFileContent,
  SkillVersionCompare,
  SkillSocialStats,
  AuditLog,
  TokenInfo,
  MetricsSnapshot,
  Notification,
  NamespaceInfo,
  NamespaceMember,
  SkillDraft,
  GroupUpdate,
  AccessEvaluateResult,
  AccessOverview,
  AccessResourceTemplate,
  AccessQuickLink,
  SandboxProfile,
  ModelProfile,
  ResourceGrant,
  CreateShareRequest,
  ShareListResponse,
  AihubResourceType,
} from './types';

export const authApi = {
  setupStatus: () => request<{ setupRequired: boolean }>('/v3/auth/setup/status'),
  setup: (data: Record<string, unknown>) =>
    request<{ tokens: { accessToken: string; refreshToken: string } }>('/v3/auth/setup', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  login: (username: string, password: string) =>
    request<{ accessToken: string; refreshToken: string }>('/v3/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  me: () => request<Record<string, unknown>>('/v3/auth/me'),
};

export const accessApi = {
  overview: () => request<AccessOverview>('/v3/admin/access/overview'),
  resources: () => request<Page<AccessResourceTemplate>>('/v3/admin/access/resources'),
  links: () => request<Page<AccessQuickLink>>('/v3/admin/access/links'),
  evaluate: (subject: string, object: string, action: string) =>
    request<AccessEvaluateResult>('/v3/admin/access/evaluate', { method: 'POST', body: JSON.stringify({ subject, object, action }) }),
};

export const skillApi = {
  list: (params: Record<string, unknown> = {}) =>
    request<Page<Skill>>(`/v3/aihub/skills?${toQuery(params)}`),
  detail: (skillName: string) =>
    request<Skill>(`/v3/aihub/skill/${encodeURIComponent(skillName)}`),
  version: (skillName: string, version: string) =>
    request<unknown>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/versions/${encodeURIComponent(version)}`),
  upload: (file: File, overwrite = true, targetVersion = '', commitMsg = '') => {
    const fd = new FormData();
    fd.append('overwrite', String(overwrite));
    if (targetVersion) fd.append('targetVersion', targetVersion);
    if (commitMsg) fd.append('commitMsg', commitMsg);
    fd.append('file', file, file.name);
    return request<string>('/v3/aihub/skills/upload', { method: 'POST', body: fd });
  },
  batchUpload: (files: File[], overwrite = true) => {
    const fd = new FormData();
    fd.append('overwrite', String(overwrite));
    files.forEach((f) => fd.append('files', f, f.name));
    return request<string>('/v3/aihub/skills/upload/batch', { method: 'POST', body: fd });
  },
  remove: (skillName: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}`, { method: 'DELETE' }),
  publish: (skillName: string, version: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/publish`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    }),
  submit: (skillName: string, version: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/submit`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    }),
  online: (skillName: string, version?: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/online`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    }),
  offline: (skillName: string, version?: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/offline`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    }),
  labels: (skillName: string, labels: Record<string, string>) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/labels`, {
      method: 'PUT',
      body: JSON.stringify({ labels }),
    }),
  downloadUrl: (skillName: string, version: string) =>
    `/v3/aihub/skill/${encodeURIComponent(skillName)}/versions/${encodeURIComponent(version)}/download`,
  files: (skillName: string, version: string) =>
    request<SkillFileList>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/versions/${encodeURIComponent(version)}/files`),
  file: (skillName: string, version: string, path: string) =>
    request<SkillFileContent>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/versions/${encodeURIComponent(version)}/file?${toQuery({ path })}`),
  compare: (skillName: string, baseVersion: string, targetVersion: string) =>
    request<SkillVersionCompare>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/compare?${toQuery({ baseVersion, targetVersion })}`),
  draft: (data: SkillDraft) =>
    request<unknown>('/v3/aihub/skills/draft', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateDraft: (data: SkillDraft) =>
    request<unknown>('/v3/aihub/skills/draft', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteDraft: (skillName: string, version: string) =>
    request<unknown>(`/v3/aihub/skills/draft?skillName=${encodeURIComponent(skillName)}&version=${encodeURIComponent(version)}`, {
      method: 'DELETE',
    }),
  forcePublish: (skillName: string, version: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/force-publish`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    }),
  redraft: (skillName: string, version: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/redraft`, {
      method: 'POST',
      body: JSON.stringify({ version }),
    }),
  bizTags: (skillName: string, tags: string[]) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/biz-tags`, {
      method: 'PUT',
      body: JSON.stringify({ bizTags: tags }),
    }),
  metadata: (skillName: string, metadata: Record<string, unknown>) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/metadata`, {
      method: 'PUT',
      body: JSON.stringify({ metadata }),
    }),
  scope: (skillName: string, scope: string) =>
    request<string>(`/v3/aihub/skill/${encodeURIComponent(skillName)}/scope`, {
      method: 'PUT',
      body: JSON.stringify({ scope }),
    }),
};

export const groupApi = {
  list: (params: Record<string, unknown> = {}) =>
    request<Page<SkillGroup>>(`/v3/aihub/skillsets?${toQuery(params)}`),
  detail: (groupName: string) =>
    request<SkillGroup>(`/v3/aihub/skillset/${encodeURIComponent(groupName)}`),
  save: (group: SkillGroup) =>
    request<unknown>('/v3/aihub/skillsets', { method: 'POST', body: JSON.stringify(group) }),
  update: (groupName: string, group: GroupUpdate) =>
    request<unknown>(`/v3/aihub/skillset/${encodeURIComponent(groupName)}`, {
      method: 'PUT',
      body: JSON.stringify(group),
    }),
  remove: (groupName: string) =>
    request<unknown>(`/v3/aihub/skillset/${encodeURIComponent(groupName)}`, { method: 'DELETE' }),
  bind: (groupName: string, member: Record<string, unknown>) =>
    request<unknown>(`/v3/aihub/skillset/${encodeURIComponent(groupName)}/skills`, {
      method: 'POST',
      body: JSON.stringify(member),
    }),
  unbind: (groupName: string, skillName: string) =>
    request<unknown>(`/v3/aihub/skillset/${encodeURIComponent(groupName)}/skills/${encodeURIComponent(skillName)}`, {
      method: 'DELETE',
    }),
  groupSkills: (groupName: string) =>
    request<SkillGroup>(`/v3/aihub/skillset/${encodeURIComponent(groupName)}/skills`),
};


export const agentApi = {
  list: (params: Record<string, unknown> = {}) =>
    request<Page<AgentListItem>>(`/v3/aihub/agents?${toQuery(params)}`),
  detail: (agentId: string) =>
    request<AgentResponse>(`/v3/aihub/agents/${encodeURIComponent(agentId)}`),
  create: (data: AgentUpsertRequest) =>
    request<AgentResponse>('/v3/aihub/agents', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (agentId: string, data: AgentUpsertRequest) =>
    request<AgentResponse>(`/v3/aihub/agents/${encodeURIComponent(agentId)}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  remove: (agentId: string) =>
    request<unknown>(`/v3/aihub/agents/${encodeURIComponent(agentId)}`, { method: 'DELETE' }),
  resolve: (agentId: string, body: { runtimeId?: string; sessionId?: string; version?: string; label?: string; policy?: string } = {}) =>
    request<AgentRuntimeSnapshot>(`/v3/aihub/runtime/agents/${encodeURIComponent(agentId)}/resolve`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
};


export const runtimeServiceApi = {
  resolve: (services: AgentServiceRef[], body: { runtimeId?: string; sessionId?: string } = {}) =>
    request<RuntimeServicesSnapshot>('/v3/aihub/runtime/services/resolve', {
      method: 'POST',
      body: JSON.stringify({ ...body, services }),
    }),
};

export const sandboxApi = {
  list: (params: Record<string, unknown> = {}) =>
    request<Page<SandboxStatus>>(`/v3/aihub/runtime/sandboxes?${toQuery(params)}`),
  ensure: (body: SandboxEnsureRequest) =>
    request<SandboxStatus>('/v3/aihub/runtime/sandboxes', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  get: (sandboxId: string) =>
    request<SandboxStatus>(`/v3/aihub/runtime/sandboxes/${encodeURIComponent(sandboxId)}`),
  restart: (sandboxId: string) =>
    request<SandboxStatus>(`/v3/aihub/runtime/sandboxes/${encodeURIComponent(sandboxId)}/restart`, { method: 'POST' }),
  remove: (sandboxId: string, deleteWorkspace = false) =>
    request<unknown>(`/v3/aihub/runtime/sandboxes/${encodeURIComponent(sandboxId)}?${toQuery({ deleteWorkspace })}`, { method: 'DELETE' }),
  logsUrl: (sandboxId: string, tailLines = 200, container = 'sandbox') =>
    `/v3/aihub/runtime/sandboxes/${encodeURIComponent(sandboxId)}/logs?${toQuery({ tailLines, container })}`,
  tools: (sandboxId: string) =>
    request<SandboxToolListResponse>(`/v3/aihub/runtime/sandboxes/${encodeURIComponent(sandboxId)}/tools`),
  callTool: (sandboxId: string, body: SandboxToolCallRequest) =>
    request<SandboxToolCallResult>(`/v3/aihub/runtime/sandboxes/${encodeURIComponent(sandboxId)}/tools/call`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
};

export const toolApi = {
  list: (params: Record<string, unknown> = {}) =>
    request<Page<ToolListItem>>(`/v3/aihub/tools?${toQuery(params)}`),
  detail: (toolId: string) =>
    request<ToolResponse>(`/v3/aihub/tools/${encodeURIComponent(toolId)}`),
  create: (data: ToolUpsertRequest) =>
    request<ToolResponse>('/v3/aihub/tools', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  update: (toolId: string, data: ToolUpsertRequest) =>
    request<ToolResponse>(`/v3/aihub/tools/${encodeURIComponent(toolId)}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  remove: (toolId: string) =>
    request<unknown>(`/v3/aihub/tools/${encodeURIComponent(toolId)}`, { method: 'DELETE' }),
  resolve: (toolId: string, body: { runtimeId?: string; sessionId?: string; version?: string; label?: string } = {}) =>
    request<ToolRuntimeSnapshot>(`/v3/aihub/runtime/tools/${encodeURIComponent(toolId)}/resolve`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  failures: (params: Record<string, unknown> = {}) =>
    request<Page<ToolFailureRecord>>(`/v3/aihub/tool-failures?${toQuery(params)}`),
};

export const proposalApi = {
  list: (params: Record<string, unknown>) =>
    request<Page<Proposal>>(`/v3/admin/ai/skill-proposals/list?${toQuery(params)}`),
  detail: (id: string) =>
    request<Proposal>(`/v3/admin/ai/skill-proposals/${encodeURIComponent(id)}`),
  validate: (id: string) =>
    request<unknown>(`/v3/admin/ai/skill-proposals/${encodeURIComponent(id)}/validate`, { method: 'POST' }),
  approve: (id: string, options: Record<string, unknown>) =>
    request<unknown>(`/v3/admin/ai/skill-proposals/${encodeURIComponent(id)}/approve`, {
      method: 'POST',
      body: JSON.stringify(options),
    }),
  reject: (id: string, reason: string) =>
    request<unknown>(`/v3/admin/ai/skill-proposals/${encodeURIComponent(id)}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
};

export const iamApi = {
  listUsers: () => request<LocalUser[]>('/v3/admin/iam/local-users/list'),
  saveUser: (u: LocalUser & { password?: string }) =>
    request<LocalUser>('/v3/admin/iam/local-users', { method: 'POST', body: JSON.stringify(u) }),
  deleteUser: (username: string) =>
    request<unknown>(`/v3/admin/iam/local-users/${encodeURIComponent(username)}`, { method: 'DELETE' }),
  whoami: () => request<Record<string, unknown>>('/v3/admin/iam/whoami'),
};

export const namespaceApi = {
  list: () => request<NamespaceInfo[]>('/v3/admin/namespaces'),
  save: (data: Record<string, unknown>) =>
    request<unknown>('/v3/admin/namespaces', { method: 'POST', body: JSON.stringify(data) }),
  members: (namespaceId: string) =>
    request<NamespaceMember[]>(`/v3/admin/namespaces/${namespaceId}/members`),
  saveMember: (namespaceId: string, data: Record<string, unknown>) =>
    request<unknown>(`/v3/admin/namespaces/${namespaceId}/members`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  deleteMember: (namespaceId: string, subjectId: string) =>
    request<unknown>(`/v3/admin/namespaces/${namespaceId}/members/${encodeURIComponent(subjectId)}`, {
      method: 'DELETE',
    }),
};

export const socialApi = {
  stats: (skillName: string) =>
    request<SkillSocialStats>(`/v3/admin/ai/skills/social?${toQuery({ skillName })}`),
  star: (skillName: string, starred: boolean) =>
    request<SkillSocialStats>('/v3/admin/ai/skills/social/star', {
      method: 'POST',
      body: JSON.stringify({ skillName, starred }),
    }),
  rating: (skillName: string, rating: number, comment = '') =>
    request<SkillSocialStats>('/v3/admin/ai/skills/social/rating', {
      method: 'POST',
      body: JSON.stringify({ skillName, rating, comment }),
    }),
  subscribe: (skillName: string, subscribed: boolean) =>
    request<unknown>('/v3/admin/ai/skills/social/subscribe', {
      method: 'POST',
      body: JSON.stringify({ skillName, subscribed }),
    }),
};

export const auditApi = {
  list: (params: Record<string, unknown>) =>
    request<Page<AuditLog>>(`/v3/admin/audit/logs?${toQuery(params)}`),
};

export const tokenApi = {
  list: (subjectId = '') =>
    request<TokenInfo[]>(`/v3/admin/iam/tokens?${toQuery({ subjectId })}`),
  create: (data: Record<string, unknown>) =>
    request<TokenInfo>('/v3/admin/iam/tokens', { method: 'POST', body: JSON.stringify(data) }),
  remove: (keyId: string) =>
    request<unknown>(`/v3/admin/iam/tokens/${keyId}`, { method: 'DELETE' }),
};

export const metricsApi = {
  get: () => request<MetricsSnapshot>('/v3/admin/metrics'),
};

export const notificationApi = {
  list: (params: Record<string, unknown> = {}) =>
    request<Notification[]>(`/v3/admin/notifications?${toQuery(params)}`),
  markRead: (id: string) =>
    request<unknown>(`/v3/admin/notifications/${encodeURIComponent(id)}/read`, { method: 'POST' }),
  streamUrl: () => '/v3/admin/notifications/stream',
};

// ─── AIHub Resource Sharing API ─────────────────────────────────────
// Implements the v3.5 sharing model: each share is a ResourceGrant
// record stored in aisphere-auth's IAM. The visible access mode
// (private / shared / public) is derived from the grants list.

function resourcePath(resourceType: AihubResourceType, resourceId: string): string {
  switch (resourceType) {
    case 'skill':
      return `/v3/aihub/skill/${encodeURIComponent(resourceId)}/shares`;
    case 'skillset':
      return `/v3/aihub/skillset/${encodeURIComponent(resourceId)}/shares`;
    case 'agent':
      return `/v3/aihub/agents/${encodeURIComponent(resourceId)}/shares`;
    case 'tool':
      return `/v3/aihub/tools/${encodeURIComponent(resourceId)}/shares`;
    case 'workflow':
      return `/v3/aihub/workflows/${encodeURIComponent(resourceId)}/shares`;
    default:
      // Reserved types — uses skill endpoint shape; backend may return 501.
      return `/v3/aihub/skill/${encodeURIComponent(resourceId)}/shares`;
  }
}

export const sharesApi = {
  list: (resourceType: AihubResourceType, resourceId: string, params: Record<string, unknown> = {}) =>
    request<ShareListResponse>(`${resourcePath(resourceType, resourceId)}?${toQuery(params)}`),

  create: (resourceType: AihubResourceType, resourceId: string, body: CreateShareRequest) =>
    request<ResourceGrant>(resourcePath(resourceType, resourceId), {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  remove: (resourceType: AihubResourceType, resourceId: string, grantId: string) =>
    request<{ deleted: boolean; id: string }>(
      `${resourcePath(resourceType, resourceId)}/${encodeURIComponent(grantId)}`,
      { method: 'DELETE' },
    ),

  // Convenience wrappers for the most common cases (kept for clarity
  // at call sites — they just delegate to the generic functions above).
  listSkillShares: (skillName: string) => sharesApi.list('skill', skillName),
  createSkillShare: (skillName: string, body: CreateShareRequest) => sharesApi.create('skill', skillName, body),
  deleteSkillShare: (skillName: string, grantId: string) => sharesApi.remove('skill', skillName, grantId),

  listSkillSetShares: (skillSetName: string) => sharesApi.list('skillset', skillSetName),
  createSkillSetShare: (skillSetName: string, body: CreateShareRequest) => sharesApi.create('skillset', skillSetName, body),
  deleteSkillSetShare: (skillSetName: string, grantId: string) => sharesApi.remove('skillset', skillSetName, grantId),
};


export const sandboxProfileApi = {
  list: () => request<SandboxProfile[]>('/v3/aihub/sandbox-profiles'),
  get: (id: string) => request<SandboxProfile>(`/v3/aihub/sandbox-profiles/${encodeURIComponent(id)}`),
  save: (profile: SandboxProfile) => request<SandboxProfile>('/v3/aihub/sandbox-profiles', { method: 'POST', body: JSON.stringify(profile) }),
  update: (id: string, profile: SandboxProfile) => request<SandboxProfile>(`/v3/aihub/sandbox-profiles/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(profile) }),
  remove: (id: string) => request<string>(`/v3/aihub/sandbox-profiles/${encodeURIComponent(id)}`, { method: 'DELETE' }),
};


export const modelProfileApi = {
  list: () => request<ModelProfile[]>('/v3/aihub/model-profiles'),
  get: (id: string) => request<ModelProfile>(`/v3/aihub/model-profiles/${encodeURIComponent(id)}`),
  save: (profile: ModelProfile) => request<ModelProfile>('/v3/aihub/model-profiles', { method: 'POST', body: JSON.stringify(profile) }),
  update: (id: string, profile: ModelProfile) => request<ModelProfile>(`/v3/aihub/model-profiles/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(profile) }),
  remove: (id: string) => request<string>(`/v3/aihub/model-profiles/${encodeURIComponent(id)}`, { method: 'DELETE' }),
};
