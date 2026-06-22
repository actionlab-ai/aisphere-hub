'use client';

import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, Hammer, Plus, RefreshCw, Save, Search, Share2, Trash2, Zap } from 'lucide-react';
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
import { ResourceSharePanel } from '@/components/aihub/resource-share-panel';
import { ConfirmDialog, EmptyState, ListSkeleton, StatCard } from '@/components/shared';
import { useToolDelete, useToolDetail, useToolFailures, useToolResolve, useToolSave, useTools, useToolUpdate } from '@/hooks/use-tools';
import { fmtTime } from '@/lib/utils';
import type { Tool, ToolDefinition, ToolListItem, ToolUpsertRequest } from '@/lib/api/types';

const DEFAULT_TOOL: ToolDefinition = {
  runtime: {
    type: 'mcp',
    server: 'github',
    name: 'create_issue',
    description: 'Create a GitHub issue through the configured MCP server.',
  },
  execution: {
    placement: 'remote',
    runner: 'mcp',
    network: 'restricted',
    filesystem: 'none',
    secretRefs: ['secret://project/github-token'],
    capabilities: ['mcp.call', 'network.restricted'],
  },
  inputSchema: {
    type: 'object',
    properties: {
      owner: { type: 'string' },
      repo: { type: 'string' },
      title: { type: 'string' },
      body: { type: 'string' },
    },
    required: ['owner', 'repo', 'title'],
  },
  timeoutMillis: 30000,
  retry: { maxAttempts: 2, backoffMillis: 1000 },
};

function pretty(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

function parseDefinition(raw: string): ToolDefinition {
  const parsed = JSON.parse(raw) as ToolDefinition;
  if (!parsed.runtime || !parsed.runtime.type) {
    throw new Error('definition must include runtime.type');
  }
  return parsed;
}

function latestVersion(tool?: Tool) {
  if (!tool) return undefined;
  return tool.latestVersion && tool.versions ? tool.versions[tool.latestVersion] : undefined;
}

function ToolCreateDialog({ onCreated }: { onCreated: (id: string) => void }) {
  const save = useToolSave();
  const [open, setOpen] = useState(false);
  const [id, setId] = useState('github.issue.create');
  const [displayName, setDisplayName] = useState('GitHub Create Issue');
  const [description, setDescription] = useState('MCP tool managed by AIHub');
  const [definitionText, setDefinitionText] = useState(pretty(DEFAULT_TOOL));

  const submit = async () => {
    try {
      const body: ToolUpsertRequest = {
        id: id.trim(),
        displayName: displayName.trim(),
        description: description.trim(),
        status: 'enable',
        definition: parseDefinition(definitionText),
      };
      const out = await save.mutateAsync(body);
      const toolId = out.tool?.id || body.id || '';
      toast.success(`Tool ${toolId} created`);
      setOpen(false);
      onCreated(toolId);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Create tool failed');
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" className="bg-gradient-to-r from-violet-600 to-fuchsia-500">
          <Plus className="h-3.5 w-3.5 mr-1" /> New Tool
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-3xl">
        <DialogHeader><DialogTitle>Create Tool</DialogTitle></DialogHeader>
        <div className="grid md:grid-cols-2 gap-3">
          <div className="space-y-1.5"><Label className="text-xs">Tool ID</Label><Input value={id} onChange={(e) => setId(e.target.value)} /></div>
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

function ToolListCard({ item, active, onClick }: { item: ToolListItem; active: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick} className={`w-full text-left rounded-lg border p-3 transition-colors ${active ? 'border-violet-500 bg-violet-500/5' : 'hover:bg-accent/50'}`}>
      <div className="flex items-start gap-2">
        <Hammer className="h-4 w-4 mt-0.5 text-violet-500" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm truncate">{item.displayName || item.id}</span>
            <Badge variant="secondary" className="text-[10px]">{item.runtimeType || '-'}</Badge>
          </div>
          <p className="text-xs text-muted-foreground truncate mt-0.5">{item.id}</p>
          {item.description && <p className="text-xs text-muted-foreground line-clamp-2 mt-1">{item.description}</p>}
        </div>
      </div>
    </button>
  );
}

export function ToolsPage() {
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [definitionText, setDefinitionText] = useState(pretty(DEFAULT_TOOL));
  const [version, setVersion] = useState('');
  const [commitMsg, setCommitMsg] = useState('update tool definition');

  const { data: tools = [], isLoading, error, refetch } = useTools({ q: search || undefined, pageSize: 80 });
  const filtered = useMemo(() => tools.filter((t) => !search || `${t.id} ${t.displayName || ''} ${t.description || ''}`.toLowerCase().includes(search.toLowerCase())), [tools, search]);
  const { data: detail, refetch: refetchDetail } = useToolDetail(selectedId);
  const { data: failures = [], refetch: refetchFailures } = useToolFailures(selectedId ? { toolId: selectedId, limit: 50 } : { limit: 50 });
  const update = useToolUpdate();
  const remove = useToolDelete();
  const resolve = useToolResolve();
  const tool = detail?.tool;
  const v = latestVersion(tool);

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
    if (tool?.latestVersion) setVersion('');
  }, [tool?.latestVersion, v?.definition]);

  const saveUpdate = async () => {
    if (!tool) return;
    try {
      await update.mutateAsync({
        toolId: tool.id,
        data: {
          id: tool.id,
          displayName: tool.displayName,
          description: tool.description,
          status: tool.status || 'enable',
          scope: tool.scope,
          labels: tool.labels,
          version: version.trim() || undefined,
          commitMsg,
          definition: parseDefinition(definitionText),
        },
      });
      toast.success(`Tool ${tool.id} updated`);
      refetch();
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Update tool failed');
    }
  };

  const runResolve = async () => {
    if (!tool) return;
    try {
      const out = await resolve.mutateAsync({ toolId: tool.id, runtimeId: 'agentkit-console', sessionId: `console-${Date.now()}` });
      toast.success(`Snapshot ${out.snapshotId} resolved`);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Resolve tool snapshot failed');
    }
  };

  const confirmDelete = async () => {
    if (!deleteId) return;
    try {
      await remove.mutateAsync(deleteId);
      toast.success(`Tool ${deleteId} deleted`);
      setSelectedId(null);
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Delete tool failed');
    }
    setDeleteId(null);
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard icon={<Hammer className="h-4 w-4" />} label="Tools" value={tools.length} />
        <StatCard icon={<Zap className="h-4 w-4" />} label="Runnable" value={tools.filter(t => t.status !== 'disable').length} />
        <StatCard icon={<AlertTriangle className="h-4 w-4" />} label="Failures" value={failures.length} />
        <StatCard icon={<Share2 className="h-4 w-4" />} label="AIHub Objects" value="tool" />
      </div>

      <div className="flex items-center gap-2">
        <div className="relative max-w-md flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input className="pl-8" placeholder="Search tools..." value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        <Button variant="outline" size="sm" onClick={() => { refetch(); refetchFailures(); }}><RefreshCw className="h-3.5 w-3.5 mr-1" /> Refresh</Button>
        <ToolCreateDialog onCreated={(id) => { setSelectedId(id); refetch(); }} />
      </div>

      {error && <div className="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm"><AlertTriangle className="h-4 w-4" />{error.message}</div>}

      <div className="grid lg:grid-cols-[360px_1fr] gap-4">
        <Card>
          <CardHeader className="py-3"><CardTitle className="text-sm">Tool Registry</CardTitle></CardHeader>
          <CardContent className="p-3 pt-0">
            {isLoading ? <ListSkeleton count={4} /> : filtered.length === 0 ? (
              <EmptyState icon={<Hammer className="h-10 w-10" />} title="No tools" description="Create an MCP/OpenAPI/HTTP tool and manage it through AIHub." />
            ) : (
              <ScrollArea className="h-[620px] pr-2">
                <div className="space-y-2">
                  {filtered.map((item) => <ToolListCard key={item.id} item={item} active={selectedId === item.id} onClick={() => setSelectedId(item.id)} />)}
                </div>
              </ScrollArea>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="py-3">
            <div className="flex items-center justify-between gap-2">
              <div>
                <CardTitle className="text-sm">{tool?.displayName || tool?.id || 'Tool Detail'}</CardTitle>
                {detail?.object && <p className="text-xs text-muted-foreground mt-1">{detail.object}</p>}
              </div>
              {tool && <div className="flex gap-2"><Button size="sm" variant="outline" onClick={runResolve}><Zap className="h-3.5 w-3.5 mr-1" /> Resolve</Button><Button size="sm" variant="destructive" onClick={() => setDeleteId(tool.id)}><Trash2 className="h-3.5 w-3.5 mr-1" /> Delete</Button></div>}
            </div>
          </CardHeader>
          <CardContent>
            {!tool ? (
              <EmptyState icon={<Hammer className="h-10 w-10" />} title="Select a tool" description="Choose a Hub-managed tool to edit its versioned runtime definition." />
            ) : (
              <Tabs defaultValue="definition" className="space-y-4">
                <TabsList><TabsTrigger value="definition">Definition</TabsTrigger><TabsTrigger value="runtime">Runtime</TabsTrigger><TabsTrigger value="failures">Failures</TabsTrigger><TabsTrigger value="shares">Shares</TabsTrigger></TabsList>
                <TabsContent value="definition" className="space-y-3">
                  <div className="grid md:grid-cols-4 gap-3 text-xs">
                    <div><Label className="text-muted-foreground">Latest</Label><div className="font-medium mt-1">{tool.latestVersion || '-'}</div></div>
                    <div><Label className="text-muted-foreground">Revision</Label><div className="font-mono mt-1">{v?.revision || '-'}</div></div>
                    <div><Label className="text-muted-foreground">Runtime</Label><div className="mt-1">{v?.definition?.runtime?.type || '-'}</div></div>
                    <div><Label className="text-muted-foreground">Updated</Label><div className="mt-1">{tool.updateTime ? fmtTime(tool.updateTime) : '-'}</div></div>
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
                  <p className="text-xs text-muted-foreground">Runtime should call the tool resolve endpoint and execute only this permission-checked immutable manifest.</p>
                  <Textarea readOnly className="font-mono text-xs min-h-[420px]" value={resolve.data ? pretty(resolve.data) : 'Click Resolve to preview the Tool runtime snapshot.'} />
                </TabsContent>
                <TabsContent value="failures" className="space-y-2">
                  {failures.length === 0 ? <EmptyState icon={<AlertTriangle className="h-10 w-10" />} title="No failures" description="Runtime failures reported by AgentKit will appear here." /> : (
                    <div className="space-y-2">
                      {failures.map((f) => (
                        <div key={f.id} className="rounded-lg border p-3 text-xs">
                          <div className="flex items-center gap-2"><Badge variant={f.retryable ? 'secondary' : 'destructive'}>{f.errorCode || 'error'}</Badge><span className="font-mono text-muted-foreground">{f.id}</span><span className="ml-auto text-muted-foreground">{f.createTime ? fmtTime(f.createTime) : '-'}</span></div>
                          <p className="mt-2 text-sm">{f.errorMessage}</p>
                          <p className="mt-1 text-muted-foreground">agent={f.agentId || '-'} session={f.sessionId || '-'} trace={f.traceId || '-'}</p>
                        </div>
                      ))}
                    </div>
                  )}
                </TabsContent>
                <TabsContent value="shares">
                  <ResourceSharePanel resourceType="tool" resourceId={tool.id} object={`aihub:tool:${tool.id}`} owner={tool.ownerSubject} />
                </TabsContent>
              </Tabs>
            )}
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={Boolean(deleteId)}
        onOpenChange={(open) => { if (!open) setDeleteId(null); }}
        title="Delete Tool"
        description={`Delete tool ${deleteId || ''}? This removes all stored versions from Hub.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={confirmDelete}
      />
    </div>
  );
}
