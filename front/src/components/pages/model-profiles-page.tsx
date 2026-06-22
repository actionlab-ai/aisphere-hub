'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { useModelProfiles, useSaveModelProfile, useDeleteModelProfile } from '@/hooks/use-model-profiles';

const defaultProfile = {
  id: 'deepseek-v4-agent',
  version: '1.0.0',
  status: 'enable',
  displayName: 'DeepSeek V4 Agent',
  provider: 'vllm',
  apiFormat: 'openai-chat-completions',
  endpoint: 'vllm-ascend',
  model: 'deepseek-v4-agent',
  upstreamModel: 'deepseek-v4',
  upstreamPath: '/v1/chat/completions',
  secretRef: 'env://VLLM_API_KEY',
  limits: { maxInputTokens: 131072, maxOutputTokens: 8192 },
  reasoning: { enabled: true, effort: 'high' },
};

export function ModelProfilesPage() {
  const { data = [] } = useModelProfiles();
  const save = useSaveModelProfile();
  const del = useDeleteModelProfile();
  const [json, setJson] = useState(JSON.stringify(defaultProfile, null, 2));
  const onSave = () => save.mutate(JSON.parse(json));
  return <div className="space-y-4">
    <div><h1 className="text-2xl font-semibold">Model Profiles</h1><p className="text-muted-foreground">Hub 管模型产品配置；aisphere-gateway 解析 profile 并代理到上游模型服务，真实 key 不进 Hub。</p></div>
    <Card><CardHeader><CardTitle>新建 / 更新 ModelProfile JSON</CardTitle></CardHeader><CardContent className="space-y-3"><Textarea className="font-mono min-h-[320px]" value={json} onChange={e=>setJson(e.target.value)} /><Button onClick={onSave}>保存 ModelProfile</Button></CardContent></Card>
    <div className="grid gap-3">{data.map(p => <Card key={p.id}><CardHeader><CardTitle>{p.displayName || p.id}</CardTitle></CardHeader><CardContent className="space-y-2 text-sm"><div>provider: {p.provider} / endpoint: {p.endpoint}</div><div>model: {p.model} → {p.upstreamModel}</div><pre className="rounded bg-muted p-3 overflow-auto text-xs">{JSON.stringify(p, null, 2)}</pre><Button variant="destructive" size="sm" onClick={()=>del.mutate(p.id)}>删除</Button></CardContent></Card>)}</div>
  </div>;
}
