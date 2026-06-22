'use client';

import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { Boxes, ExternalLink, Play, RefreshCw, RotateCcw, Search, Terminal, Trash2, Wifi, WifiOff } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Textarea } from '@/components/ui/textarea';
import { EmptyState, ListSkeleton, StatCard } from '@/components/shared';
import { useSandboxDelete, useSandboxEnsure, useSandboxRestart, useSandboxes, useSandboxToolCall, useSandboxTools } from '@/hooks/use-sandboxes';
import { sandboxApi } from '@/lib/api';
import type { SandboxEnsureRequest, SandboxStatus, SandboxToolSchema } from '@/lib/api/types';

function pretty(v: unknown): string {
  return JSON.stringify(v, null, 2);
}

function phaseVariant(phase?: string): 'default' | 'secondary' | 'destructive' {
  const p = (phase || '').toLowerCase();
  if (p === 'running') return 'default';
  if (p === 'failed' || p === 'unknown') return 'destructive';
  return 'secondary';
}

function SandboxCard({ item, active, onClick }: { item: SandboxStatus; active: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick} className={`w-full text-left rounded-lg border p-3 transition-colors ${active ? 'border-violet-500 bg-violet-500/5' : 'hover:bg-accent/50'}`}>
      <div className="flex items-start gap-2">
        <Boxes className="h-4 w-4 mt-0.5 text-violet-500" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm truncate">{item.sandboxId}</span>
            <Badge variant={phaseVariant(item.phase)} className="text-[10px]">{item.phase}</Badge>
          </div>
          <p className="text-xs text-muted-foreground truncate mt-0.5">session: {item.sessionId || '-'}</p>
          <p className="text-xs text-muted-foreground truncate">agent: {item.agentId || '-'}</p>
        </div>
      </div>
    </button>
  );
}

function ToolRow({ tool, selected, onClick }: { tool: SandboxToolSchema; selected: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick} className={`w-full rounded-md border px-3 py-2 text-left ${selected ? 'border-violet-500 bg-violet-500/5' : 'hover:bg-accent/50'}`}>
      <div className="font-mono text-xs font-medium">{tool.name}</div>
      {tool.description && <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">{tool.description}</div>}
    </button>
  );
}

const DEFAULT_CREATE: SandboxEnsureRequest = {
  runtimeId: 'agentkit-console',
  sessionId: 'console-session',
  agentId: 'debug-agent',
  workspaceSize: '10Gi',
  network: { mode: 'offline' },
  limits: { cpu: '500m', memory: '1Gi', idleTtlSeconds: 3600 },
  metadata: { source: 'aihub-console' },
};

export function SandboxesPage() {
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [createText, setCreateText] = useState(pretty(DEFAULT_CREATE));
  const [selectedTool, setSelectedTool] = useState<string>('workspace.list');
  const [toolInput, setToolInput] = useState(pretty({ path: '.', recursive: false }));
  const { data: sandboxes = [], isLoading, refetch, error } = useSandboxes({ pageSize: 100 });
  const ensure = useSandboxEnsure();
  const restart = useSandboxRestart();
  const remove = useSandboxDelete();
  const callTool = useSandboxToolCall();
  const { data: toolList, refetch: refetchTools } = useSandboxTools(selectedId);
  const tools = toolList?.tools || [];

  const filtered = useMemo(() => sandboxes.filter((s) => !search || `${s.sandboxId} ${s.sessionId || ''} ${s.agentId || ''} ${s.phase || ''}`.toLowerCase().includes(search.toLowerCase())), [sandboxes, search]);
  const selected = filtered.find((s) => s.sandboxId === selectedId) || sandboxes.find((s) => s.sandboxId === selectedId) || null;

  useEffect(() => {
    if (!selectedId && filtered.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- selects the first sandbox after async data arrives.
      setSelectedId(filtered[0].sandboxId);
    }
  }, [filtered, selectedId]);

  useEffect(() => {
    const current = tools.find((t) => t.name === selectedTool);
    if (!current && tools.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- keeps the tool selection valid after a sandbox refresh.
      setSelectedTool(tools[0].name);
    }
  }, [tools, selectedTool]);

  const createSandbox = async () => {
    try {
      const body = JSON.parse(createText) as SandboxEnsureRequest;
      const st = await ensure.mutateAsync(body);
      toast.success(`Sandbox ${st.sandboxId} ensured`);
      setSelectedId(st.sandboxId);
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Ensure sandbox failed');
    }
  };

  const restartSelected = async () => {
    if (!selected) return;
    try {
      const st = await restart.mutateAsync(selected.sandboxId);
      toast.success(`Sandbox ${st.sandboxId} restarted`);
      refetch();
      refetchTools();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Restart sandbox failed');
    }
  };

  const deleteSelected = async (deleteWorkspace = false) => {
    if (!selected) return;
    try {
      await remove.mutateAsync({ sandboxId: selected.sandboxId, deleteWorkspace });
      toast.success(`Sandbox ${selected.sandboxId} deleted`);
      setSelectedId(null);
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Delete sandbox failed');
    }
  };

  const runTool = async () => {
    if (!selected || !selectedTool) return;
    try {
      const input = toolInput.trim() ? JSON.parse(toolInput) as Record<string, unknown> : {};
      const out = await callTool.mutateAsync({ sandboxId: selected.sandboxId, body: { tool: selectedTool, input, traceId: `console-${Date.now()}` } });
      toast.success(out.ok ? `Tool ${selectedTool} ok` : `Tool ${selectedTool} failed`);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Tool call failed');
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard icon={<Boxes className="h-4 w-4" />} label="Sandboxes" value={sandboxes.length} />
        <StatCard icon={<Play className="h-4 w-4" />} label="Running" value={sandboxes.filter((s) => s.phase === 'Running').length} />
        <StatCard icon={<WifiOff className="h-4 w-4" />} label="Offline" value={sandboxes.filter((s) => s.networkMode === 'offline').length} />
        <StatCard icon={<Terminal className="h-4 w-4" />} label="Tools" value={tools.length} />
      </div>

      <div className="flex items-center gap-2">
        <div className="relative max-w-md flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input className="pl-8" placeholder="Search sandboxes..." value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}><RefreshCw className="h-3.5 w-3.5 mr-1" /> Refresh</Button>
      </div>

      {error && <div className="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">{error.message}</div>}

      <div className="grid lg:grid-cols-[360px_1fr] gap-4">
        <Card>
          <CardHeader className="py-3"><CardTitle className="text-sm">Sandbox Pods</CardTitle></CardHeader>
          <CardContent className="p-3 pt-0">
            {isLoading ? <ListSkeleton count={4} /> : filtered.length === 0 ? (
              <EmptyState icon={<Boxes className="h-10 w-10" />} title="No sandboxes" description="Ensure a sandbox to create a Kubernetes Pod with /workspace PVC." />
            ) : (
              <ScrollArea className="h-[560px] pr-2">
                <div className="space-y-2">{filtered.map((item) => <SandboxCard key={item.sandboxId} item={item} active={item.sandboxId === selectedId} onClick={() => setSelectedId(item.sandboxId)} />)}</div>
              </ScrollArea>
            )}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader className="py-3"><CardTitle className="text-sm">Ensure Sandbox</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <Textarea className="font-mono text-xs min-h-[180px]" value={createText} onChange={(e) => setCreateText(e.target.value)} />
              <div className="flex justify-end"><Button onClick={createSandbox} disabled={ensure.isPending}><Play className="h-3.5 w-3.5 mr-1" /> Ensure</Button></div>
            </CardContent>
          </Card>

          {selected && (
            <Card>
              <CardHeader className="py-3">
                <div className="flex items-center justify-between gap-2">
                  <CardTitle className="text-sm">{selected.sandboxId}</CardTitle>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={restartSelected}><RotateCcw className="h-3.5 w-3.5 mr-1" /> Restart</Button>
                    <Button variant="outline" size="sm" onClick={() => deleteSelected(false)}><Trash2 className="h-3.5 w-3.5 mr-1" /> Delete Pod</Button>
                    <Button variant="destructive" size="sm" onClick={() => deleteSelected(true)}>Delete PVC</Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid md:grid-cols-2 gap-3 text-xs">
                  <div><Label className="text-muted-foreground">Phase</Label><div><Badge variant={phaseVariant(selected.phase)}>{selected.phase}</Badge></div></div>
                  <div><Label className="text-muted-foreground">Network</Label><div className="flex items-center gap-1">{selected.networkMode === 'online' ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}{selected.networkMode || '-'}</div></div>
                  <div><Label className="text-muted-foreground">Pod</Label><div className="font-mono break-all">{selected.podName || '-'}</div></div>
                  <div><Label className="text-muted-foreground">PVC</Label><div className="font-mono break-all">{selected.workspacePvc || '-'}</div></div>
                  <div><Label className="text-muted-foreground">Image</Label><div className="font-mono break-all">{selected.image || '-'}</div></div>
                  <div><Label className="text-muted-foreground">Logs</Label><div><a className="inline-flex items-center text-violet-600" href={sandboxApi.logsUrl(selected.sandboxId)} target="_blank" rel="noreferrer">Open logs <ExternalLink className="ml-1 h-3 w-3" /></a></div></div>
                </div>
                <Separator />
                <div className="grid lg:grid-cols-[260px_1fr] gap-3">
                  <div className="space-y-2">
                    <div className="flex items-center justify-between"><Label>Sandbox Tools</Label><Button size="sm" variant="ghost" onClick={() => refetchTools()}>Refresh</Button></div>
                    <ScrollArea className="h-[360px] pr-2"><div className="space-y-2">{tools.map((tool) => <ToolRow key={tool.name} tool={tool} selected={tool.name === selectedTool} onClick={() => { setSelectedTool(tool.name); setToolInput(pretty(defaultInputFor(tool.name))); }} />)}</div></ScrollArea>
                  </div>
                  <div className="space-y-2">
                    <Label>Call Tool</Label>
                    <Select value={selectedTool} onValueChange={(v) => { setSelectedTool(v); setToolInput(pretty(defaultInputFor(v))); }}>
                      <SelectTrigger><SelectValue placeholder="Choose tool" /></SelectTrigger>
                      <SelectContent>{tools.map((t) => <SelectItem key={t.name} value={t.name}>{t.name}</SelectItem>)}</SelectContent>
                    </Select>
                    <Textarea className="font-mono text-xs min-h-[160px]" value={toolInput} onChange={(e) => setToolInput(e.target.value)} />
                    <Button onClick={runTool} disabled={!selectedTool || callTool.isPending}><Terminal className="h-3.5 w-3.5 mr-1" /> Call</Button>
                    {callTool.data && <pre className="max-h-[260px] overflow-auto rounded-lg border bg-muted p-3 text-xs">{pretty(callTool.data)}</pre>}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

function defaultInputFor(tool: string): Record<string, unknown> {
  switch (tool) {
    case 'workspace.read': return { path: 'README.md', limit: 65536 };
    case 'workspace.write': return { path: 'notes/demo.txt', content: 'hello from aisphere sandbox\n', append: false };
    case 'workspace.patch': return { path: 'notes/demo.txt', old: 'hello', new: 'hello patched', count: 1 };
    case 'workspace.search_files': return { path: '.', query: '', glob: '*', maxResults: 100 };
    case 'workspace.search_text': return { path: '.', query: 'TODO', maxResults: 50 };
    case 'workspace.mkdir': return { path: 'notes' };
    case 'workspace.delete': return { path: 'notes/demo.txt', recursive: false };
    case 'browser.open': return { url: 'about:blank' };
    default: return { path: '.', recursive: false };
  }
}
