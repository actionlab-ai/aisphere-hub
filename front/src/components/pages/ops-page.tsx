'use client';

import { useState } from 'react';
import { Activity, AlertTriangle, Key, ScrollText, Check, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { StatCard } from '@/components/shared';
import { useAuditLogs, useMetrics, useTokens, useTokenCreate, useTokenDelete, useNotifications, useNotificationMarkRead } from '@/hooks/use-ops';
import { getAccessSpace } from '@/lib/api/client';
import { getStatusColor, fmtRelativeTime } from '@/lib/utils';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';

export function OpsPage() {
  const t = useT();
  const accessSpaceId = getAccessSpace();
  const [form, setForm] = useState({ name: 'agent-token', subjectId: 'agent:demo', subjectType: 'agent', permissions: 'skill:read,skill:proposal:create', namespaces: accessSpaceId });

  const { data: audit = [], isLoading: auditLoading } = useAuditLogs({ namespaceId: accessSpaceId, pageNo: 1, pageSize: 100 });
  const { data: metrics } = useMetrics();
  const { data: tokens = [] } = useTokens();
  const { data: notifications = [] } = useNotifications({ pageNo: 1, pageSize: 30 });

  const createTokenMutation = useTokenCreate();
  const deleteTokenMutation = useTokenDelete();
  const markReadMutation = useNotificationMarkRead();

  const createToken = async () => {
    try {
      const r = await createTokenMutation.mutateAsync({
        ...form,
        permissions: form.permissions.split(',').map((x) => x.trim()),
        namespaces: form.namespaces.split(',').map((x) => x.trim()),
      });
      toast.success(t('ops.tokenCreated'));
      alert('Copy this token (shown only once):\n' + (r as { token?: string }).token);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('ops.tokenCreateFailed'));
    }
  };

  const handleDeleteToken = async (keyId: string) => {
    try {
      await deleteTokenMutation.mutateAsync(keyId);
      toast.success(t('ops.tokenDeleted'));
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('ops.tokenDeleteFailed'));
    }
  };

  const handleMarkRead = async (id: string) => {
    try {
      await markReadMutation.mutateAsync(id);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('ops.tokenDeleteFailed'));
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard icon={<Activity className="h-4 w-4" />} label={t('ops.metrics')} value={metrics?.requestsTotal || 0} />
        <StatCard icon={<AlertTriangle className="h-4 w-4" />} label={t('ops.metrics')} value={metrics?.errorsTotal || 0} />
        <StatCard icon={<Key className="h-4 w-4" />} label={t('ops.tokens')} value={tokens.length} />
        <StatCard icon={<ScrollText className="h-4 w-4" />} label={t('ops.auditLogs')} value={audit.length} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm flex items-center justify-between">{t('ops.auditLogs')} <Badge variant="secondary">{audit.length}</Badge></CardTitle></CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader><TableRow><TableHead className="text-xs">{t('common.actions')}</TableHead><TableHead className="text-xs">Resource</TableHead><TableHead className="text-xs">Operator</TableHead><TableHead className="text-xs">{t('common.created')}</TableHead></TableRow></TableHeader>
              <TableBody>
                {auditLoading ? Array.from({ length: 3 }).map((_, i) => <TableRow key={i}><TableCell colSpan={4}><Skeleton className="h-4 w-full" /></TableCell></TableRow>) :
                  audit.length === 0 ? (
                    <TableRow><TableCell colSpan={4} className="text-center text-xs text-muted-foreground py-6">{t('ops.noAuditLogs')}</TableCell></TableRow>
                  ) : audit.slice(0, 20).map((l) => (
                    <TableRow key={l.id}>
                      <TableCell className="font-medium text-xs">{l.action}</TableCell>
                      <TableCell className="text-xs">{l.resourceType}/{l.resourceName}</TableCell>
                      <TableCell className="text-xs">{l.operator}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">{fmtRelativeTime(l.createTime)}</TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm">{t('ops.metrics')}</CardTitle></CardHeader>
          <CardContent>
            {metrics ? (
              <pre className="text-[10px] font-mono bg-muted/50 p-3 rounded-lg overflow-auto max-h-80">{JSON.stringify(metrics, null, 2)}</pre>
            ) : <Skeleton className="h-40" />}
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm flex items-center justify-between">{t('ops.tokens')} <Badge variant="secondary">{tokens.length}</Badge></CardTitle></CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader><TableRow><TableHead className="text-xs">{t('common.name')}</TableHead><TableHead className="text-xs">Subject</TableHead><TableHead className="text-xs">{t('common.status')}</TableHead><TableHead className="text-xs"></TableHead></TableRow></TableHeader>
              <TableBody>
                {tokens.length === 0 ? (
                  <TableRow><TableCell colSpan={4} className="text-center text-xs text-muted-foreground py-6">{t('ops.noTokens')}</TableCell></TableRow>
                ) : tokens.map((tk) => (
                  <TableRow key={tk.keyId}>
                    <TableCell><div className="font-medium text-xs">{tk.name}</div><div className="text-[10px] text-muted-foreground font-mono">{tk.keyId}</div></TableCell>
                    <TableCell className="text-xs">{tk.subjectId}</TableCell>
                    <TableCell><Badge variant="secondary" className={`text-[10px] ${getStatusColor(tk.status)}`}>{tk.status}</Badge></TableCell>
                    <TableCell><Button variant="ghost" size="sm" className="h-6 text-[10px] text-destructive" onClick={() => handleDeleteToken(tk.keyId)}><Trash2 className="h-3 w-3" /></Button></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm">{t('ops.createToken')}</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ops.tokenName')}</label><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ops.tokenSubject')}</label><Input value={form.subjectId} onChange={(e) => setForm({ ...form, subjectId: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">{t('ns.subjectType')}</label>
              <Select value={form.subjectType} onValueChange={(v) => setForm({ ...form, subjectType: v })}>
                <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent><SelectItem value="agent">Agent</SelectItem><SelectItem value="service">Service</SelectItem><SelectItem value="human">Human</SelectItem></SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.permissions')}</label><Input value={form.permissions} onChange={(e) => setForm({ ...form, permissions: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.namespaces')}</label><Input value={form.namespaces} onChange={(e) => setForm({ ...form, namespaces: e.target.value })} className="h-8 text-xs" /></div>
            <Button size="sm" className="w-full bg-gradient-to-r from-violet-600 to-fuchsia-500" onClick={createToken} disabled={createTokenMutation.isPending}><Key className="h-3 w-3 mr-1" /> {t('ops.tokenCreate')}</Button>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-sm flex items-center justify-between">Notifications <Badge variant="secondary">{notifications.length}</Badge></CardTitle></CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader><TableRow><TableHead className="text-xs">Title</TableHead><TableHead className="text-xs">Target</TableHead><TableHead className="text-xs">{t('common.status')}</TableHead><TableHead className="text-xs"></TableHead></TableRow></TableHeader>
            <TableBody>
              {notifications.map((n) => (
                <TableRow key={n.id}>
                  <TableCell><div className="font-medium text-xs">{n.title}</div><div className="text-[10px] text-muted-foreground">{n.message}</div></TableCell>
                  <TableCell className="text-xs">{n.targetType}/{n.targetName}</TableCell>
                  <TableCell><Badge variant={n.read ? 'secondary' : 'default'} className="text-[10px]">{n.read ? 'Read' : 'Unread'}</Badge></TableCell>
                  <TableCell>{!n.read && <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => handleMarkRead(n.id)}><Check className="h-3 w-3" /></Button>}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
