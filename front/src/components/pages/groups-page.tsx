'use client';

import { useState } from 'react';
import { Layers, Cpu, Globe, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Search, RefreshCw, Plus } from 'lucide-react';
import { StatCard, ListSkeleton, EmptyState, ConfirmDialog } from '@/components/shared';
import { GroupCard, GroupDetailSheet, GroupCreateDialog } from '@/components/groups';
import { useGroups, useGroupDelete } from '@/hooks/use-groups';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';
import type { SkillGroup } from '@/lib/api/types';

export function GroupsPage() {
  const t = useT();
  const [search, setSearch] = useState('');
  const [selectedGroupName, setSelectedGroupName] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const { data: items = [], isLoading, error, refetch } = useGroups({
    q: search || undefined,
    pageNo: 1,
    pageSize: 50,
  });

  const deleteMutation = useGroupDelete();

  const handleDelete = async () => {
    if (!deleteConfirm) return;
    try {
      await deleteMutation.mutateAsync(deleteConfirm);
      toast.success(t('group.detail.deleted'));
      setDetailOpen(false);
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('group.detail.deleteFailed'));
    }
    setDeleteConfirm(null);
  };

  const openDetail = (group: SkillGroup) => {
    setSelectedGroupName(group.name);
    setDetailOpen(true);
  };

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
        <StatCard icon={<Layers className="h-4 w-4" />} label={t('groups.title')} value={items.length} />
        <StatCard icon={<Cpu className="h-4 w-4" />} label={t('skills.total')} value={items.reduce((a, g) => a + (g.members?.length || 0), 0)} />
        <StatCard icon={<Globe className="h-4 w-4" />} label={t('accessMode.public')} value={items.filter(g => g.scope === 'public').length} />
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input placeholder={t('groups.searchPlaceholder')} value={search} onChange={(e) => setSearch(e.target.value)} className="pl-9" />
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="h-3.5 w-3.5 mr-1" /> {t('skills.refresh')}
        </Button>
        <Button size="sm" className="bg-gradient-to-r from-violet-600 to-fuchsia-500" onClick={() => setCreateOpen(true)}>
          <Plus className="h-3.5 w-3.5 mr-1" /> {t('groups.createSkillSet')}
        </Button>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
          <AlertTriangle className="h-4 w-4 shrink-0" /> {error.message}
          <Button variant="ghost" size="sm" onClick={() => refetch()} className="ml-auto">{t('common.retry')}</Button>
        </div>
      )}

      {isLoading ? (
        <ListSkeleton count={4} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={<Layers className="h-10 w-10" />}
          title={t('groups.empty.title')}
          description={t('groups.empty.desc')}
          action={<Button size="sm" className="bg-gradient-to-r from-violet-600 to-fuchsia-500" onClick={() => setCreateOpen(true)}>{t('groups.createSkillSet')}</Button>}
        />
      ) : (
        <div className="space-y-2">
          {items.map((g) => (
            <GroupCard
              key={g.name}
              group={g}
              onClick={() => openDetail(g)}
              onEdit={() => openDetail(g)}
              onDelete={(name) => setDeleteConfirm(name)}
            />
          ))}
        </div>
      )}

      <GroupDetailSheet
        key={selectedGroupName}
        groupName={selectedGroupName}
        open={detailOpen}
        onOpenChange={setDetailOpen}
      />

      <GroupCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
      />

      <ConfirmDialog
        open={Boolean(deleteConfirm)}
        onOpenChange={(open) => { if (!open) setDeleteConfirm(null); }}
        title={t('group.detail.deleteTitle')}
        description={t('group.detail.deleteDesc', { name: deleteConfirm || '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  );
}
