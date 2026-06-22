'use client';

import { useMemo, useState } from 'react';
import { ExternalLink, FlaskConical, ShieldCheck, Server, Settings2 } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { asItems } from '@/lib/api/client';
import { useAccessLinks, useAccessOverview, useAccessResources, useEvaluateAccess } from '@/hooks/use-access';
import type { AccessQuickLink, AccessResourceTemplate } from '@/lib/api/types';

const actionExamples = [
  'skill:admin:read', 'skill:admin:write', 'skill:publish', 'skill:read',
  'skill:group:read', 'skill:group:write', 'skill:proposal:review', 'skill:proposal:create',
  'access:admin:read', 'notification:read', 'system:admin',
];
const objectExamples = ['skill:*', 'group:*', 'proposal:*', 'overlay:*', 'access:*', 'notification:*', 'system:*'];

export function AccessPage() {
  const overview = useAccessOverview();
  const resources = useAccessResources();
  const links = useAccessLinks();
  const evalAccess = useEvaluateAccess();

  const resourceItems = useMemo(() => asItems<AccessResourceTemplate>(resources.data) || overview.data?.resources || [], [resources.data, overview.data]);
  const linkItems = useMemo(() => asItems<AccessQuickLink>(links.data) || overview.data?.quickLinks || [], [links.data, overview.data]);
  // Lazy initial state: if resolvedSubject is already available on first
  // render, pre-fill the form. We intentionally avoid setState-in-useEffect
  // by using a lazy initializer that runs only once.
  const [evalForm, setEvalForm] = useState(() => ({
    subject: overview.data?.resolvedSubject || '',
    object: 'skill:*',
    action: 'skill:admin:read',
  }));

  const handleEvaluate = async () => {
    try {
      await evalAccess.mutateAsync(evalForm);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Permission test failed');
    }
  };

  const ov = overview.data;

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2"><ShieldCheck className="h-5 w-5 text-violet-500" /> Access Center</h1>
          <p className="text-sm text-muted-foreground">Casdoor 是平台内部 IAM 与 Casbin 权限中心；SkillHub 只做权限诊断和资源动作说明，不再本地维护 policy。</p>
        </div>
        <Badge variant="outline" className="w-fit">{ov?.provider || 'casdoor-remote'}</Badge>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        <InfoCard title="Authz Provider" value={ov?.provider || '-'} />
        <InfoCard title="Casdoor Endpoint" value={ov?.endpoint || '-'} />
        <InfoCard title="Permission" value={ov?.permissionId || '-'} />
        <InfoCard title="Resolved Subject" value={ov?.resolvedSubject || '-'} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <Card className="xl:col-span-2">
          <CardHeader><CardTitle className="text-sm flex items-center gap-1"><Server className="h-4 w-4" /> SkillHub Resource Actions</CardTitle></CardHeader>
          <CardContent className="p-0">
            {resources.isLoading ? <Skeleton className="h-48 m-4" /> : <ResourceTable items={resourceItems} />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="text-sm flex items-center gap-1"><ExternalLink className="h-4 w-4" /> Casdoor Admin Links</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            {linkItems.length === 0 && <p className="text-sm text-muted-foreground">No links available.</p>}
            {linkItems.map((l) => (
              <Button key={l.title} variant="outline" className="w-full justify-between" asChild>
                <a href={l.url} target="_blank" rel="noreferrer"><span>{l.title}</span><ExternalLink className="h-3.5 w-3.5" /></a>
              </Button>
            ))}
            <p className="text-xs text-muted-foreground pt-2">角色、角色绑定、权限策略、Casbin Model/Adapter 都在 Casdoor 控制台维护。</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <Card className="xl:col-span-2">
          <CardHeader><CardTitle className="text-sm flex items-center gap-1"><FlaskConical className="h-4 w-4" /> Permission Test</CardTitle></CardHeader>
          <CardContent className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <Field label="Subject" value={evalForm.subject} onChange={(v) => setEvalForm({ ...evalForm, subject: v })} placeholder="built-in/test01" />
            <Field label="Object" value={evalForm.object} onChange={(v) => setEvalForm({ ...evalForm, object: v })} placeholder="skill:*" />
            <Field label="Action" value={evalForm.action} onChange={(v) => setEvalForm({ ...evalForm, action: v })} placeholder="skill:admin:read" />
            <div className="md:col-span-3 flex flex-wrap gap-2">
              {objectExamples.map((v) => <Badge key={v} variant="secondary" className="cursor-pointer" onClick={() => setEvalForm({ ...evalForm, object: v })}>{v}</Badge>)}
            </div>
            <div className="md:col-span-3 flex flex-wrap gap-2">
              {actionExamples.map((v) => <Badge key={v} variant="outline" className="cursor-pointer" onClick={() => setEvalForm({ ...evalForm, action: v })}>{v}</Badge>)}
            </div>
            <div className="md:col-span-3 flex items-center gap-3">
              <Button onClick={handleEvaluate} disabled={evalAccess.isPending}>Evaluate via Casdoor</Button>
              {evalAccess.data && <Badge variant={evalAccess.data.allowed ? 'default' : 'destructive'}>{evalAccess.data.allowed ? 'ALLOW' : 'DENY'}</Badge>}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="text-sm flex items-center gap-1"><Settings2 className="h-4 w-4" /> Integration</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <KV label="Owner" value={ov?.owner} />
            <KV label="Subject Format" value={ov?.subjectFormat} />
            <KV label="Cache TTL" value={ov?.cacheTTLSeconds != null ? `${ov.cacheTTLSeconds}s` : '-'} />
            <KV label="Fail Closed" value={String(ov?.failClosed ?? true)} />
            <p className="text-xs text-muted-foreground pt-2">生产推荐 failClosed=true：Casdoor 不可用时拒绝敏感操作。</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function InfoCard({ title, value }: { title: string; value: string }) {
  return <Card><CardHeader className="pb-2"><CardTitle className="text-xs text-muted-foreground">{title}</CardTitle></CardHeader><CardContent className="text-sm font-semibold truncate" title={value}>{value}</CardContent></Card>;
}

function Field({ label, value, onChange, placeholder }: { label: string; value: string; placeholder?: string; onChange: (v: string) => void }) {
  return <div className="space-y-1"><label className="text-xs text-muted-foreground">{label}</label><Input value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} /></div>;
}

function KV({ label, value }: { label: string; value?: string }) {
  return <div className="flex justify-between gap-3 border-b py-1"><span className="text-muted-foreground">{label}</span><span className="font-mono text-xs truncate">{value || '-'}</span></div>;
}

function ResourceTable({ items }: { items: AccessResourceTemplate[] }) {
  return <Table><TableHeader><TableRow><TableHead>Area</TableHead><TableHead>Object</TableHead><TableHead>Action</TableHead><TableHead>Description</TableHead></TableRow></TableHeader><TableBody>{items.map((r) => <TableRow key={`${r.object}-${r.action}`}><TableCell>{r.area}</TableCell><TableCell><code>{r.object}</code></TableCell><TableCell><code>{r.action}</code></TableCell><TableCell className="text-muted-foreground">{r.description}</TableCell></TableRow>)}</TableBody></Table>;
}
