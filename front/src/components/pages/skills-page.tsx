'use client';

import { useState, useCallback, useMemo } from 'react';
import { Cpu, Globe, Download, Zap, AlertTriangle, Package, Layers, FolderOpen } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { StatCard, CardGridSkeleton, EmptyState, ConfirmDialog } from '@/components/shared';
import {
  SkillCard, SkillCardCompact, SkillUploadDialog, SkillCreateDialog, SkillFilters,
  type SkillViewMode,
} from '@/components/skills';
import { useOpenSkillEditor } from '@/components/layout/app-shell';
import { useSkills, useSkillOnline, useSkillOffline, useSkillSubmit, useSkillPublish, useSkillDelete } from '@/hooks/use-skills';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';
import type { Skill } from '@/lib/api/types';

export function SkillsPage() {
  const t = useT();
  const openSkillEditor = useOpenSkillEditor();

  const [search, setSearch] = useState('');
  const [groupName, setGroupName] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [scopeFilter, setScopeFilter] = useState('all');
  const [viewMode, setViewMode] = useState<SkillViewMode>('grid');
  const [uploadOpen, setUploadOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<{ action: string; skill: Skill } | null>(null);

  const { data: items = [], isLoading, error, refetch } = useSkills({
    search: search || undefined,
    skillName: search || undefined,
    groupName: groupName || undefined,
    pageNo: 1,
    pageSize: 80,
  });

  const onlineMutation = useSkillOnline();
  const offlineMutation = useSkillOffline();
  const submitMutation = useSkillSubmit();
  const publishMutation = useSkillPublish();
  const deleteMutation = useSkillDelete();

  // Filter items client-side for status and scope
  const filteredItems = useMemo(() => items.filter((s) => {
    if (statusFilter !== 'all') {
      const status = s.status || (s.enable === false ? 'disable' : 'enable');
      if (statusFilter === 'online' && status !== 'online') return false;
      if (statusFilter === 'offline' && status !== 'offline') return false;
      if (statusFilter === 'draft' && status !== 'draft') return false;
      if (statusFilter === 'published' && status !== 'published') return false;
    }
    if (scopeFilter !== 'all') {
      if ((s.scope || 'public').toLowerCase() !== scopeFilter) return false;
    }
    return true;
  }), [items, statusFilter, scopeFilter]);

  // Group skills by their `groups` field for the grouped view
  const groupedSkills = useMemo(() => {
    const groups = new Map<string, Skill[]>();
    const ungrouped: Skill[] = [];

    for (const skill of filteredItems) {
      const skillGroups = skill.groups || [];
      if (skillGroups.length === 0) {
        ungrouped.push(skill);
      } else {
        for (const g of skillGroups) {
          if (!groups.has(g)) groups.set(g, []);
          groups.get(g)!.push(skill);
        }
      }
    }

    // Sort groups alphabetically; ungrouped goes last
    const sortedGroups = Array.from(groups.entries()).sort((a, b) => a[0].localeCompare(b[0]));
    return { groups: sortedGroups, ungrouped };
  }, [filteredItems]);

  const handleAction = (action: string, skill: Skill) => {
    if (action === 'delete' || action === 'offline') {
      setConfirmAction({ action, skill });
    } else {
      executeAction(action, skill);
    }
  };

  const executeAction = async (action: string, skill: Skill) => {
    try {
      switch (action) {
        case 'online':
          await onlineMutation.mutateAsync({ skillName: skill.name });
          toast.success(`${skill.name} ${t('skills.broughtOnline')}`);
          break;
        case 'offline':
          await offlineMutation.mutateAsync({ skillName: skill.name });
          toast.success(`${skill.name} ${t('skills.takenOffline')}`);
          break;
        case 'submit': {
          const v = skill.latestVersion || skill.stableVersion || '';
          if (!v) { toast.error(t('skills.noVersion')); return; }
          await submitMutation.mutateAsync({ skillName: skill.name, version: v });
          toast.success(`${skill.name} ${t('skills.submitted')}`);
          break;
        }
        case 'publish': {
          const pv = skill.latestVersion || skill.stableVersion || '';
          if (!pv) { toast.error(t('skills.noVersion')); return; }
          await publishMutation.mutateAsync({ skillName: skill.name, version: pv });
          toast.success(`${skill.name} ${t('skills.published')}`);
          break;
        }
        case 'delete':
          await deleteMutation.mutateAsync(skill.name);
          toast.success(`${skill.name} ${t('skills.deleted')}`);
          break;
      }
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
    setConfirmAction(null);
  };

  const openEditor = useCallback((skill: Skill) => {
    openSkillEditor(skill.name);
  }, [openSkillEditor]);

  return (
    <div className="h-full overflow-y-auto scrollbar-thin">
      <div className="p-4 md:p-6 space-y-5 max-w-[1600px] mx-auto">
        {/* Hero Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <StatCard
            icon={<Cpu className="h-4 w-4" />}
            label={t('skills.total')}
            value={items.length}
            trend={`${items.filter(s => s.status === 'online').length} ${t('skills.statusOnline').toLowerCase()}`}
            accent="violet"
          />
          <StatCard
            icon={<Globe className="h-4 w-4" />}
            label={t('skills.public')}
            value={items.filter(s => (s.scope || 'public').toLowerCase() === 'public').length}
            accent="emerald"
          />
          <StatCard
            icon={<Download className="h-4 w-4" />}
            label={t('skills.downloads')}
            value={items.reduce((a, s) => a + (s.downloadCount || 0), 0)}
            accent="sky"
          />
          <StatCard
            icon={<Zap className="h-4 w-4" />}
            label={t('skills.online')}
            value={items.filter(s => s.status === 'online' || (s.onlineCnt || 0) > 0).length}
            accent="amber"
          />
        </div>

        {/* Toolbar */}
        <SkillFilters
          search={search}
          onSearchChange={setSearch}
          groupName={groupName}
          onGroupNameChange={setGroupName}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          scopeFilter={scopeFilter}
          onScopeFilterChange={setScopeFilter}
          onRefresh={() => refetch()}
          onUploadClick={() => setUploadOpen(true)}
          onCreateClick={() => setCreateOpen(true)}
          viewMode={viewMode}
          onViewModeChange={setViewMode}
        />

        {/* Error */}
        {error && (
          <div className="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span className="flex-1">{error.message}</span>
            <Button variant="ghost" size="sm" onClick={() => refetch()}>{t('common.retry')}</Button>
          </div>
        )}

        {/* Skills — Grid view */}
        {viewMode === 'grid' && (
          <>
            {isLoading ? (
              <CardGridSkeleton count={8} />
            ) : filteredItems.length === 0 ? (
              <EmptyState
                icon={<Package className="h-10 w-10" />}
                title={search || statusFilter !== 'all' || scopeFilter !== 'all' ? t('skills.empty.filtered') : t('skills.empty.title')}
                description={
                  search || statusFilter !== 'all' || scopeFilter !== 'all'
                    ? t('skills.empty.filteredDesc')
                    : t('skills.empty.desc')
                }
                action={
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>{t('skills.createDraft')}</Button>
                    <Button size="sm" className="bg-gradient-to-r from-violet-600 to-fuchsia-500 hover:from-violet-700 hover:to-fuchsia-600" onClick={() => setUploadOpen(true)}>
                      <Download className="h-3.5 w-3.5 mr-1" /> {t('skills.uploadZip')}
                    </Button>
                  </div>
                }
              />
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
                {filteredItems.map((s) => (
                  <SkillCard
                    key={s.name}
                    skill={s}
                    onClick={() => openEditor(s)}
                    onAction={handleAction}
                  />
                ))}
              </div>
            )}
          </>
        )}

        {/* Skills — Grouped view */}
        {viewMode === 'group' && (
          <>
            {isLoading ? (
              <CardGridSkeleton count={6} />
            ) : filteredItems.length === 0 ? (
              <EmptyState
                icon={<Package className="h-10 w-10" />}
                title={search || statusFilter !== 'all' || scopeFilter !== 'all' ? t('skills.empty.filtered') : t('skills.empty.title')}
                description={
                  search || statusFilter !== 'all' || scopeFilter !== 'all'
                    ? t('skills.empty.filteredDesc')
                    : t('skills.empty.desc')
                }
                action={
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>{t('skills.createDraft')}</Button>
                    <Button size="sm" className="bg-gradient-to-r from-violet-600 to-fuchsia-500 hover:from-violet-700 hover:to-fuchsia-600" onClick={() => setUploadOpen(true)}>
                      <Download className="h-3.5 w-3.5 mr-1" /> {t('skills.uploadZip')}
                    </Button>
                  </div>
                }
              />
            ) : (
              <div className="space-y-6">
                {groupedSkills.groups.length === 0 && groupedSkills.ungrouped.length === 0 && (
                  <p className="text-sm text-muted-foreground text-center py-8">{t('skills.empty.filtered')}</p>
                )}

                {/* Grouped skills */}
                {groupedSkills.groups.map(([groupName, groupSkills]) => (
                  <div key={groupName} className="space-y-3">
                    <div className="flex items-center gap-2 px-1">
                      <div className="flex items-center justify-center w-7 h-7 rounded-md bg-gradient-to-br from-violet-500/15 to-fuchsia-500/15 text-violet-600 dark:text-violet-300 shrink-0">
                        <FolderOpen className="h-3.5 w-3.5" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-sm truncate">{groupName}</h3>
                          <Badge variant="secondary" className="text-[10px] h-4 px-1.5">
                            {groupSkills.length} {t('skills.skillsInGroup')}
                          </Badge>
                        </div>
                      </div>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 pl-1">
                      {groupSkills.map((s) => (
                        <SkillCardCompact
                          key={`${groupName}-${s.name}`}
                          skill={s}
                          onClick={() => openEditor(s)}
                        />
                      ))}
                    </div>
                  </div>
                ))}

                {/* Ungrouped */}
                {groupedSkills.ungrouped.length > 0 && (
                  <div className="space-y-3">
                    <div className="flex items-center gap-2 px-1">
                      <div className="flex items-center justify-center w-7 h-7 rounded-md bg-muted text-muted-foreground shrink-0">
                        <Layers className="h-3.5 w-3.5" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-sm">{t('skills.ungrouped')}</h3>
                          <Badge variant="outline" className="text-[10px] h-4 px-1.5">
                            {groupedSkills.ungrouped.length} {t('skills.skillsInGroup')}
                          </Badge>
                        </div>
                      </div>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 pl-1">
                      {groupedSkills.ungrouped.map((s) => (
                        <SkillCardCompact
                          key={`ungrouped-${s.name}`}
                          skill={s}
                          onClick={() => openEditor(s)}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </>
        )}

        {/* Upload Dialog */}
        <SkillUploadDialog open={uploadOpen} onOpenChange={setUploadOpen} />

        {/* Create Dialog */}
        <SkillCreateDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          onCreated={(name) => {
            refetch();
            openSkillEditor(name);
          }}
        />

        {/* Confirm Dialog */}
        <ConfirmDialog
          open={Boolean(confirmAction)}
          onOpenChange={(open) => { if (!open) setConfirmAction(null); }}
          title={confirmAction?.action === 'delete' ? t('editor.deleteTitle') : t('skillCard.takeOffline')}
          description={
            confirmAction?.action === 'delete'
              ? t('editor.deleteDesc', { name: confirmAction.skill.name })
              : `${t('skillCard.takeOffline')} "${confirmAction?.skill.name}"?`
          }
          confirmLabel={confirmAction?.action === 'delete' ? t('common.delete') : t('skillCard.takeOffline')}
          variant="destructive"
          onConfirm={() => confirmAction && executeAction(confirmAction.action, confirmAction.skill)}
        />
      </div>
    </div>
  );
}
