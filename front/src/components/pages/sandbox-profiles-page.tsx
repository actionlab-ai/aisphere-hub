'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { useSandboxProfiles, useSaveSandboxProfile, useDeleteSandboxProfile } from '@/hooks/use-sandbox-profiles';

const defaultProfile = {
  id: 'default-python-offline',
  version: '1.0.0',
  status: 'enable',
  displayName: '默认 Python 离线沙箱',
  driver: 'agent-sandbox',
  templateRef: 'aisphere-agent-session',
  network: { mode: 'offline' },
  workspace: { size: '10Gi', mountPath: '/workspace', reuse: 'session' },
  resources: { cpu: '500m', memory: '1Gi' },
  capabilities: { browser: false, shell: false, mcp: true, online: false, customTools: false },
  builtinTools: ['workspace.list','workspace.read','workspace.write','workspace.search_text'],
};

export function SandboxProfilesPage() {
  const { data = [] } = useSandboxProfiles();
  const save = useSaveSandboxProfile();
  const del = useDeleteSandboxProfile();
  const [json, setJson] = useState(JSON.stringify(defaultProfile, null, 2));
  const onSave = () => save.mutate(JSON.parse(json));
  return <div className="space-y-4">
    <div><h1 className="text-2xl font-semibold">Sandbox Profiles</h1><p className="text-muted-foreground">Hub 管产品层沙箱规格；Adapter 负责转换到 agent-sandbox CRD。</p></div>
    <Card><CardHeader><CardTitle>新建 / 更新 Profile JSON</CardTitle></CardHeader><CardContent className="space-y-3"><Textarea className="font-mono min-h-[320px]" value={json} onChange={e=>setJson(e.target.value)} /><Button onClick={onSave}>保存 Profile</Button></CardContent></Card>
    <div className="grid gap-3">{data.map(p => <Card key={p.id}><CardHeader><CardTitle>{p.displayName || p.id}</CardTitle></CardHeader><CardContent className="space-y-2 text-sm"><div>driver: {p.driver} / template: {p.templateRef}</div><div>network: {p.network?.mode} / workspace: {p.workspace?.size}</div><pre className="rounded bg-muted p-3 overflow-auto text-xs">{JSON.stringify(p, null, 2)}</pre><Button variant="destructive" size="sm" onClick={()=>del.mutate(p.id)}>删除</Button></CardContent></Card>)}</div>
  </div>;
}
