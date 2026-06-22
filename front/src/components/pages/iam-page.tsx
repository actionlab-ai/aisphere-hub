'use client';

import { useState } from 'react';
import { Users, Save, Trash2, Pencil } from 'lucide-react';
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
import { useIamUsers, useIamSaveUser, useIamDeleteUser } from '@/hooks/use-iam';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';
import type { LocalUser } from '@/lib/api/types';

function EditIcon({ className }: { className?: string }) {
  return <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>;
}

export function IamPage() {
  const t = useT();
  const [form, setForm] = useState<LocalUser & { password?: string }>({
    username: '', password: '', subjectType: 'human', roles: ['agent'], permissions: ['skill:read'], namespaces: ['public'],
  });

  const { data: items = [], isLoading, refetch } = useIamUsers();
  const saveMutation = useIamSaveUser();
  const deleteMutation = useIamDeleteUser();

  const handleSave = async () => {
    try {
      await saveMutation.mutateAsync(form);
      toast.success(t('iam.saved'));
      setForm({ username: '', password: '', subjectType: 'human', roles: ['agent'], permissions: ['skill:read'], namespaces: ['public'] });
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('iam.saveFailed'));
    }
  };

  const handleDelete = async (username: string) => {
    try {
      await deleteMutation.mutateAsync(username);
      toast.success(t('iam.deleted'));
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('iam.deleteFailed'));
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader className="pb-3"><CardTitle className="text-sm flex items-center justify-between">{t('iam.title')} <Badge variant="secondary">{items.length}</Badge></CardTitle></CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="text-xs">{t('iam.username')}</TableHead>
                    <TableHead className="text-xs">Subject ID</TableHead>
                    <TableHead className="text-xs">{t('common.type')}</TableHead>
                    <TableHead className="text-xs">{t('iam.roles')}</TableHead>
                    <TableHead className="text-xs">{t('common.actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? Array.from({ length: 3 }).map((_, i) => (
                    <TableRow key={i}><TableCell colSpan={5}><Skeleton className="h-4 w-full" /></TableCell></TableRow>
                  )) : items.length === 0 ? (
                    <TableRow><TableCell colSpan={5} className="text-center text-xs text-muted-foreground py-6">{t('iam.empty')}</TableCell></TableRow>
                  ) : items.map((u) => (
                    <TableRow key={u.username}>
                      <TableCell className="font-medium text-xs">{u.username}</TableCell>
                      <TableCell className="text-xs">{u.subjectId || '-'}</TableCell>
                      <TableCell className="text-xs">{u.subjectType}</TableCell>
                      <TableCell><div className="flex flex-wrap gap-1">{(u.roles || []).map((r) => <Badge key={r} variant="secondary" className="text-[10px]">{r}</Badge>)}</div></TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => setForm(u)}><EditIcon className="h-3 w-3" /></Button>
                          <Button variant="ghost" size="sm" className="h-6 text-[10px] text-destructive" onClick={() => handleDelete(u.username)}><Trash2 className="h-3 w-3" /></Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>

        <Card className="h-fit sticky top-4">
          <CardHeader className="pb-3"><CardTitle className="text-sm">{t('iam.createEditAccount')}</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.username')}</label><Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.password')}</label><Input type="password" value={form.password || ''} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder={t('iam.passwordPlaceholder')} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">Subject ID</label><Input value={form.subjectId || ''} onChange={(e) => setForm({ ...form, subjectId: e.target.value })} placeholder="agent:xxx" className="h-8 text-xs" /></div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium">{t('common.type')}</label>
              <Select value={form.subjectType || 'human'} onValueChange={(v) => setForm({ ...form, subjectType: v })}>
                <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="human">Human</SelectItem>
                  <SelectItem value="agent">Agent</SelectItem>
                  <SelectItem value="service">Service</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.roles')}</label><Input value={(form.roles || []).join(',')} onChange={(e) => setForm({ ...form, roles: e.target.value.split(',').map(x => x.trim()).filter(Boolean) })} className="h-8 text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.permissions')}</label><Textarea value={(form.permissions || []).join('\n')} onChange={(e) => setForm({ ...form, permissions: e.target.value.split('\n').map(x => x.trim()).filter(Boolean) })} rows={3} className="text-xs" /></div>
            <div className="space-y-1.5"><label className="text-xs font-medium">{t('iam.namespaces')}</label><Input value={(form.namespaces || []).join(',')} onChange={(e) => setForm({ ...form, namespaces: e.target.value.split(',').map(x => x.trim()).filter(Boolean) })} className="h-8 text-xs" /></div>
            <Button size="sm" className="w-full bg-gradient-to-r from-violet-600 to-fuchsia-500" onClick={handleSave} disabled={saveMutation.isPending}><Save className="h-3 w-3 mr-1" /> {t('iam.save')}</Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
