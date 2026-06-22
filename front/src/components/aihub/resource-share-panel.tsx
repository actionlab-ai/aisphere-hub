'use client';

import { useState, useMemo } from 'react';
import {
  Loader2, Globe, Lock, Users, User, Building2, FolderKanban, Trash2,
  Plus, ShieldCheck, ShieldAlert, ShieldQuestion, Crown, Eye, Pencil,
  CheckCircle2, PlayCircle, AlertTriangle, Info,
} from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { ConfirmDialog } from '@/components/shared';
import { useT } from '@/lib/i18n';
import { cn, fmtTime } from '@/lib/utils';
import {
  useResourceShares, useCreateShare, useDeleteShare, useSetPrivate,
} from '@/hooks/use-shares';
import { toast } from 'sonner';
import type {
  AihubResourceType, ShareSubjectType, ShareRole, ResourceGrant, AccessMode,
} from '@/lib/api/types';

// ─── Props ──────────────────────────────────────────────────────────
interface ResourceSharePanelProps {
  resourceType: AihubResourceType;
  resourceId: string;
  /** Optional: pre-computed object string (e.g. 'aihub:skill:demo-skill').
   *  If omitted, the panel builds it from type + id. */
  object?: string;
  /** Optional: resource owner (shown in the access state card). */
  owner?: string;
  /** Optional: override canManage (e.g. when the parent already knows the
   *  user's role). When undefined, the panel uses the backend's hint. */
  canManage?: boolean;
  /** Optional: render in a compact mode (no card padding, used inside tight
   *  tabs). Default false. */
  compact?: boolean;
}

// ─── Helpers ────────────────────────────────────────────────────────
const SUBJECT_TYPES: ShareSubjectType[] = ['user', 'group', 'org', 'project'];
const BASE_SHARE_ROLES: ShareRole[] = ['viewer', 'editor', 'reviewer', 'admin'];

function subjectTypeIcon(t: ShareSubjectType) {
  switch (t) {
    case 'user': return <User className="h-3.5 w-3.5" />;
    case 'group': return <Users className="h-3.5 w-3.5" />;
    case 'org': return <Building2 className="h-3.5 w-3.5" />;
    case 'project': return <FolderKanban className="h-3.5 w-3.5" />;
    case 'public': return <Globe className="h-3.5 w-3.5" />;
  }
}

function roleIcon(role: ShareRole) {
  switch (role) {
    case 'viewer': return <Eye className="h-3 w-3" />;
    case 'runner': return <PlayCircle className="h-3 w-3" />;
    case 'editor': return <Pencil className="h-3 w-3" />;
    case 'reviewer': return <CheckCircle2 className="h-3 w-3" />;
    case 'admin': return <ShieldCheck className="h-3 w-3" />;
    case 'owner': return <Crown className="h-3 w-3" />;
  }
}

function accessModeIcon(mode: AccessMode) {
  switch (mode) {
    case 'private': return <Lock className="h-4 w-4" />;
    case 'shared': return <Users className="h-4 w-4" />;
    case 'public': return <Globe className="h-4 w-4" />;
  }
}

function accessModeColor(mode: AccessMode): string {
  switch (mode) {
    case 'private':
      return 'text-violet-600 bg-violet-50 dark:text-violet-300 dark:bg-violet-950/40 border-violet-500/30';
    case 'shared':
      return 'text-amber-600 bg-amber-50 dark:text-amber-300 dark:bg-amber-950/40 border-amber-500/30';
    case 'public':
      return 'text-emerald-600 bg-emerald-50 dark:text-emerald-300 dark:bg-emerald-950/40 border-emerald-500/30';
  }
}

function roleColor(role: ShareRole): string {
  switch (role) {
    case 'viewer':
      return 'text-sky-600 bg-sky-50 dark:text-sky-300 dark:bg-sky-950/40';
    case 'runner':
      return 'text-emerald-600 bg-emerald-50 dark:text-emerald-300 dark:bg-emerald-950/40';
    case 'editor':
      return 'text-violet-600 bg-violet-50 dark:text-violet-300 dark:bg-violet-950/40';
    case 'reviewer':
      return 'text-amber-600 bg-amber-50 dark:text-amber-300 dark:bg-amber-950/40';
    case 'admin':
      return 'text-rose-600 bg-rose-50 dark:text-rose-300 dark:bg-rose-950/40';
    case 'owner':
      return 'text-fuchsia-600 bg-fuchsia-50 dark:text-fuchsia-300 dark:bg-fuchsia-950/40';
  }
}

// ─── Main component ─────────────────────────────────────────────────
export function ResourceSharePanel({
  resourceType,
  resourceId,
  object,
  owner,
  canManage: canManageOverride,
  compact = false,
}: ResourceSharePanelProps) {
  const t = useT();
  const { data, isLoading, error, refetch } = useResourceShares(resourceType, resourceId);
  const createMutation = useCreateShare();
  const deleteMutation = useDeleteShare();
  const setPrivateMutation = useSetPrivate();

  const items = data?.items || [];
  const accessMode = data?.accessMode ?? 'private';
  const canManage = canManageOverride ?? data?.canManage ?? true;
  const resourceObject = object ?? `aihub:${resourceType}:${resourceId}`;
  const shareRoles: ShareRole[] = resourceType === 'agent' || resourceType === 'tool'
    ? ['viewer', 'runner', 'editor', 'reviewer', 'admin']
    : BASE_SHARE_ROLES;

  // Public grant (if any) — toggling the public switch operates on this.
  const publicGrant = useMemo(
    () => items.find((g) => g.subjectType === 'public'),
    [items],
  );

  // Collaborator grants (exclude public — that's handled by the toggle)
  const collaboratorGrants = useMemo(
    () => items.filter((g) => g.subjectType !== 'public'),
    [items],
  );

  // ─── Add-share form state ────────────────────────────────────────
  const [formSubjectType, setFormSubjectType] = useState<ShareSubjectType>('user');
  const [formSubjectId, setFormSubjectId] = useState('');
  const [formRole, setFormRole] = useState<ShareRole>('viewer');

  const [setPrivateConfirmOpen, setSetPrivateConfirmOpen] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<ResourceGrant | null>(null);

  // ─── Handlers ────────────────────────────────────────────────────
  const handleTogglePublic = async (enable: boolean) => {
    if (!canManage) {
      toast.error(t('share.noPermission'));
      return;
    }
    if (enable && !publicGrant) {
      // Create a public viewer grant
      try {
        await createMutation.mutateAsync({
          resourceType,
          resourceId,
          body: { subjectType: 'public', subjectId: '*', role: 'viewer' },
        });
        toast.success(t('share.publicOn'));
      } catch (e: unknown) {
        toast.error(e instanceof Error ? e.message : t('share.createFailed'));
      }
    } else if (!enable && publicGrant) {
      // Delete the existing public grant
      try {
        await deleteMutation.mutateAsync({
          resourceType, resourceId, grantId: publicGrant.id,
        });
        toast.success(t('share.publicOff'));
      } catch (e: unknown) {
        toast.error(e instanceof Error ? e.message : t('share.deleteFailed'));
      }
    }
  };

  const handleAddShare = async () => {
    if (!canManage) {
      toast.error(t('share.noPermission'));
      return;
    }
    if (!formSubjectId.trim()) {
      toast.error(t('share.needSubjectId'));
      return;
    }
    try {
      await createMutation.mutateAsync({
        resourceType,
        resourceId,
        body: {
          subjectType: formSubjectType,
          subjectId: formSubjectId.trim(),
          role: formRole,
        },
      });
      toast.success(t('share.created'));
      setFormSubjectId('');
      setFormRole('viewer');
      setFormSubjectType('user');
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '';
      if (msg.includes('409') || msg.toLowerCase().includes('already')) {
        toast.error(t('share.duplicateShare'));
      } else if (msg.includes('403')) {
        toast.error(t('share.noPermission'));
      } else {
        toast.error(msg || t('share.createFailed'));
      }
    }
  };

  const handleDeleteShare = async (grant: ResourceGrant) => {
    if (!canManage) {
      toast.error(t('share.noPermission'));
      return;
    }
    try {
      await deleteMutation.mutateAsync({
        resourceType, resourceId, grantId: grant.id,
      });
      toast.success(t('share.deleted'));
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('share.deleteFailed'));
    }
    setPendingDelete(null);
  };

  const handleSetPrivate = async () => {
    if (!canManage) {
      toast.error(t('share.noPermission'));
      return;
    }
    const grantIds = items.map((g) => g.id);
    if (grantIds.length === 0) {
      toast.success(t('share.setPrivateDone'));
      setSetPrivateConfirmOpen(false);
      return;
    }
    try {
      const result = await setPrivateMutation.mutateAsync({
        resourceType, resourceId, grantIds,
      });
      if (result.failures.length === 0) {
        toast.success(t('share.setPrivateDone'));
      } else {
        toast.warning(`${result.deleted} removed, ${result.failures.length} failed`);
      }
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('share.deleteFailed'));
    }
    setSetPrivateConfirmOpen(false);
  };

  // ─── Render ──────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-10 text-muted-foreground gap-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-xs">{t('share.loadingShares')}</span>
      </div>
    );
  }

  if (error) {
    const msg = error instanceof Error ? error.message : String(error);
    const isNotImpl = msg.includes('iam_resource_grants_disabled') || msg.includes('501') || msg.includes('Not Implemented');
    return (
      <div className="flex flex-col items-center justify-center py-10 text-center gap-3">
        <AlertTriangle className="h-8 w-8 text-amber-500" />
        <div>
          <p className="text-sm font-medium">{t('share.loadFailed')}</p>
          <p className="text-xs text-muted-foreground mt-1">{msg}</p>
          {isNotImpl && (
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-2 max-w-md">
              {t('share.notImpl')}
            </p>
          )}
        </div>
        <Button size="sm" variant="outline" onClick={() => refetch()}>
          {t('share.retry')}
        </Button>
      </div>
    );
  }

  const subjectIdPlaceholder =
    formSubjectType === 'user' ? t('share.subjectIdPlaceholder') :
    formSubjectType === 'group' ? t('share.subjectIdPlaceholderGroup') :
    formSubjectType === 'org' ? t('share.subjectIdPlaceholderOrg') :
    t('share.subjectIdPlaceholderProject');

  return (
    <div className={cn('space-y-4', compact ? '' : 'p-1')}>
      {/* ─── Access State Card ────────────────────────────────────── */}
      <Card className={cn('border-border/50', accessModeColor(accessMode).split(' ').filter(c => c.startsWith('border-')).join(' '))}>
        <CardContent className="p-4 space-y-3">
          <div className="flex items-start gap-3">
            <div className={cn('flex items-center justify-center w-10 h-10 rounded-lg shrink-0', accessModeColor(accessMode))}>
              {accessModeIcon(accessMode)}
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
                  {t('share.accessState')}
                </span>
                <Badge variant="outline" className={cn('text-xs', accessModeColor(accessMode))}>
                  {t(`share.access${accessMode.charAt(0).toUpperCase() + accessMode.slice(1)}`)}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                {t(`share.access${accessMode.charAt(0).toUpperCase() + accessMode.slice(1)}Desc`)}
              </p>
            </div>
          </div>

          <Separator />

          <div className="grid grid-cols-2 gap-3 text-xs">
            <div>
              <div className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">
                {t('share.owner')}
              </div>
              <div className="font-mono truncate">{owner || '-'}</div>
            </div>
            <div>
              <div className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">
                {t('share.resourceObject')}
              </div>
              <div className="font-mono truncate text-[11px]">{resourceObject}</div>
            </div>
          </div>

          {/* Current user permission hint */}
          <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground pt-1">
            {canManage ? (
              <>
                <ShieldCheck className="h-3 w-3 text-emerald-500" />
                <span>{t('share.currentUserCanManage')}</span>
              </>
            ) : (
              <>
                <ShieldAlert className="h-3 w-3 text-amber-500" />
                <span>{t('share.currentUserCannotManage')}</span>
              </>
            )}
          </div>
        </CardContent>
      </Card>

      {/* ─── Public Access Toggle ─────────────────────────────────── */}
      <Card className="border-border/50">
        <CardContent className="p-4">
          <div className="flex items-start gap-3">
            <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-300 shrink-0">
              <Globe className="h-4 w-4" />
            </div>
            <div className="flex-1 min-w-0">
              <Label className="text-sm font-medium">{t('share.publicToggle')}</Label>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t('share.publicToggleDesc')}
              </p>
            </div>
            <Switch
              checked={Boolean(publicGrant)}
              onCheckedChange={handleTogglePublic}
              disabled={!canManage || createMutation.isPending || deleteMutation.isPending}
            />
          </div>
        </CardContent>
      </Card>

      {/* ─── Collaborators Table ──────────────────────────────────── */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <Users className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">{t('share.collaborators')}</span>
            <Badge variant="secondary" className="text-[10px] h-4 px-1.5">
              {collaboratorGrants.length}
            </Badge>
          </div>
          {canManage && collaboratorGrants.length > 0 && (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 text-xs text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setSetPrivateConfirmOpen(true)}
              disabled={setPrivateMutation.isPending}
            >
              <Lock className="h-3 w-3 mr-1" />
              {t('share.setPrivate')}
            </Button>
          )}
        </div>

        {collaboratorGrants.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center border border-dashed rounded-lg border-border/60">
            <Users className="h-7 w-7 text-muted-foreground/40 mb-2" />
            <p className="text-xs font-medium">{t('share.noShares')}</p>
            <p className="text-[11px] text-muted-foreground mt-0.5 max-w-xs">
              {t('share.noSharesDesc')}
            </p>
          </div>
        ) : (
          <Card className="border-border/50">
            <ScrollArea className="max-h-[400px] scrollbar-thin">
              <Table>
                <TableHeader>
                  <TableRow className="text-[10px] uppercase tracking-wider text-muted-foreground">
                    <TableHead className="h-8">{t('share.colSubject')}</TableHead>
                    <TableHead className="h-8">{t('share.colRole')}</TableHead>
                    <TableHead className="h-8 hidden md:table-cell">{t('share.colCreatedBy')}</TableHead>
                    <TableHead className="h-8 hidden md:table-cell">{t('share.colCreatedAt')}</TableHead>
                    {canManage && <TableHead className="h-8 text-right">{t('share.colActions')}</TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {collaboratorGrants.map((grant) => (
                    <TableRow key={grant.id} className="text-xs">
                      <TableCell>
                        <div className="flex items-center gap-2 min-w-0">
                          <span className="text-muted-foreground shrink-0">
                            {subjectTypeIcon(grant.subjectType)}
                          </span>
                          <div className="min-w-0">
                            <div className="truncate font-medium">
                              {grant.subjectName || grant.subjectId}
                            </div>
                            {grant.subjectName && (
                              <div className="text-[10px] text-muted-foreground font-mono truncate">
                                {grant.subjectId}
                              </div>
                            )}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary" className={cn('text-[10px] gap-0.5', roleColor(grant.role))}>
                          {roleIcon(grant.role)}
                          {t(`share.role${grant.role.charAt(0).toUpperCase() + grant.role.slice(1)}`)}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-muted-foreground font-mono text-[11px]">
                        {grant.createdBy || '-'}
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-muted-foreground text-[11px]">
                        {grant.createdAt ? fmtTime(grant.createdAt) : '-'}
                      </TableCell>
                      {canManage && (
                        <TableCell className="text-right">
                          <Button
                            size="icon"
                            variant="ghost"
                            className="h-6 w-6 text-muted-foreground hover:text-destructive"
                            onClick={() => setPendingDelete(grant)}
                            disabled={deleteMutation.isPending}
                            title={t('common.delete')}
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </ScrollArea>
          </Card>
        )}
      </div>

      {/* ─── Add Share Form ───────────────────────────────────────── */}
      {canManage && (
        <Card className="border-border/50">
          <CardContent className="p-4 space-y-3">
            <div className="flex items-center gap-1.5">
              <Plus className="h-3.5 w-3.5 text-violet-500" />
              <span className="text-xs font-medium">{t('share.addShare')}</span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2">
              <div className="md:col-span-3 space-y-1">
                <Label className="text-[10px] text-muted-foreground uppercase tracking-wider">
                  {t('share.subjectType')}
                </Label>
                <Select
                  value={formSubjectType}
                  onValueChange={(v) => setFormSubjectType(v as ShareSubjectType)}
                >
                  <SelectTrigger className="h-8 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SUBJECT_TYPES.map((st) => (
                      <SelectItem key={st} value={st} className="text-xs">
                        <span className="flex items-center gap-1.5">
                          {subjectTypeIcon(st)}
                          {t(`share.subjectType${st.charAt(0).toUpperCase() + st.slice(1)}`)}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="md:col-span-5 space-y-1">
                <Label className="text-[10px] text-muted-foreground uppercase tracking-wider">
                  {t('share.subjectId')}
                </Label>
                <Input
                  value={formSubjectId}
                  onChange={(e) => setFormSubjectId(e.target.value)}
                  placeholder={subjectIdPlaceholder}
                  className="h-8 text-xs font-mono"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !createMutation.isPending) {
                      e.preventDefault();
                      handleAddShare();
                    }
                  }}
                />
              </div>
              <div className="md:col-span-2 space-y-1">
                <Label className="text-[10px] text-muted-foreground uppercase tracking-wider">
                  {t('share.role')}
                </Label>
                <Select
                  value={formRole}
                  onValueChange={(v) => setFormRole(v as ShareRole)}
                >
                  <SelectTrigger className="h-8 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {shareRoles.map((r) => (
                      <SelectItem key={r} value={r} className="text-xs">
                        <span className="flex items-center gap-1.5">
                          {roleIcon(r)}
                          {t(`share.role${r.charAt(0).toUpperCase() + r.slice(1)}`)}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="md:col-span-2 flex items-end">
                <Button
                  size="sm"
                  className="w-full h-8 bg-gradient-to-r from-violet-600 to-fuchsia-500 hover:from-violet-700 hover:to-fuchsia-600"
                  onClick={handleAddShare}
                  disabled={createMutation.isPending || !formSubjectId.trim()}
                >
                  {createMutation.isPending ? (
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  ) : (
                    <Plus className="h-3 w-3 mr-1" />
                  )}
                  {t('share.confirmShare')}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* ─── Role Permission Matrix ───────────────────────────────── */}
      <details className="group">
        <summary className="flex items-center gap-1.5 cursor-pointer text-xs text-muted-foreground hover:text-foreground transition-colors select-none">
          <Info className="h-3 w-3" />
          <span>{t('share.roleMatrix')}</span>
          <span className="ml-auto text-[10px] group-open:rotate-180 transition-transform">▼</span>
        </summary>
        <Card className="mt-2 border-border/50">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="text-[10px] uppercase tracking-wider text-muted-foreground">
                  <TableHead className="h-8">{t('share.role')}</TableHead>
                  <TableHead className="h-8 text-center">{t('share.roleActionRead')}</TableHead>
                  <TableHead className="h-8 text-center">{t('share.roleActionWrite')}</TableHead>
                  <TableHead className="h-8 text-center">{t('share.roleActionPublish')}</TableHead>
                  <TableHead className="h-8 text-center">{t('share.roleActionRollback')}</TableHead>
                  <TableHead className="h-8 text-center">{t('share.roleActionReview')}</TableHead>
                  <TableHead className="h-8 text-center">{t('share.roleActionManage')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {([
                  { role: 'viewer' as ShareRole,    caps: [true, false, false, false, false, false] },
                  ...(resourceType === 'agent' || resourceType === 'tool' ? [{ role: 'runner' as ShareRole, caps: [true, false, false, false, false, false] }] : []),
                  { role: 'editor' as ShareRole,    caps: [true, true,  true,  true,  false, false] },
                  { role: 'reviewer' as ShareRole,  caps: [true, false, false, false, true,  false] },
                  { role: 'admin' as ShareRole,     caps: [true, true,  true,  true,  true,  true] },
                  { role: 'owner' as ShareRole,     caps: [true, true,  true,  true,  true,  true] },
                ]).map(({ role, caps }) => (
                  <TableRow key={role} className="text-xs">
                    <TableCell>
                      <Badge variant="secondary" className={cn('text-[10px] gap-0.5', roleColor(role))}>
                        {roleIcon(role)}
                        {t(`share.role${role.charAt(0).toUpperCase() + role.slice(1)}`)}
                      </Badge>
                    </TableCell>
                    {caps.map((ok, i) => (
                      <TableCell key={i} className="text-center">
                        {ok ? (
                          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 inline" />
                        ) : (
                          <span className="text-muted-foreground/40">—</span>
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </details>

      {/* ─── Confirm dialogs ──────────────────────────────────────── */}
      <ConfirmDialog
        open={Boolean(pendingDelete)}
        onOpenChange={(open) => { if (!open) setPendingDelete(null); }}
        title={t('share.colActions')}
        description={
          pendingDelete
            ? `${t('share.colSubject')}: ${pendingDelete.subjectName || pendingDelete.subjectId} (${pendingDelete.subjectType})`
            : ''
        }
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={() => pendingDelete && handleDeleteShare(pendingDelete)}
      />

      <ConfirmDialog
        open={setPrivateConfirmOpen}
        onOpenChange={setSetPrivateConfirmOpen}
        title={t('share.setPrivateTitle')}
        description={t('share.setPrivateConfirm')}
        confirmLabel={t('share.setPrivate')}
        variant="destructive"
        onConfirm={handleSetPrivate}
      />
    </div>
  );
}
