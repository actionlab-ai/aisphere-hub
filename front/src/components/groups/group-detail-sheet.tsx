'use client';

import { useState } from 'react';
import { Loader2, Share2 } from 'lucide-react';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { InfoItem, ConfirmDialog } from '@/components/shared';
import { GroupMemberList } from './group-member-list';
import { ResourceSharePanel } from '@/components/aihub';
import { useGroupDetail, useGroupUpdate, useGroupDelete } from '@/hooks/use-groups';
import { useResourceShares } from '@/hooks/use-shares';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';
import type { SkillGroup, GroupDetailTab, AccessMode } from '@/lib/api/types';
import { deriveAccessMode } from '@/lib/api/types';

interface GroupDetailSheetProps {
  groupName: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function AccessModeBadge({ mode }: { mode: AccessMode }) {
  const t = useT();
  const colors: Record<AccessMode, string> = {
    private: 'text-violet-600 bg-violet-50 dark:text-violet-300 dark:bg-violet-950/50 border-violet-500/30',
    shared: 'text-amber-600 bg-amber-50 dark:text-amber-300 dark:bg-amber-950/50 border-amber-500/30',
    public: 'text-emerald-600 bg-emerald-50 dark:text-emerald-300 dark:bg-emerald-950/50 border-emerald-500/30',
  };
  return (
    <Badge variant="outline" className={`text-[10px] ${colors[mode]}`}>
      {t(`accessMode.${mode}`)}
    </Badge>
  );
}

export function GroupDetailSheet({ groupName, open, onOpenChange }: GroupDetailSheetProps) {
  const t = useT();
  const [detailTab, setDetailTab] = useState<GroupDetailTab>('overview');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [editDisplayName, setEditDisplayName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [editLabelsText, setEditLabelsText] = useState('{}');

  const { data: detail, isLoading, refetch } = useGroupDetail(groupName);
  const updateMutation = useGroupUpdate();
  const deleteMutation = useGroupDelete();

  // Fetch shares to derive the access mode (replaces the old `scope` field)
  const { data: sharesData } = useResourceShares('skillset', groupName);
  const accessMode: AccessMode = sharesData?.accessMode ?? deriveAccessMode(sharesData?.items || []);

  const saveSettings = async () => {
    if (!detail) return;
    try {
      await updateMutation.mutateAsync({
        groupName: detail.name,
        data: {
          displayName: editDisplayName || undefined,
          description: editDescription || undefined,
          labels: JSON.parse(editLabelsText),
        },
      });
      toast.success(t('group.detail.settingsUpdated'));
      refetch();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('group.detail.updateFailed'));
    }
  };

  const handleDelete = async () => {
    if (!detail) return;
    try {
      await deleteMutation.mutateAsync(detail.name);
      toast.success(t('group.detail.deleted'));
      onOpenChange(false);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('group.detail.deleteFailed'));
    }
    setDeleteConfirmOpen(false);
  };

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="w-full sm:max-w-2xl p-0">
          <SheetHeader className="p-4 pb-0">
            <SheetTitle className="flex items-center gap-3">
              {detail ? (
                <>
                  <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-fuchsia-500/10 to-violet-500/10 flex items-center justify-center text-sm font-bold text-fuchsia-600 dark:text-fuchsia-400">
                    {(detail.displayName || detail.name).slice(0, 2).toUpperCase()}
                  </div>
                  <div>
                    <div>{detail.displayName || detail.name}</div>
                    <div className="text-xs font-normal text-muted-foreground flex items-center gap-2">
                      <AccessModeBadge mode={accessMode} />
                      <Badge variant="secondary" className="text-[10px]">
                        {detail.members?.length || 0} {t('group.detail.skills')}
                      </Badge>
                    </div>
                  </div>
                </>
              ) : (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" /> {t('group.detail.loading')}
                </div>
              )}
            </SheetTitle>
          </SheetHeader>

          {detail && !isLoading && (
            <div className="flex flex-col h-[calc(100%-80px)]">
              <Tabs value={detailTab} onValueChange={(v) => setDetailTab(v as GroupDetailTab)} className="flex flex-col flex-1">
                <div className="px-4 border-b">
                  <TabsList className="h-9 bg-transparent p-0 gap-1">
                    <TabsTrigger value="overview" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">
                      {t('group.detail.tabOverview')}
                    </TabsTrigger>
                    <TabsTrigger value="members" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">
                      {t('group.detail.tabMembers')}
                    </TabsTrigger>
                    <TabsTrigger value="manifest" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">
                      {t('group.detail.tabManifest')}
                    </TabsTrigger>
                    <TabsTrigger value="shares" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">
                      <Share2 className="h-3 w-3 mr-1" />{t('group.detail.tabSharing')}
                    </TabsTrigger>
                    <TabsTrigger value="settings" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">
                      {t('group.detail.tabSettings')}
                    </TabsTrigger>
                  </TabsList>
                </div>

                <ScrollArea className="flex-1">
                  {/* Overview */}
                  <TabsContent value="overview" className="p-4 space-y-4 mt-0">
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                      <InfoItem label={t('group.detail.name')} value={detail.name} mono />
                      <InfoItem label={t('group.detail.displayName')} value={detail.displayName || '-'} />
                      <InfoItem label={t('group.detail.accessMode')} value={t(`accessMode.${accessMode}`)} />
                      <InfoItem label={t('group.detail.owner')} value={detail.owner || '-'} />
                      <InfoItem label={t('group.detail.members')} value={String(detail.members?.length || 0)} />
                      <InfoItem label={t('group.detail.downloads')} value={String(detail.downloadCount || 0)} />
                    </div>
                    {detail.description && (
                      <div className="rounded-lg bg-muted/50 p-3">
                        <div className="text-[10px] text-muted-foreground">{t('group.detail.description')}</div>
                        <div className="text-sm mt-0.5">{detail.description}</div>
                      </div>
                    )}
                    {detail.labels && Object.keys(detail.labels).length > 0 && (
                      <div className="space-y-2">
                        <h4 className="text-sm font-medium">{t('group.detail.labels')}</h4>
                        <div className="flex flex-wrap gap-1.5">
                          {Object.entries(detail.labels).map(([k, v]) => (
                            <Badge key={k} variant="outline" className="text-[10px]">{k}={v}</Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  </TabsContent>

                  {/* Members */}
                  <TabsContent value="members" className="p-4 mt-0">
                    <GroupMemberList group={detail} onUpdate={() => refetch()} />
                  </TabsContent>

                  {/* Manifest */}
                  <TabsContent value="manifest" className="p-4 space-y-3 mt-0">
                    <h4 className="text-sm font-medium">{t('group.detail.manifestTitle')}</h4>
                    <p className="text-xs text-muted-foreground">{t('group.detail.manifestDesc')}</p>
                    <pre className="text-[11px] font-mono bg-muted/50 p-3 rounded-lg overflow-auto max-h-96 whitespace-pre-wrap">
                      {JSON.stringify(detail, null, 2)}
                    </pre>
                  </TabsContent>

                  {/* Sharing & Permissions */}
                  <TabsContent value="shares" className="p-4 mt-0">
                    <ResourceSharePanel
                      resourceType="skillset"
                      resourceId={detail.name}
                      owner={detail.owner}
                      compact
                    />
                  </TabsContent>

                  {/* Settings */}
                  <TabsContent value="settings" className="p-4 space-y-4 mt-0">
                    <div className="space-y-3">
                      <div className="space-y-1.5">
                        <Label>{t('group.detail.displayName')}</Label>
                        <Input value={editDisplayName} onChange={(e) => setEditDisplayName(e.target.value)} className="text-xs" />
                      </div>
                      <div className="space-y-1.5">
                        <Label>{t('group.detail.description')}</Label>
                        <Textarea value={editDescription} onChange={(e) => setEditDescription(e.target.value)} rows={3} className="text-xs" />
                      </div>
                      <div className="space-y-1.5">
                        <Label>{t('group.detail.labelsJson')}</Label>
                        <Textarea value={editLabelsText} onChange={(e) => setEditLabelsText(e.target.value)} rows={4} className="font-mono text-xs" />
                      </div>
                      {/* Access mode hint — managed by the Sharing tab */}
                      <div className="rounded-md border border-violet-500/20 bg-violet-500/5 px-3 py-2 text-[11px] text-muted-foreground">
                        <span className="font-medium text-violet-600 dark:text-violet-300">
                          {t('accessMode.label')}:
                        </span>{' '}
                        {t('accessMode.managedByShares')}{' '}
                        <span className="text-violet-600 dark:text-violet-300 font-medium">
                          {t(`accessMode.${accessMode}`)}
                        </span>
                      </div>
                      <div className="flex gap-2">
                        <Button onClick={saveSettings} className="bg-gradient-to-r from-violet-600 to-fuchsia-500" disabled={updateMutation.isPending}>
                          {t('group.detail.saveSettings')}
                        </Button>
                        <Button variant="destructive" onClick={() => setDeleteConfirmOpen(true)}>
                          {t('group.detail.deleteSkillSet')}
                        </Button>
                      </div>
                    </div>
                  </TabsContent>
                </ScrollArea>
              </Tabs>
            </div>
          )}
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={deleteConfirmOpen}
        onOpenChange={setDeleteConfirmOpen}
        title={t('group.detail.deleteTitle')}
        description={t('group.detail.deleteDesc', { name: detail?.name || '' })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  );
}
