'use client';

import { useState, useMemo, useCallback } from 'react';
import { Loader2, Star, StarOff, Bell, BellOff, Save, Trash2 } from 'lucide-react';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import ReactMarkdown from 'react-markdown';
import { getScopeColor, getStatusColor, versionOf, buildTree } from '@/lib/utils';
import { InfoItem, ConfirmDialog } from '@/components/shared';
import { SkillVersionTimeline } from './skill-version-timeline';
import { SkillFileBrowser } from './skill-file-browser';
import { SkillCompareView } from './skill-compare-view';
import {
  useSkillDetail,
  useSkillSocial,
  useSkillFiles,
  useSkillFileContent,
  useSkillCompare,
  useSkillLabels,
  useSkillDelete,
  useSkillPublish,
  useSkillSubmit,
  useSkillOnline,
  useSkillOffline,
  useSkillForcePublish,
  useSkillRedraft,
  useSkillBizTags,
  useSkillScope,
  useSocialStar,
  useSocialRating,
  useSocialSubscribe,
} from '@/hooks/use-skills';
import { toast } from 'sonner';
import type { Skill, DetailTab, SkillVersion, SkillVersionCompare } from '@/lib/api/types';

interface SkillDetailSheetProps {
  skill: Skill | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SkillDetailSheet({ skill, open, onOpenChange }: SkillDetailSheetProps) {
  const [detailTab, setDetailTab] = useState<DetailTab>('overview');
  const [activeVersion, setActiveVersion] = useState('');
  const [labelsText, setLabelsText] = useState('{}');
  const [bizTagsText, setBizTagsText] = useState('');
  const [scopeValue, setScopeValue] = useState('public');
  const [compareBase, setCompareBase] = useState('');
  const [compareTarget, setCompareTarget] = useState('');
  const [filePath, setFilePath] = useState('');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [editDisplayName, setEditDisplayName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [isEditing, setIsEditing] = useState(false);

  const { data: detail, isLoading: detailLoading, refetch: refetchDetail } = useSkillDetail(skill?.name || null);
  const { data: social } = useSkillSocial(skill?.name || null);
  const { data: filesData } = useSkillFiles(skill?.name || null, activeVersion || null);
  const { data: fileContentData } = useSkillFileContent(skill?.name || null, activeVersion || null, filePath || null);

  const labelsMutation = useSkillLabels();
  const deleteMutation = useSkillDelete();
  const publishMutation = useSkillPublish();
  const submitMutation = useSkillSubmit();
  const onlineMutation = useSkillOnline();
  const offlineMutation = useSkillOffline();
  const forcePublishMutation = useSkillForcePublish();
  const redraftMutation = useSkillRedraft();
  const bizTagsMutation = useSkillBizTags();
  const scopeMutation = useSkillScope();
  const starMutation = useSocialStar();
  const ratingMutation = useSocialRating();
  const subscribeMutation = useSocialSubscribe();

  const files = filesData?.files || [];
  const fileContent = fileContentData
    ? fileContentData.binary
      ? '[binary resource, download ZIP to view]'
      : fileContentData.content || ''
    : '';
  const tree = useMemo(() => buildTree(files), [files]);

  // Compute derived values from detail - use useMemo to avoid useEffect setState
  const computedLabelsText = useMemo(() => JSON.stringify(detail?.labels || {}, null, 2), [detail?.labels]);
  const computedVersion = useMemo(() => detail ? versionOf(detail) : '', [detail]);
  const computedCompareBase = useMemo(() => {
    if (!detail) return '';
    return detail.versions?.[1]?.version || detail.versions?.[0]?.version || computedVersion;
  }, [detail, computedVersion]);
  const computedScopeValue = detail?.scope || 'public';
  const computedBizTagsText = useMemo(() => {
    if (!detail) return '';
    return Array.isArray(detail.bizTags) ? detail.bizTags.join(', ') : (detail.bizTags || '');
  }, [detail?.bizTags]);
  const computedDisplayName = detail?.displayName || '';
  const computedDescription = detail?.description || '';

  // Sync state from computed values only when skill name changes
  // Parent uses key={skill?.name} to remount on skill change

  const handleCompare = useCallback(() => {
    // Compare is triggered by the component, result is fetched via useSkillCompare
  }, []);

  const { data: compareResult } = useSkillCompare(
    skill?.name || null,
    compareBase,
    compareTarget
  );

  const saveLabels = async () => {
    if (!detail) return;
    try {
      await labelsMutation.mutateAsync({ skillName: detail.name, labels: JSON.parse(labelsText) });
      toast.success('Labels saved');
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to save labels');
    }
  };

  const saveBizTags = async () => {
    if (!detail) return;
    try {
      const tags = bizTagsText.split(',').map((x) => x.trim()).filter(Boolean);
      await bizTagsMutation.mutateAsync({ skillName: detail.name, tags });
      toast.success('Business tags updated');
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to update tags');
    }
  };

  const saveScope = async (newScope: string) => {
    if (!detail) return;
    try {
      await scopeMutation.mutateAsync({ skillName: detail.name, scope: newScope });
      setScopeValue(newScope);
      toast.success('Scope updated');
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to update scope');
    }
  };

  const handleDelete = async () => {
    if (!detail) return;
    try {
      await deleteMutation.mutateAsync(detail.name);
      toast.success('Skill deleted');
      onOpenChange(false);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete');
    }
    setDeleteConfirmOpen(false);
  };

  const doVersionAction = async (action: string, version: string) => {
    if (!detail) return;
    try {
      switch (action) {
        case 'submit':
          await submitMutation.mutateAsync({ skillName: detail.name, version });
          toast.success('Version submitted');
          break;
        case 'publish':
          await publishMutation.mutateAsync({ skillName: detail.name, version });
          toast.success('Version published');
          break;
        case 'online':
          await onlineMutation.mutateAsync({ skillName: detail.name, version });
          toast.success('Skill brought online');
          break;
        case 'offline':
          await offlineMutation.mutateAsync({ skillName: detail.name, version });
          toast.success('Skill taken offline');
          break;
        case 'forcePublish':
          await forcePublishMutation.mutateAsync({ skillName: detail.name, version });
          toast.success('Version force-published');
          break;
        case 'redraft':
          await redraftMutation.mutateAsync({ skillName: detail.name, version });
          toast.success('Version redrafted');
          break;
      }
      refetchDetail();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Action failed');
    }
  };

  const openFile = (path: string) => {
    setFilePath(path);
  };

  const toggleStar = async () => {
    if (!detail || !social) return;
    try {
      await starMutation.mutateAsync({ skillName: detail.name, starred: !social.myStarred });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed');
    }
  };

  const toggleSubscribe = async () => {
    if (!detail || !social) return;
    try {
      await subscribeMutation.mutateAsync({ skillName: detail.name, subscribed: !social.mySubscribed });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed');
    }
  };

  const rate = async (n: number) => {
    if (!detail) return;
    try {
      await ratingMutation.mutateAsync({ skillName: detail.name, rating: n });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed');
    }
  };

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="w-full sm:max-w-2xl p-0">
          <SheetHeader className="p-4 pb-0">
            <SheetTitle className="flex items-center gap-3">
              {detail ? (
                <>
                  <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-violet-500/10 to-fuchsia-500/10 flex items-center justify-center text-sm font-bold text-violet-600 dark:text-violet-400">
                    {detail.name.slice(0, 2).toUpperCase()}
                  </div>
                  <div>
                    <div>{detail.displayName || detail.name}</div>
                    <div className="text-xs font-normal text-muted-foreground flex items-center gap-2">
                      <Badge variant="outline" className={`text-[10px] ${getScopeColor(detail.scope)}`}>{detail.scope || 'PUBLIC'}</Badge>
                      <Badge variant="secondary" className={`text-[10px] ${getStatusColor(detail.status || (detail.enable === false ? 'disable' : 'enable'))}`}>
                        {detail.status || (detail.enable === false ? 'disabled' : 'active')}
                      </Badge>
                    </div>
                  </div>
                </>
              ) : (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading detail...
                </div>
              )}
            </SheetTitle>
          </SheetHeader>

          {detail && !detailLoading && (
            <div className="flex flex-col h-[calc(100%-80px)]">
              <Tabs value={detailTab} onValueChange={(v) => setDetailTab(v as DetailTab)} className="flex flex-col flex-1">
                <div className="px-4 border-b">
                  <TabsList className="h-9 bg-transparent p-0 gap-1">
                    <TabsTrigger value="overview" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">Overview</TabsTrigger>
                    <TabsTrigger value="versions" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">Versions</TabsTrigger>
                    <TabsTrigger value="files" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">Files</TabsTrigger>
                    <TabsTrigger value="compare" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">Compare</TabsTrigger>
                    <TabsTrigger value="runtime" className="text-xs h-8 px-3 data-[state=active]:bg-violet-600/10 data-[state=active]:text-violet-600">Runtime</TabsTrigger>
                  </TabsList>
                </div>

                <ScrollArea className="flex-1">
                  {/* Overview Tab */}
                  <TabsContent value="overview" className="p-4 space-y-4 mt-0">
                    {/* Social strip */}
                    {social && (
                      <div className="flex flex-wrap items-center gap-2">
                        <Button variant="outline" size="sm" onClick={toggleStar} className={social.myStarred ? 'text-amber-500 border-amber-200' : ''}>
                          {social.myStarred ? <Star className="h-3.5 w-3.5 mr-1 fill-amber-500" /> : <StarOff className="h-3.5 w-3.5 mr-1" />}
                          {social.stars}
                        </Button>
                        <Button variant="outline" size="sm" onClick={toggleSubscribe} className={social.mySubscribed ? 'text-violet-500 border-violet-200' : ''}>
                          {social.mySubscribed ? <Bell className="h-3.5 w-3.5 mr-1" /> : <BellOff className="h-3.5 w-3.5 mr-1" />}
                          {social.subscribers}
                        </Button>
                        <span className="text-xs text-muted-foreground">
                          Rating: {Number(social.ratingAverage || 0).toFixed(1)} ({social.ratingCount} votes)
                        </span>
                        <div className="flex gap-0.5">
                          {[1, 2, 3, 4, 5].map((n) => (
                            <button key={n} onClick={() => rate(n)} className="hover:scale-110 transition-transform">
                              <Star className={`h-3.5 w-3.5 ${(social.myRating || 0) >= n ? 'fill-amber-400 text-amber-400' : 'text-muted-foreground'}`} />
                            </button>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Info grid */}
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                      <InfoItem label="Owner" value={detail.owner || '-'} />
                      <InfoItem label="Scope" value={detail.scope || '-'} />
                      <InfoItem label="Online" value={String(detail.onlineCnt || 0)} />
                      <InfoItem label="Downloads" value={String(detail.downloadCount || 0)} />
                      <InfoItem label="Editing" value={detail.editingVersion || '-'} />
                      <InfoItem label="Reviewing" value={detail.reviewingVersion || '-'} />
                    </div>

                    <Separator />

                    {/* Scope editor */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <Label className="text-sm font-medium">Scope</Label>
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-muted-foreground">Private</span>
                          <Switch
                            checked={scopeValue === 'public'}
                            onCheckedChange={(checked) => saveScope(checked ? 'public' : 'private')}
                          />
                          <span className="text-xs text-muted-foreground">Public</span>
                        </div>
                      </div>
                    </div>

                    {/* Business Tags */}
                    <div className="space-y-2">
                      <Label className="text-sm font-medium">Business Tags</Label>
                      <div className="flex gap-2">
                        <Input
                          value={bizTagsText}
                          onChange={(e) => setBizTagsText(e.target.value)}
                          placeholder="tag1, tag2, ..."
                          className="text-xs"
                        />
                        <Button size="sm" onClick={saveBizTags} variant="outline">Save</Button>
                      </div>
                    </div>

                    {/* Display Name / Description inline edit */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <Label className="text-sm font-medium">Display Name</Label>
                        <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={() => setIsEditing(!isEditing)}>
                          {isEditing ? 'Done' : 'Edit'}
                        </Button>
                      </div>
                      {isEditing ? (
                        <Input value={editDisplayName} onChange={(e) => setEditDisplayName(e.target.value)} className="text-xs" />
                      ) : (
                        <div className="text-sm">{editDisplayName || '-'}</div>
                      )}
                    </div>

                    <div className="space-y-2">
                      <Label className="text-sm font-medium">Description</Label>
                      {isEditing ? (
                        <Textarea value={editDescription} onChange={(e) => setEditDescription(e.target.value)} rows={3} className="text-xs" />
                      ) : (
                        <div className="text-sm text-muted-foreground">{editDescription || '-'}</div>
                      )}
                    </div>

                    <Separator />

                    {/* Labels */}
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium">Labels</h4>
                      <Textarea
                        value={labelsText}
                        onChange={(e) => setLabelsText(e.target.value)}
                        rows={6}
                        className="font-mono text-xs"
                      />
                      <div className="flex gap-2">
                        <Button size="sm" onClick={saveLabels} className="bg-gradient-to-r from-violet-600 to-fuchsia-500">
                          <Save className="h-3 w-3 mr-1" /> Save Labels
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => setDeleteConfirmOpen(true)}>
                          <Trash2 className="h-3 w-3 mr-1" /> Delete Skill
                        </Button>
                      </div>
                    </div>
                  </TabsContent>

                  {/* Versions Tab */}
                  <TabsContent value="versions" className="p-4 space-y-3 mt-0">
                    <div className="flex items-center justify-between">
                      <h4 className="text-sm font-medium">Version Timeline</h4>
                      <Badge variant="secondary">{detail.versions?.length || 0} versions</Badge>
                    </div>
                    <SkillVersionTimeline
                      versions={detail.versions || []}
                      skillName={detail.name}
                      onViewFiles={(v) => { setActiveVersion(v); setDetailTab('files'); }}
                      onAction={doVersionAction}
                    />
                  </TabsContent>

                  {/* Files Tab */}
                  <TabsContent value="files" className="p-4 mt-0">
                    <div className="flex items-center gap-2 mb-3">
                      <h4 className="text-sm font-medium">Files</h4>
                      <Select value={activeVersion} onValueChange={setActiveVersion}>
                        <SelectTrigger className="w-32 h-7 text-xs"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          {(detail.versions || []).map((v: SkillVersion) => (
                            <SelectItem key={v.version} value={v.version}>v{v.version}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="grid grid-cols-1 lg:grid-cols-5 gap-3">
                      <div className="lg:col-span-2 border rounded-lg p-2 max-h-96 overflow-auto">
                        <SkillFileBrowser nodes={tree} openFile={openFile} depth={0} selectedPath={filePath} />
                      </div>
                      <div className="lg:col-span-3 border rounded-lg p-2 max-h-96 overflow-auto">
                        <h5 className="text-xs font-medium mb-2">{filePath || 'Select a file to preview'}</h5>
                        {fileContent ? (
                          /\.(md|markdown)$/i.test(filePath) || filePath === 'SKILL.md' ? (
                            <div className="prose prose-xs dark:prose-invert max-w-none text-xs">
                              <ReactMarkdown>{fileContent}</ReactMarkdown>
                            </div>
                          ) : (
                            <pre className="text-[11px] font-mono whitespace-pre-wrap">{fileContent}</pre>
                          )
                        ) : (
                          <div className="text-xs text-muted-foreground py-4 text-center">Select a file from the tree to preview</div>
                        )}
                      </div>
                    </div>
                  </TabsContent>

                  {/* Compare Tab */}
                  <TabsContent value="compare" className="p-4 mt-0">
                    <SkillCompareView
                      versions={detail.versions || []}
                      baseVersion={compareBase}
                      targetVersion={compareTarget}
                      onBaseChange={setCompareBase}
                      onTargetChange={setCompareTarget}
                      onCompare={handleCompare}
                      compareResult={compareResult || null}
                    />
                  </TabsContent>

                  {/* Runtime Tab */}
                  <TabsContent value="runtime" className="p-4 space-y-3 mt-0">
                    <h4 className="text-sm font-medium">Runtime API Endpoints</h4>
                    <p className="text-xs text-muted-foreground">Client API is used by Agent runtime for precise skill resolution. Well-known endpoints support external discovery.</p>
                    <pre className="text-[11px] font-mono bg-muted/50 p-3 rounded-lg overflow-auto whitespace-pre-wrap">
{`# Stable version
GET /v3/client/ai/skills/${detail.name}?label=stable

# Latest version
GET /v3/client/ai/skills/${detail.name}?label=latest

# Specific version
GET /v3/client/ai/skills/${detail.name}?version=${activeVersion || '1.0.0'}

# Well-known discovery
GET /registry-global/.well-known/agent-skills/index.json

# Resolved: stable=${detail.labels?.stable || detail.stableVersion || '-'}, latest=${detail.labels?.latest || detail.latestVersion || '-'}`}
                    </pre>
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
        title="Delete Skill"
        description={`Are you sure you want to delete "${detail?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  );
}
