'use client';

import { useState } from 'react';
import { Shield, Users, AlertTriangle, Plus, Save, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
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
import { useNamespaces, useNamespaceMembers, useNamespaceSave, useNamespaceSaveMember, useNamespaceDeleteMember } from '@/hooks/use-namespaces';
import { getAccessSpace } from '@/lib/api/client';
import { getScopeColor } from '@/lib/utils';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';

export function NamespacesPage() {
  const t = useT();
  const accessSpaceId = getAccessSpace();
  const [form, setForm] = useState({ namespaceId: '', displayName: '', description: '', visibility: 'PRIVATE' });
  const [memberForm, setMemberForm] = useState({ subjectId: '', subjectType: 'human', roles: 'viewer' });

  const { data: namespaces = [], isLoading: nsLoading, error: nsError, refetch: refetchNs } = useNamespaces();
  const { data: members = [], isLoading: mbLoading, refetch: refetchMb } = useNamespaceMembers(accessSpaceId);
  const saveNsMutation = useNamespaceSave();
  const saveMemberMutation = useNamespaceSaveMember();
  const deleteMemberMutation = useNamespaceDeleteMember();

  const saveNs = async () => {
    try {
      await saveNsMutation.mutateAsync(form);
      toast.success(t('ns.saved'));
      setForm({ namespaceId: '', displayName: '', description: '', visibility: 'PRIVATE' });
      refetchNs();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('ns.saveFailed'));
    }
  };

  const saveMember = async () => {
    try {
      await saveMemberMutation.mutateAsync({
        namespaceId: accessSpaceId,
        data: {
          ...memberForm,
          roles: memberForm.roles.split(',').map((x) => x.trim()).filter(Boolean),
        },
      });
      toast.success(t('ns.memberAdded'));
      setMemberForm({ subjectId: '', subjectType: 'human', roles: 'viewer' });
      refetchMb();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('ns.memberAddFailed'));
    }
  };

  const deleteMember = async (subjectId: string) => {
    try {
      await deleteMemberMutation.mutateAsync({ namespaceId: accessSpaceId, subjectId });
      toast.success(t('ns.memberRemoved'));
      refetchMb();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('ns.memberRemoveFailed'));
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <StatCard icon={<Shield className="h-4 w-4" />} label={t('namespaces.title')} value={namespaces.length} />
        <StatCard icon={<Users className="h-4 w-4" />} label={t('ns.members')} value={members.length} />
      </div>

      {nsError && (
        <div className="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
          <AlertTriangle className="h-4 w-4 shrink-0" /> {nsError.message}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm flex items-center justify-between">{t('namespaces.title')} <Badge variant="secondary">{namespaces.length}</Badge></CardTitle></CardHeader>
          <CardContent>
            {nsLoading ? <Skeleton className="h-40" /> : (
              <Table>
                <TableHeader>
                  <TableRow><TableHead className="text-xs">{t('ns.id')}</TableHead><TableHead className="text-xs">{t('ns.owner')}</TableHead><TableHead className="text-xs">{t('ns.visibility')}</TableHead></TableRow>
                </TableHeader>
                <TableBody>
                  {namespaces.map((n) => (
                    <TableRow key={n.namespaceId}>
                      <TableCell><div className="font-medium text-xs">{n.namespaceId}</div><div className="text-[10px] text-muted-foreground">{n.displayName}</div></TableCell>
                      <TableCell className="text-xs">{n.owner || '-'}</TableCell>
                      <TableCell><Badge variant="outline" className={`text-[10px] ${getScopeColor(n.visibility === 'PUBLIC' ? 'PUBLIC' : 'PRIVATE')}`}>{n.visibility === 'PUBLIC' ? t('ns.visibilityPublic') : t('ns.visibilityPrivate')}</Badge></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm">{t('ns.createUpdate')}</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ns.id')}</label><Input value={form.namespaceId} onChange={(e) => setForm({ ...form, namespaceId: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ns.displayName')}</label><Input value={form.displayName} onChange={(e) => setForm({ ...form, displayName: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ns.description')}</label><Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={2} className="text-xs" /></div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">{t('ns.visibility')}</label>
              <Select value={form.visibility} onValueChange={(v) => setForm({ ...form, visibility: v })}>
                <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="PRIVATE">{t('ns.visibilityPrivate')}</SelectItem>
                  <SelectItem value="PUBLIC">{t('ns.visibilityPublic')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button size="sm" className="w-full bg-gradient-to-r from-violet-600 to-fuchsia-500" onClick={saveNs}><Save className="h-3 w-3 mr-1" /> {t('ns.save')}</Button>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm flex items-center justify-between">{accessSpaceId} {t('ns.members')} <Badge variant="secondary">{members.length}</Badge></CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader><TableRow><TableHead className="text-xs">{t('ns.subjectId')}</TableHead><TableHead className="text-xs">{t('ns.subjectType')}</TableHead><TableHead className="text-xs">{t('ns.memberRoles')}</TableHead><TableHead className="text-xs"></TableHead></TableRow></TableHeader>
              <TableBody>
                {members.map((m) => (
                  <TableRow key={m.subjectId}>
                    <TableCell className="text-xs font-medium">{m.displayName || m.subjectId}</TableCell>
                    <TableCell className="text-xs">{m.subjectType}</TableCell>
                    <TableCell><div className="flex flex-wrap gap-1">{(m.roles || []).map((r) => <Badge key={r} variant="secondary" className="text-[10px]">{r}</Badge>)}</div></TableCell>
                    <TableCell><Button variant="ghost" size="sm" className="h-6 text-[10px] text-destructive" onClick={() => deleteMember(m.subjectId)}><Trash2 className="h-3 w-3" /></Button></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-sm">{t('ns.addMember')}</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ns.subjectId')}</label><Input placeholder="user:admin / agent:writer" value={memberForm.subjectId} onChange={(e) => setMemberForm({ ...memberForm, subjectId: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">{t('ns.subjectType')}</label>
              <Select value={memberForm.subjectType} onValueChange={(v) => setMemberForm({ ...memberForm, subjectType: v })}>
                <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="human">Human</SelectItem>
                  <SelectItem value="agent">Agent</SelectItem>
                  <SelectItem value="service">Service</SelectItem>
                  <SelectItem value="organization">Organization</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('ns.memberRoles')}</label><Input placeholder="owner,admin,reviewer,viewer" value={memberForm.roles} onChange={(e) => setMemberForm({ ...memberForm, roles: e.target.value })} className="h-8 text-xs" /></div>
            <Button size="sm" className="w-full bg-gradient-to-r from-violet-600 to-fuchsia-500" onClick={saveMember}><Plus className="h-3 w-3 mr-1" /> {t('ns.addMember')}</Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
