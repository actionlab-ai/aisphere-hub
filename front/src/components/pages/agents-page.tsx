'use client';

import { useEffect, useMemo, useState } from 'react';
import { Bot, PlayCircle, RefreshCw, Save, Share2, Trash2, Plus, AlertTriangle, FileCode2 } from 'lucide-react';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Textarea } from '@/components/ui/textarea';
import { ConfirmDialog, EmptyState, ListSkeleton, StatCard } from '@/components/shared';
import { ResourceSharePanel } from '@/components/aihub/resource-share-panel';
import { useAgentDelete, useAgentDetail, useAgentResolve, useAgentSave, useAgentUpdate, useAgents } from '@/hooks/use-agents';
import { fmtTime } from '@/lib/utils';
import type { Agent, AgentDefinition, AgentListItem, AgentUpsertRequest } from '@/lib/api/types';

const DEFAULT_DEFINITION: AgentDefinition = {
  entryPoint: 'root_agent.yaml',
  files: {
    'root_agent.yaml': 'name: demo-agent\ndescription: Managed by AIHub\n',
  },
  services: [
    { kind: 'mcp', name: 'filesystem', provider: 'external', required: false, reload: 'live', runtime: { transport: 'stdio' } },
  ],
  skills: [],
  skillSets: [],
  tools: [],
};

function pretty(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

function parseDefinition(raw: string): AgentDefinition {
  const parsed = JSON.parse(raw) as AgentDefinition;
  if (!parsed.entryPoint || !parsed.files || typeof parsed.files !== 'object') {
    throw new Error('definition must include entryPoint and files');
  }
  return parsed;
}

function latestVersion(agent?: Agent) {
  if (!agent) return undefined;
  return agent.latestVersion && agent.versions ? agent.versions[agent.latestVersion] : undefined;
}

function AgentCreateDialog({ onCreated }: { onCreated: (id: string) => void }) {
  const save = useAgentSave();
  const [open, setOpen] = useState(false);
  const [id, setId] = useState('demo-agent');
  const [displayName, setDisplayName] = useState('Demo Agent');
  const [description, setDescription] = useState('AgentKit agent definition stored in AIHub');
  const [definitionText, setDefinitionText] = useState(pretty(DEFAULT_DEFINITION));

  const submit = async () => {
    try {
      const body: AgentUpsertRequest = {
        id: id.trim(),
        displayName: displayName.trim(),
        description: description.trim(),
        status: 'enable',
        definition: parseDefinition(definitionText),
      };
      const out = await save.mutateAsync(body);
      const agentId = out.agent?.id || body.id || '';
      toast.success(`Agent ${agentId} created`);
      setOpen(false);
      onCreated(agentId);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Create agent failed');
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" className="bg-gradient-to-r from-violet-600 to-fuchsia-500">
          <Plus className="h-3.5 w-3.5 mr-1" /> New Agent
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-3xl">
        <DialogHeader><DialogTitle>Create Agent</DialogTitle></DialogHeader>
        <div className="grid md:grid-cols-2 gap-3">
          <div className="space-y-1.5"><Label className="text-xs">Agent ID</Label><Input value={id} onChange={(e) => setId(e.target.value)} /></div>
          <div className="space-y-1.5"><Label className="text-xs">Display Name</Label><Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} /></div>
          <div className="space-y-1.5 md:col-span-2"><Label className="text-xs">Description</Label><Input value={description} onChange={(e) => setDescription(e.target.value)} /></div>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Definition JSON</Label>
          <Textarea className="font-mono text-xs min-h-[320px]" value={definitionText} onChange={(e) => setDefinitionText(e.target.value)} />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={submit} disabled={save.isPending}><Save className="h-3.5 w-3.5 mr-1" /> Create</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function AgentListCard({ item, active, onClick }: { item: AgentListItem; active: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick} className={`w-full text-left rounded-lg border p-3 transition-colors ${active ? 'border-violet-500 bg-violet-500/5' : 'hover:bg-accent/50'}`}>
      <div className="flex items-start gap-2">
        <Bot className="h-4 w-4 mt-0.5 text-violet-500" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm truncate">{item.displayName || item.id}</span>
            <Badge variant="secondary" className="text-[10px]">{item.latestVersion || '-'}</Badge>
          </div>
          <p className="text-xs text-muted-foreground truncate mt-0.5">{item.id}</p>
          {item.description && <p className="text-xs text-muted-foreground line-clamp-2 mt-1">{item.description}</p>}
        </div>
      </div>
    </button>
  );
}

export function AgentsPage() {
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [definitionText, setDefinitionText] = useState(pretty(DEFAULT_DEFINITION));
  const [version, setVersion] = useState('');
  const [commitMsg, setCommitMsg] = useState('update agent definition');

  const { data: agents = [], isLoading, error, refetch } = useAgents({ q: search || undefined, pageSize: 80 });
  const filtered = useMemo(() => agents.filter((a) => !search || `${a.id} ${a.displayName || ''} ${a.description || ''}`.toLowerCase().includes(search.toLowerCase())), [agents, search]);
  const { data: detail, refetch: refetchDetail } = useAgentDetail(selectedId);
  const update = useAgentUpdate();
  const remove = useAgentDelete();
  const resolve = useAgentResolve();
  const agent = detail?.agent;
  const v = latestVersion(agent);

  useEffect(() => {
    if (!selectedId && filtered.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- selects the first result after async data arrives.
      setSelectedId(filtered[0].id);
    }
  }, [filtered, selectedId]);

  useEffect(() => {
    if (v?.definition) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- resets the editor when the selected version changes.
      setDefinitionText(pretty(v.definition));
    }
    if (agent?.latestVersion) setVersion('');
  }, [agent?.latestVersion, v?.definition]);

  const saveUpdate = async () => {
    if (!agent) return;
    try {
      const nextVersion = version.trim();
      await update.mutateAsync({
        agentId: agent.id,
        data: {
          id: agent.id,
          displayName: agent.displayName,
          description: agent.description,
          status: agent.status || 'enable',
          scope: agent.scope,
          labels: agent.labels,
          version: nextVersion || undefined,
          commitMsg,
          definition: parseDefinition(definitionText),
        },
      });
      toast.success(`Agent ${agent.id} updated`);
      refetch();
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Update agent failed');
    }
  };

  const runResolve = async () => {
    if (!agent) return;
    try {
      const out = await resolve.mutateAsync({ agentId: agent.id, runtimeId: 'agentkit-console', sessionId: `console-${Date.now()}` });
      toast.success(`Snapshot ${out.snapshotId} resolved, ${out.services?.length || 0} services`);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Resolve runtime snapshot failed');
    }
  };

  const confirmDelete = async () => {
    if (!deleteId) return;
    try {
      await remove.mutateAsync(deleteId);
      toast.success(`Agent ${deleteId} deleted`);
      setSelectedId(null);
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Delete agent failed');
    }
    setDeleteId(null);
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard icon={<Bot className="h-4 w-4" />} label="Agents" value={agents.length} />
        <StatCard icon={<FileCode2 className="h-4 w-4" />} label="Versioned" value={agents.filter(a => a.latestVersion).length} />
        <StatCard icon={<PlayCircle className="h-4 w-4" />} label="Runnable" value={agents.filter(a => a.status !== 'disable').length} />
        <StatCard icon={<Share2 className="h-4 w-4" />} label="AIHub Objects" value="agent" />
      </div>

      <div className="flex items-center gap-2">
        <Input className="max-w-md" placeholder="Search agents..." value={search} onChange={(e) => setSearch(e.target.value)} />
        <Button variant="outline" size="sm" onClick={() => refetch()}><RefreshCw className="h-3.5 w-3.5 mr-1" /> Refresh</Button>
        <AgentCreateDialog onCreated={(id) => { setSelectedId(id); refetch(); }} />
      </div>

      {error && <div className="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm"><AlertTriangle className="h-4 w-4" />{error.message}</div>}

      <div className="grid lg:grid-cols-[360px_1fr] gap-4">
        <Card>
          <CardHeader className="py-3"><CardTitle className="text-sm">Agent Registry</CardTitle></CardHeader>
          <CardContent className="p-3 pt-0">
            {isLoading ? <ListSkeleton count={4} /> : filtered.length === 0 ? (
              <EmptyState icon={<Bot className="h-10 w-10" />} title="No agents" description="Create an AgentKit definition and store it in AIHub." />
            ) : (
              <ScrollArea className="h-[620px] pr-2">
                <div className="space-y-2">
                  {filtered.map((item) => <AgentListCard key={item.id} item={item} active={selectedId === item.id} onClick={() => setSelectedId(item.id)} />)}
                </div>
              </ScrollArea>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="py-3">
            <div className="flex items-center justify-between gap-2">
              <div>
                <CardTitle className="text-sm">{agent?.displayName || agent?.id || 'Agent Detail'}</CardTitle>
                {detail?.object && <p className="text-xs text-muted-foreground mt-1">{detail.object}</p>}
              </div>
              {agent && <div className="flex gap-2"><Button size="sm" variant="outline" onClick={runResolve}><PlayCircle className="h-3.5 w-3.5 mr-1" /> Resolve</Button><Button size="sm" variant="destructive" onClick={() => setDeleteId(agent.id)}><Trash2 className="h-3.5 w-3.5 mr-1" /> Delete</Button></div>}
            </div>
          </CardHeader>
          <CardContent>
            {!agent ? (
              <EmptyState icon={<Bot className="h-10 w-10" />} title="Select an agent" description="Choose an agent from the registry to view and edit its versioned definition." />
            ) : (
              <Tabs defaultValue="definition" className="space-y-4">
                <TabsList><TabsTrigger value="definition">Definition</TabsTrigger><TabsTrigger value="runtime">Runtime</TabsTrigger><TabsTrigger value="shares">Shares</TabsTrigger></TabsList>
                <TabsContent value="definition" className="space-y-3">
                  <div className="grid md:grid-cols-4 gap-3 text-xs">
                    <div><Label className="text-muted-foreground">Latest</Label><div className="font-medium mt-1">{agent.latestVersion || '-'}</div></div>
                    <div><Label className="text-muted-foreground">Revision</Label><div className="font-mono mt-1">{v?.revision || '-'}</div></div>
                    <div><Label className="text-muted-foreground">Updated</Label><div className="mt-1">{agent.updateTime ? fmtTime(agent.updateTime) : '-'}</div></div>
                    <div><Label className="text-muted-foreground">Services</Label><div className="mt-1">{v?.definition?.services?.length || 0} canonical</div></div>
                  </div>
                  <Separator />
                  <div className="grid md:grid-cols-2 gap-3">
                    <div className="space-y-1.5"><Label className="text-xs">New Version</Label><Input placeholder="blank = next patch" value={version} onChange={(e) => setVersion(e.target.value)} /></div>
                    <div className="space-y-1.5"><Label className="text-xs">Commit Message</Label><Input value={commitMsg} onChange={(e) => setCommitMsg(e.target.value)} /></div>
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs">Definition JSON</Label>
                    <Textarea className="font-mono text-xs min-h-[420px]" value={definitionText} onChange={(e) => setDefinitionText(e.target.value)} />
                  </div>
                  <div className="flex justify-end"><Button onClick={saveUpdate} disabled={update.isPending}><Save className="h-3.5 w-3.5 mr-1" /> Save new version</Button></div>
                </TabsContent>
                <TabsContent value="runtime" className="space-y-3">
                  <p className="text-xs text-muted-foreground">AgentKit should call the runtime resolve endpoint, mount only this immutable service snapshot, and never read Hub management APIs directly.</p>
                  <Textarea readOnly className="font-mono text-xs min-h-[420px]" value={resolve.data ? pretty(resolve.data) : 'Click Resolve to preview the AgentKit snapshot.'} />
                </TabsContent>
                <TabsContent value="shares">
                  <ResourceSharePanel resourceType="agent" resourceId={agent.id} object={`aihub:agent:${agent.id}`} owner={agent.ownerSubject} />
                </TabsContent>
              </Tabs>
            )}
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={Boolean(deleteId)}
        onOpenChange={(open) => { if (!open) setDeleteId(null); }}
        title="Delete Agent"
        description={`Delete agent ${deleteId || ''}? This removes all stored versions from Hub.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={confirmDelete}
      />
    </div>
  );
}
