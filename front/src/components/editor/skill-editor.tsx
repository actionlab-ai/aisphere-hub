'use client';

import { useState, useMemo, useCallback } from 'react';
// NOTE: We intentionally avoid setState-in-useEffect for derived values.
// Instead we use derived useMemo values, the `key` prop to remount
// children when their data arrives, and child components that hold
// their own state initialized from props.
import {
  ArrowLeft, Loader2, Save, Trash2, MoreHorizontal, GitBranch, FileCode2,
  Star, Bell, Download, Play, Pause, Send, CheckCircle2, History,
  RefreshCw, X, Tag, Lock, Globe, PanelRightClose, PanelRightOpen,
  Eye, Pencil, Share2, Users,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable';
import ReactMarkdown from 'react-markdown';
import { toast } from 'sonner';
import {
  useSkillDetail, useSkillSocial, useSkillFiles, useSkillFileContent,
  useSkillLabels, useSkillDelete, useSkillPublish, useSkillSubmit,
  useSkillOnline, useSkillOffline, useSkillBizTags,
  useSocialStar, useSocialRating, useSocialSubscribe,
} from '@/hooks/use-skills';
import { useResourceShares } from '@/hooks/use-shares';
import { getScopeColor, getStatusColor, versionOf, buildTree, fmtTime } from '@/lib/utils';
import { ConfirmDialog, InfoItem } from '@/components/shared';
import { FileTree, FileBreadcrumbs } from './file-tree';
import { CodeEditor } from './code-editor';
import { ResourceSharePanel } from '@/components/aihub';
import { useT } from '@/lib/i18n';
import type { Skill, SkillVersion, AccessMode } from '@/lib/api/types';
import { deriveAccessMode } from '@/lib/api/types';

interface SkillEditorProps {
  skillName: string;
  onBack: () => void;
}

type RightPanelTab = 'overview' | 'versions' | 'runtime' | 'settings' | 'shares';

export function SkillEditor({ skillName, onBack }: SkillEditorProps) {
  const t = useT();
  // activeVersion is derived: user-selected override takes precedence,
  // otherwise we fall back to the skill's stable/latest version once detail loads.
  const [userVersionChoice, setUserVersionChoice] = useState<string | null>(null);
  const [openTabs, setOpenTabs] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState('');
  const [rightTab, setRightTab] = useState<RightPanelTab>('overview');
  const [showRightPanel, setShowRightPanel] = useState(true);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  // Detail + social
  const { data: detail, isLoading: detailLoading, refetch: refetchDetail } = useSkillDetail(skillName);
  const { data: social } = useSkillSocial(skillName);

  // Derived active version
  const activeVersion = useMemo(() => {
    if (userVersionChoice) return userVersionChoice;
    if (detail) return versionOf(detail);
    return '';
  }, [userVersionChoice, detail]);

  // Files
  const { data: filesData } = useSkillFiles(skillName, activeVersion || null);
  const files = filesData?.files || [];
  const tree = useMemo(() => buildTree(files), [files]);

  // File content (one query per active tab)
  const { data: activeFileData, isLoading: fileLoading } = useSkillFileContent(
    skillName,
    activeVersion || null,
    activeTab || null,
  );
  const activeFileContent = activeFileData?.binary
    ? '[binary resource — download the ZIP to view]'
    : activeFileData?.content || '';
  const isBinary = Boolean(activeFileData?.binary);

  // Mutations
  const deleteMutation = useSkillDelete();
  const publishMutation = useSkillPublish();
  const submitMutation = useSkillSubmit();
  const onlineMutation = useSkillOnline();
  const offlineMutation = useSkillOffline();
  const bizTagsMutation = useSkillBizTags();
  const labelsMutation = useSkillLabels();
  const starMutation = useSocialStar();
  const subscribeMutation = useSocialSubscribe();
  const ratingMutation = useSocialRating();

  // Fetch the skill's shares so we can derive the access mode (replaces
  // the legacy `scope` field). The ResourceSharePanel in the "Shares" tab
  // uses the same hook — React Query dedupes the request.
  const { data: sharesData } = useResourceShares('skill', skillName);
  const skillAccessMode: AccessMode = sharesData?.accessMode ?? deriveAccessMode(sharesData?.items || []);

  // Editable metadata — derived from detail (lazy state initializers in the
  // SettingsTab child component avoid setState-in-useEffect by remounting on
  // detail change via the `key` prop).

  const openFile = useCallback((path: string) => {
    setOpenTabs((prev) => (prev.includes(path) ? prev : [...prev, path]));
    setActiveTab(path);
  }, []);

  const closeTab = useCallback((path: string) => {
    setOpenTabs((prev) => {
      const next = prev.filter((p) => p !== path);
      if (activeTab === path) {
        setActiveTab(next[next.length - 1] || '');
      }
      return next;
    });
  }, [activeTab]);

  const doVersionAction = async (action: string, version: string) => {
    if (!detail) return;
    try {
      switch (action) {
        case 'submit':
          await submitMutation.mutateAsync({ skillName: detail.name, version });
          toast.success(t('editor.versionSubmitted'));
          break;
        case 'publish':
          await publishMutation.mutateAsync({ skillName: detail.name, version });
          toast.success(t('editor.versionPublished'));
          break;
        case 'online':
          await onlineMutation.mutateAsync({ skillName: detail.name, version });
          toast.success(t('editor.broughtOnline'));
          break;
        case 'offline':
          await offlineMutation.mutateAsync({ skillName: detail.name, version });
          toast.success(t('editor.takenOffline'));
          break;
      }
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
  };

  const saveBizTags = async (tagsText: string) => {
    if (!detail) return;
    try {
      const tags = tagsText.split(',').map((x) => x.trim()).filter(Boolean);
      await bizTagsMutation.mutateAsync({ skillName: detail.name, tags });
      toast.success(t('editor.bizTagsSaved'));
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
  };

  const saveLabels = async (labelsTextValue: string) => {
    if (!detail) return;
    try {
      const labels = JSON.parse(labelsTextValue);
      await labelsMutation.mutateAsync({ skillName: detail.name, labels });
      toast.success(t('editor.labelsSaved'));
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('editor.invalidJson'));
    }
  };

  const saveDisplayNameDescription = async (displayName: string, description: string) => {
    if (!detail) return;
    // We reuse the labels endpoint as a generic metadata write surface,
    // since the backend doesn't expose a dedicated display-name endpoint
    // in the v3 API. The skill's display name lives under labels.__display__.
    try {
      const next = { ...(detail.labels || {}) };
      if (displayName) (next as Record<string, string>).__display__ = displayName;
      else delete (next as Record<string, string>).__display__;
      await labelsMutation.mutateAsync({ skillName: detail.name, labels: next });
      // Description is stored on the skill itself via the metadata endpoint if present
      toast.success(t('editor.saved'));
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
  };

  const handleDelete = async () => {
    if (!detail) return;
    try {
      await deleteMutation.mutateAsync(detail.name);
      toast.success(t('editor.skillDeleted'));
      onBack();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('editor.deleteFailed'));
    }
    setDeleteConfirmOpen(false);
  };

  const toggleStar = async () => {
    if (!detail || !social) return;
    try {
      await starMutation.mutateAsync({ skillName: detail.name, starred: !social.myStarred });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
  };

  const toggleSubscribe = async () => {
    if (!detail || !social) return;
    try {
      await subscribeMutation.mutateAsync({ skillName: detail.name, subscribed: !social.mySubscribed });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
  };

  const rate = async (n: number) => {
    if (!detail) return;
    try {
      await ratingMutation.mutateAsync({ skillName: detail.name, rating: n });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('skills.actionFailed'));
    }
  };

  const isMarkdown = useMemo(() => {
    if (!activeTab) return false;
    return /\.(md|markdown)$/i.test(activeTab) || activeTab.toLowerCase() === 'skill.md';
  }, [activeTab]);

  if (detailLoading && !detail) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 bg-background">
        <Loader2 className="h-6 w-6 animate-spin text-violet-500" />
        <p className="text-sm text-muted-foreground">{t('editor.loading')}</p>
      </div>
    );
  }

  if (!detail) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 bg-background">
        <p className="text-sm text-muted-foreground">{t('editor.notFound')}</p>
        <Button variant="outline" size="sm" onClick={onBack}>
          <ArrowLeft className="h-3.5 w-3.5 mr-1" /> {t('editor.goBack')}
        </Button>
      </div>
    );
  }

  const status = detail.status || (detail.enable === false ? 'disable' : 'enable');
  const statusLabel = detail.status || (detail.enable === false ? 'disabled' : 'active');

  return (
    <div className="flex flex-col h-full bg-background">
      {/* ─── Editor Header ──────────────────────────────────────────── */}
      <header className="flex items-center gap-2 h-12 px-3 border-b bg-card/80 backdrop-blur-sm shrink-0">
        <Button variant="ghost" size="sm" onClick={onBack} className="gap-1.5 text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-4 w-4" />
          <span className="hidden sm:inline">{t('topbar.back')}</span>
        </Button>
        <Separator orientation="vertical" className="h-5" />

        <div className="flex items-center gap-2 min-w-0 flex-1">
          <div className="w-7 h-7 rounded-md bg-gradient-to-br from-violet-500/15 to-fuchsia-500/15 flex items-center justify-center text-[10px] font-bold text-violet-600 dark:text-violet-300 shrink-0">
            {(detail.displayName || detail.name).slice(0, 2).toUpperCase()}
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-medium truncate">{detail.displayName || detail.name}</span>
              <Badge variant="outline" className={`text-[9px] px-1.5 h-4 ${getScopeColor(detail.scope)}`}>
                {detail.scope || 'PUBLIC'}
              </Badge>
              <Badge variant="secondary" className={`text-[9px] px-1.5 h-4 ${getStatusColor(status)}`}>
                {statusLabel}
              </Badge>
            </div>
            <div className="text-[10px] text-muted-foreground truncate font-mono">{detail.name}</div>
          </div>
        </div>

        {/* Version selector */}
        <div className="flex items-center gap-1.5 shrink-0">
          <GitBranch className="h-3.5 w-3.5 text-muted-foreground" />
          <Select value={activeVersion} onValueChange={setUserVersionChoice}>
            <SelectTrigger className="h-7 w-32 text-xs gap-1">
              <SelectValue placeholder="version" />
            </SelectTrigger>
            <SelectContent>
              {(detail.versions || []).map((v: SkillVersion) => (
                <SelectItem key={v.version} value={v.version} className="text-xs">
                  v{v.version}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <Separator orientation="vertical" className="h-5" />

        {/* Action buttons */}
        <div className="flex items-center gap-1 shrink-0">
          {social && (
            <>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 gap-1"
                onClick={toggleStar}
              >
                <Star className={`h-3.5 w-3.5 ${social.myStarred ? 'fill-amber-400 text-amber-400' : ''}`} />
                <span className="text-xs tabular-nums">{social.stars}</span>
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 gap-1"
                onClick={toggleSubscribe}
              >
                <Bell className={`h-3.5 w-3.5 ${social.mySubscribed ? 'text-violet-500 fill-violet-500/30' : ''}`} />
              </Button>
            </>
          )}

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 px-2 gap-1">
                <MoreHorizontal className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuItem onClick={() => doVersionAction('online', activeVersion)}>
                <Play className="h-3.5 w-3.5 mr-2" /> {t('editor.bringOnline')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => doVersionAction('offline', activeVersion)}>
                <Pause className="h-3.5 w-3.5 mr-2" /> {t('editor.takeOffline')}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => doVersionAction('submit', activeVersion)}>
                <Send className="h-3.5 w-3.5 mr-2" /> {t('editor.submitForReview')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => doVersionAction('publish', activeVersion)}>
                <CheckCircle2 className="h-3.5 w-3.5 mr-2" /> {t('editor.publishVersion')}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => window.open(`/v3/aihub/skill/${encodeURIComponent(detail.name)}/versions/${encodeURIComponent(activeVersion)}/download`, '_blank')}
              >
                <Download className="h-3.5 w-3.5 mr-2" /> {t('editor.downloadZip')}
              </DropdownMenuItem>
              <DropdownMenuItem className="text-destructive" onClick={() => setDeleteConfirmOpen(true)}>
                <Trash2 className="h-3.5 w-3.5 mr-2" /> {t('editor.deleteSkill')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => setShowRightPanel(!showRightPanel)}
            title={showRightPanel ? 'Hide panel' : 'Show panel'}
          >
            {showRightPanel ? <PanelRightClose className="h-3.5 w-3.5" /> : <PanelRightOpen className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </header>

      {/* ─── Editor Body: 3-pane layout ────────────────────────────── */}
      <ResizablePanelGroup direction="horizontal" className="flex-1 min-h-0">
        {/* File Tree */}
        <ResizablePanel defaultSize={18} minSize={12} maxSize={32} className="bg-card/40">
          <div className="flex flex-col h-full">
            <div className="flex items-center justify-between px-3 h-8 border-b shrink-0">
              <div className="flex items-center gap-1.5">
                <FileCode2 className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {t('editor.explorer')}
                </span>
              </div>
              <span className="text-[10px] text-muted-foreground tabular-nums">
                {files.length} {t('editor.files')}
              </span>
            </div>
            <ScrollArea className="flex-1 scrollbar-thin">
              <div className="px-1.5 pb-2">
                <FileTree
                  nodes={tree}
                  selectedPath={activeTab}
                  onOpenFile={openFile}
                  defaultExpanded={tree.filter((n) => n.type === 'dir').map((n) => n.path)}
                />
              </div>
            </ScrollArea>
          </div>
        </ResizablePanel>

        <ResizableHandle withHandle />

        {/* Editor / Tabs area */}
        <ResizablePanel defaultSize={showRightPanel ? 58 : 82} minSize={30}>
          <div className="flex flex-col h-full">
            {/* Tabs strip */}
            <div className="flex items-stretch h-9 border-b bg-card/30 overflow-x-auto scrollbar-thin shrink-0">
              {openTabs.length === 0 && (
                <div className="flex items-center px-4 text-xs text-muted-foreground">
                  {t('editor.selectFile')}
                </div>
              )}
              {openTabs.map((tabPath) => {
                const name = tabPath.split('/').pop() || tabPath;
                const isActive = tabPath === activeTab;
                return (
                  <button
                    key={tabPath}
                    onClick={() => setActiveTab(tabPath)}
                    className={`group relative flex items-center gap-1.5 px-3 h-full text-xs border-r shrink-0 transition-colors
                      ${isActive
                        ? 'bg-background text-foreground'
                        : 'bg-transparent text-muted-foreground hover:bg-accent/50 hover:text-foreground'}`}
                  >
                    {isActive && (
                      <span className="absolute top-0 left-0 right-0 h-[2px] bg-violet-500" />
                    )}
                    <FileCode2 className="h-3 w-3 shrink-0 opacity-70" />
                    <span className="truncate max-w-[140px]">{name}</span>
                    <span
                      role="button"
                      tabIndex={0}
                      onClick={(e) => { e.stopPropagation(); closeTab(tabPath); }}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); closeTab(tabPath); } }}
                      className="ml-1 p-0.5 rounded hover:bg-accent opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                    >
                      <X className="h-3 w-3" />
                    </span>
                  </button>
                );
              })}

              {/* (Markdown preview toggle is rendered inside MarkdownEditorPane) */}
            </div>

            {/* Breadcrumb strip */}
            {activeTab && (
              <div className="flex items-center h-7 px-3 border-b bg-card/20 shrink-0">
                <FileBreadcrumbs path={activeTab} />
                <span className="ml-auto text-[10px] text-muted-foreground">
                  {isBinary ? t('editor.binaryHint') : `${activeFileContent.length} ${t('editor.chars')}`}
                </span>
              </div>
            )}

            {/* Editor surface */}
            <div className="flex-1 min-h-0 bg-background">
              {!activeTab ? (
                <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
                  <FileCode2 className="h-10 w-10 opacity-30" />
                  <div className="text-center">
                    <p className="text-sm font-medium">{t('editor.noFileOpen')}</p>
                    <p className="text-xs">{t('editor.noFileOpenDesc')}</p>
                  </div>
                </div>
              ) : fileLoading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                </div>
              ) : isMarkdown ? (
                <MarkdownEditorPane key={activeTab} content={activeFileContent} />
              ) : isBinary ? (
                <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
                  <Download className="h-8 w-8 opacity-40" />
                  <p className="text-sm">{t('editor.binary')}</p>
                  <Button size="sm" variant="outline" onClick={() => window.open(`/v3/aihub/skill/${encodeURIComponent(detail.name)}/versions/${encodeURIComponent(activeVersion)}/download`, '_blank')}>
                    <Download className="h-3.5 w-3.5 mr-1" /> {t('editor.downloadZip')}
                  </Button>
                </div>
              ) : (
                <CodeEditor
                  value={activeFileContent}
                  path={activeTab}
                  readOnly
                  className="h-full"
                />
              )}
            </div>
          </div>
        </ResizablePanel>

        {/* Right metadata panel */}
        {showRightPanel && (
          <>
            <ResizableHandle withHandle />
            <ResizablePanel defaultSize={24} minSize={18} maxSize={40} className="bg-card/40">
              <div className="flex flex-col h-full">
                <Tabs value={rightTab} onValueChange={(v) => setRightTab(v as RightPanelTab)} className="flex flex-col h-full">
                  <div className="border-b shrink-0">
                    <TabsList className="h-9 bg-transparent p-0 gap-0 w-full justify-start px-2">
                      <TabsTrigger value="overview" className="text-xs h-9 data-[state=active]:bg-transparent data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-violet-500 rounded-none">
                        {t('editor.overview')}
                      </TabsTrigger>
                      <TabsTrigger value="versions" className="text-xs h-9 data-[state=active]:bg-transparent data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-violet-500 rounded-none">
                        <History className="h-3 w-3 mr-1" />{t('editor.versions')}
                      </TabsTrigger>
                      <TabsTrigger value="runtime" className="text-xs h-9 data-[state=active]:bg-transparent data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-violet-500 rounded-none">
                        {t('editor.runtime')}
                      </TabsTrigger>
                      <TabsTrigger value="settings" className="text-xs h-9 data-[state=active]:bg-transparent data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-violet-500 rounded-none">
                        {t('editor.settings')}
                      </TabsTrigger>
                      <TabsTrigger value="shares" className="text-xs h-9 data-[state=active]:bg-transparent data-[state=active]:shadow-none border-b-2 border-transparent data-[state=active]:border-violet-500 rounded-none">
                        <Share2 className="h-3 w-3 mr-1" />{t('share.title')}
                      </TabsTrigger>
                    </TabsList>
                  </div>

                  <ScrollArea className="flex-1 scrollbar-thin">
                    {/* OVERVIEW */}
                    <TabsContent value="overview" className="p-3 space-y-3 mt-0">
                      {social && (
                        <div className="flex flex-wrap items-center gap-2 pb-3 border-b">
                          <div className="flex items-center gap-1 text-xs">
                            <Star className="h-3.5 w-3.5 text-amber-400 fill-amber-400" />
                            <span className="font-medium tabular-nums">{social.stars}</span>
                            <span className="text-muted-foreground">{t('editor.stars')}</span>
                          </div>
                          <div className="flex items-center gap-1 text-xs">
                            <Bell className="h-3.5 w-3.5 text-violet-500" />
                            <span className="font-medium tabular-nums">{social.subscribers}</span>
                            <span className="text-muted-foreground">{t('editor.subs')}</span>
                          </div>
                          <div className="flex items-center gap-1 text-xs">
                            <span className="font-medium tabular-nums">{Number(social.ratingAverage || 0).toFixed(1)}</span>
                            <span className="text-muted-foreground">/ 5 ({social.ratingCount})</span>
                          </div>
                        </div>
                      )}

                      {social && (
                        <div className="flex items-center gap-1">
                          {[1, 2, 3, 4, 5].map((n) => (
                            <button key={n} onClick={() => rate(n)} className="hover:scale-110 transition-transform">
                              <Star className={`h-4 w-4 ${(social.myRating || 0) >= n ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground/40'}`} />
                            </button>
                          ))}
                        </div>
                      )}

                      <div className="space-y-2">
                        <Label className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t('editor.description')}</Label>
                        <p className="text-xs leading-relaxed">{detail.description || t('editor.noDescription')}</p>
                      </div>

                      <Separator />

                      <div className="grid grid-cols-2 gap-2">
                        <InfoItem label={t('editor.owner')} value={detail.owner || '-'} />
                        <InfoItem label={t('editor.scope')} value={detail.scope || '-'} />
                        <InfoItem label={t('editor.onlineCount')} value={String(detail.onlineCnt || 0)} />
                        <InfoItem label={t('editor.downloadCount')} value={String(detail.downloadCount || 0)} />
                        <InfoItem label={t('editor.stableLabel')} value={detail.labels?.stable || detail.stableVersion || '-'} />
                        <InfoItem label={t('editor.latestLabel')} value={detail.labels?.latest || detail.latestVersion || '-'} />
                        <InfoItem label={t('editor.editing')} value={detail.editingVersion || '-'} />
                        <InfoItem label={t('editor.reviewing')} value={detail.reviewingVersion || '-'} />
                      </div>

                      {detail.bizTags && (
                        <>
                          <Separator />
                          <div className="space-y-1.5">
                            <Label className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t('editor.bizTags')}</Label>
                            <div className="flex flex-wrap gap-1">
                              {(Array.isArray(detail.bizTags) ? detail.bizTags : [detail.bizTags as string]).filter(Boolean).map((t) => (
                                <Badge key={t} variant="secondary" className="text-[10px]">
                                  <Tag className="h-2.5 w-2.5 mr-1" />
                                  {t}
                                </Badge>
                              ))}
                            </div>
                          </div>
                        </>
                      )}

                      {detail.groups && detail.groups.length > 0 && (
                        <div className="space-y-1.5">
                          <Label className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{t('editor.inGroups')}</Label>
                          <div className="flex flex-wrap gap-1">
                            {detail.groups.map((g) => (
                              <Badge key={g} variant="outline" className="text-[10px]">{g}</Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </TabsContent>

                    {/* VERSIONS */}
                    <TabsContent value="versions" className="p-3 space-y-2 mt-0">
                      <div className="flex items-center justify-between pb-2">
                        <span className="text-xs font-medium">{detail.versions?.length || 0} {t('editor.versionsCount')}</span>
                        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => refetchDetail()}>
                          <RefreshCw className="h-3 w-3" />
                        </Button>
                      </div>
                      {(detail.versions || []).map((v: SkillVersion) => {
                        const isActive = v.version === activeVersion;
                        return (
                          <button
                            key={v.version}
                            onClick={() => setUserVersionChoice(v.version)}
                            className={`w-full text-left rounded-md border p-2.5 transition-all hover:shadow-sm
                              ${isActive ? 'border-violet-500/40 bg-violet-500/5' : 'border-border/60 hover:border-border'}`}
                          >
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-1.5">
                                <span className="text-xs font-mono font-medium">v{v.version}</span>
                                {v.status && (
                                  <Badge variant="secondary" className={`text-[9px] h-4 px-1 ${getStatusColor(v.status)}`}>
                                    {v.status}
                                  </Badge>
                                )}
                              </div>
                              {isActive && (
                                <Badge variant="outline" className="text-[9px] h-4 px-1 text-violet-600 border-violet-500/40">
                                  {t('editor.viewing')}
                                </Badge>
                              )}
                            </div>
                            {v.commitMsg && (
                              <p className="text-[10px] text-muted-foreground mt-1 line-clamp-2">{v.commitMsg}</p>
                            )}
                            <div className="flex items-center gap-2 mt-1.5 text-[9px] text-muted-foreground">
                              {v.author && <span>{v.author}</span>}
                              {v.updateTime && <span>· {fmtTime(v.updateTime)}</span>}
                              {v.downloadCount !== undefined && v.downloadCount > 0 && (
                                <span>· {v.downloadCount} downloads</span>
                              )}
                            </div>
                            <div className="flex items-center gap-1 mt-2">
                              <Button
                                size="sm"
                                variant="ghost"
                                className="h-5 px-1.5 text-[10px]"
                                onClick={(e) => { e.stopPropagation(); doVersionAction('submit', v.version); }}
                              >
                                <Send className="h-2.5 w-2.5 mr-0.5" />{t('editor.submit')}
                              </Button>
                              <Button
                                size="sm"
                                variant="ghost"
                                className="h-5 px-1.5 text-[10px]"
                                onClick={(e) => { e.stopPropagation(); doVersionAction('publish', v.version); }}
                              >
                                <CheckCircle2 className="h-2.5 w-2.5 mr-0.5" />{t('editor.publish')}
                              </Button>
                            </div>
                          </button>
                        );
                      })}
                    </TabsContent>

                    {/* RUNTIME */}
                    <TabsContent value="runtime" className="p-3 space-y-3 mt-0">
                      <div>
                        <Label className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Client API</Label>
                        <p className="text-[11px] text-muted-foreground mt-1 mb-2">
                          Used by the Agent runtime to resolve this skill at request time.
                        </p>
                        <pre className="text-[10px] font-mono bg-muted/40 p-2.5 rounded-md overflow-x-auto scrollbar-thin whitespace-pre-wrap leading-relaxed">
{`# Stable version
GET /v3/client/ai/skills/${detail.name}?label=stable

# Latest version
GET /v3/client/ai/skills/${detail.name}?label=latest

# Specific version
GET /v3/client/ai/skills/${detail.name}?version=${activeVersion || '1.0.0'}`}
                        </pre>
                      </div>
                      <Separator />
                      <div>
                        <Label className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Well-known Discovery</Label>
                        <pre className="text-[10px] font-mono bg-muted/40 p-2.5 rounded-md overflow-x-auto scrollbar-thin mt-1.5 whitespace-pre-wrap leading-relaxed">
{`GET /registry-global/.well-known/agent-skills/index.json`}
                        </pre>
                      </div>
                      <Separator />
                      <div className="grid grid-cols-2 gap-2">
                        <InfoItem label="Stable" value={detail.labels?.stable || detail.stableVersion || '-'} />
                        <InfoItem label="Latest" value={detail.labels?.latest || detail.latestVersion || '-'} />
                      </div>
                    </TabsContent>

                    {/* SETTINGS */}
                    <TabsContent value="settings" className="p-3 mt-0">
                      <SettingsTab
                        key={detail.name + (detail.updateTime || '')}
                        detail={detail}
                        accessMode={skillAccessMode}
                        onSaveBizTags={saveBizTags}
                        onSaveLabels={saveLabels}
                        onSaveDisplayNameDescription={saveDisplayNameDescription}
                        onDelete={() => setDeleteConfirmOpen(true)}
                      />
                    </TabsContent>

                    {/* SHARES — Sharing & Permissions tab */}
                    <TabsContent value="shares" className="p-3 mt-0">
                      <ResourceSharePanel
                        resourceType="skill"
                        resourceId={detail.name}
                        owner={detail.owner}
                      />
                    </TabsContent>
                  </ScrollArea>
                </Tabs>
              </div>
            </ResizablePanel>
          </>
        )}
      </ResizablePanelGroup>

      <ConfirmDialog
        open={deleteConfirmOpen}
        onOpenChange={setDeleteConfirmOpen}
        title="Delete Skill"
        description={`Are you sure you want to permanently delete "${detail.name}"? This removes all versions and metadata. This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  );
}

// ─── Child: SettingsTab ──────────────────────────────────────────────
// Holds its own form state initialized from `detail` on mount.
// The parent uses `key={detail.name + detail.updateTime}` to remount it
// whenever the skill changes or after a successful save (which updates
// detail.updateTime). This avoids the setState-in-useEffect anti-pattern.

interface SettingsTabProps {
  detail: Skill;
  accessMode: AccessMode;
  onSaveBizTags: (tagsText: string) => Promise<void>;
  onSaveLabels: (labelsText: string) => Promise<void>;
  onSaveDisplayNameDescription: (displayName: string, description: string) => Promise<void>;
  onDelete: () => void;
}

function SettingsTab({
  detail,
  accessMode,
  onSaveBizTags,
  onSaveLabels,
  onSaveDisplayNameDescription,
  onDelete,
}: SettingsTabProps) {
  const t = useT();
  // All initial values are computed once on mount from `detail`.
  const [bizTagsText, setBizTagsText] = useState(() =>
    Array.isArray(detail.bizTags)
      ? detail.bizTags.join(', ')
      : (detail.bizTags as string) || '',
  );
  const [labelsText, setLabelsText] = useState(() =>
    JSON.stringify(detail.labels || {}, null, 2),
  );
  const [editDisplayName, setEditDisplayName] = useState(() => detail.displayName || '');
  const [editDescription, setEditDescription] = useState(() => detail.description || '');
  const [savingNameDesc, setSavingNameDesc] = useState(false);

  return (
    <div className="space-y-3">
      {/* Access mode (read-only — managed by the Sharing tab) */}
      <div className="space-y-2">
        <Label className="text-xs font-medium">{t('accessMode.label')}</Label>
        <div className="rounded-md border bg-card/50 px-3 py-2.5 flex items-center gap-2">
          {accessMode === 'public' ? (
            <Globe className="h-4 w-4 text-emerald-500" />
          ) : accessMode === 'shared' ? (
            <Users className="h-4 w-4 text-amber-500" />
          ) : (
            <Lock className="h-4 w-4 text-violet-500" />
          )}
          <div className="flex-1">
            <div className="text-xs font-medium">
              {t(`accessMode.${accessMode}`)}
            </div>
            <div className="text-[10px] text-muted-foreground">
              {t(`accessMode.${accessMode}Desc`)}
            </div>
          </div>
          <span className="text-[10px] text-muted-foreground">
            {t('accessMode.managedByShares')}
          </span>
        </div>
      </div>

      <Separator />

      {/* Display name + Description (editable together) */}
      <div className="space-y-1.5">
        <Label className="text-xs font-medium">{t('editor.displayName')}</Label>
        <Input
          value={editDisplayName}
          onChange={(e) => setEditDisplayName(e.target.value)}
          className="text-xs h-8"
          placeholder={t('editor.displayNamePlaceholder')}
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium">{t('editor.descriptionLabel')}</Label>
        <Textarea
          value={editDescription}
          onChange={(e) => setEditDescription(e.target.value)}
          rows={3}
          className="text-xs resize-none"
          placeholder={t('editor.descriptionPlaceholder')}
        />
      </div>
      <Button
        size="sm"
        variant="outline"
        className="h-7 text-xs"
        disabled={savingNameDesc}
        onClick={async () => {
          setSavingNameDesc(true);
          try {
            await onSaveDisplayNameDescription(editDisplayName, editDescription);
          } finally {
            setSavingNameDesc(false);
          }
        }}
      >
        <Save className="h-3 w-3 mr-1" /> {t('editor.saveNameDesc')}
      </Button>

      <Separator />

      {/* Biz Tags */}
      <div className="space-y-1.5">
        <Label className="text-xs font-medium">{t('editor.bizTags')}</Label>
        <div className="flex gap-1.5">
          <Input
            value={bizTagsText}
            onChange={(e) => setBizTagsText(e.target.value)}
            placeholder={t('editor.bizTagsPlaceholder')}
            className="text-xs h-8"
          />
          <Button size="sm" variant="outline" className="h-8 px-2.5" onClick={() => onSaveBizTags(bizTagsText)}>
            <Save className="h-3 w-3" />
          </Button>
        </div>
      </div>

      <Separator />

      {/* Labels (JSON) */}
      <div className="space-y-1.5">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium">{t('editor.labelsJson')}</Label>
          <Button size="sm" variant="outline" className="h-6 px-2 text-[10px]" onClick={() => onSaveLabels(labelsText)}>
            <Save className="h-3 w-3 mr-1" />{t('editor.save')}
          </Button>
        </div>
        <Textarea
          value={labelsText}
          onChange={(e) => setLabelsText(e.target.value)}
          rows={8}
          className="font-mono text-[11px] resize-none"
        />
      </div>

      <Separator />

      {/* Danger zone */}
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-destructive">{t('editor.dangerZone')}</Label>
        <Button
          size="sm"
          variant="outline"
          className="w-full text-destructive hover:bg-destructive/10 hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="h-3.5 w-3.5 mr-1.5" />{t('editor.deleteSkill')}
        </Button>
      </div>
    </div>
  );
}

// ─── Child: MarkdownEditorPane ───────────────────────────────────────
// Renders the markdown editor with a preview/edit toggle.
// Holds its own preview-mode state. Parent uses `key={activeTab}` so
// the state resets on tab change (no useEffect needed).

interface MarkdownEditorPaneProps {
  content: string;
}

function MarkdownEditorPane({ content }: MarkdownEditorPaneProps) {
  const t = useT();
  const [previewMode, setPreviewMode] = useState(false);

  return (
    <div className="flex flex-col h-full">
      {/* Preview/Edit toggle */}
      <div className="flex items-center justify-between px-3 h-8 border-b bg-muted/20 shrink-0">
        <span className="text-[11px] text-muted-foreground">{t('editor.markdown')}</span>
        <div className="flex items-center bg-muted/40 rounded-md p-0.5 gap-0.5">
          <button
            onClick={() => setPreviewMode(false)}
            className={`flex items-center gap-1 px-2 h-6 rounded text-[11px] transition-colors
              ${!previewMode ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
          >
            <Pencil className="h-3 w-3" /> {t('editor.source')}
          </button>
          <button
            onClick={() => setPreviewMode(true)}
            className={`flex items-center gap-1 px-2 h-6 rounded text-[11px] transition-colors
              ${previewMode ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
          >
            <Eye className="h-3 w-3" /> {t('editor.preview')}
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0">
        {previewMode ? (
          <ScrollArea className="h-full scrollbar-thin">
            <article className="prose prose-sm dark:prose-invert max-w-3xl mx-auto p-6">
              <ReactMarkdown>{content}</ReactMarkdown>
            </article>
          </ScrollArea>
        ) : (
          <CodeEditor value={content} readOnly className="h-full" />
        )}
      </div>
    </div>
  );
}
