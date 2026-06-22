'use client';

import { useState } from 'react';
import { Loader2, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { ResourceIdInput } from '@/components/shared';
import { useSkillDraft } from '@/hooks/use-skills';
import { useT } from '@/lib/i18n';
import { isValidResourceId, isValidVersion } from '@/lib/utils';
import { toast } from 'sonner';
import type { SkillDraft } from '@/lib/api/types';

interface SkillCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (name: string) => void;
}

export function SkillCreateDialog({ open, onOpenChange, onCreated }: SkillCreateDialogProps) {
  const t = useT();
  const [form, setForm] = useState<SkillDraft>({
    name: '',
    displayName: '',
    description: '',
    keywords: [],
    bizTags: [],
  });
  const [keywordsText, setKeywordsText] = useState('');
  const [bizTagsText, setBizTagsText] = useState('');
  const draftMutation = useSkillDraft();

  const handleCreate = async () => {
    if (!form.name) {
      toast.error(t('id.required'));
      return;
    }
    if (!isValidResourceId(form.name)) {
      toast.error(t('id.invalid'));
      return;
    }
    if (form.version && !isValidVersion(form.version)) {
      toast.error(t('id.versionInvalid'));
      return;
    }
    try {
      const data: SkillDraft = {
        ...form,
        // Note: scope is intentionally omitted — access mode is now
        // managed by the ResourceSharePanel via IAM ResourceGrants.
        keywords: keywordsText.split(',').map((x) => x.trim()).filter(Boolean),
        bizTags: bizTagsText.split(',').map((x) => x.trim()).filter(Boolean),
      };
      await draftMutation.mutateAsync(data);
      toast.success(t('create.created', { name: form.name }));
      onCreated?.(form.name);
      setForm({ name: '', displayName: '', description: '', keywords: [], bizTags: [] });
      setKeywordsText('');
      setBizTagsText('');
      onOpenChange(false);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('create.createFailed'));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('create.title')}</DialogTitle>
          <DialogDescription>{t('create.desc')}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <ResourceIdInput
              value={form.name}
              onChange={(v) => setForm({ ...form, name: v })}
              label={t('id.label')}
              placeholder={t('create.namePlaceholder')}
              disabled={draftMutation.isPending}
              required
            />
            <div className="space-y-1.5">
              <Label>{t('id.displayName')}</Label>
              <Input
                value={form.displayName || ''}
                onChange={(e) => setForm({ ...form, displayName: e.target.value })}
                placeholder={t('id.displayNamePlaceholder')}
                disabled={draftMutation.isPending}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>{t('create.description')}</Label>
            <Textarea
              value={form.description || ''}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder={t('create.descriptionPlaceholder')}
              rows={3}
              disabled={draftMutation.isPending}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <ResourceIdInput
              value={form.version || ''}
              onChange={(v) => setForm({ ...form, version: v })}
              label={t('id.versionLabel')}
              placeholder="1.0.0"
              disabled={draftMutation.isPending}
              validateVersion
              showHint
            />
            <div className="space-y-1.5">
              <Label>{t('create.keywords')}</Label>
              <Input
                value={keywordsText}
                onChange={(e) => setKeywordsText(e.target.value)}
                placeholder={t('create.keywordsPlaceholder')}
                disabled={draftMutation.isPending}
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>{t('create.bizTags')}</Label>
            <Input
              value={bizTagsText}
              onChange={(e) => setBizTagsText(e.target.value)}
              placeholder={t('create.bizTagsPlaceholder')}
              disabled={draftMutation.isPending}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={draftMutation.isPending}>{t('create.cancel')}</Button>
          <Button
            onClick={handleCreate}
            disabled={!form.name || !isValidResourceId(form.name) || draftMutation.isPending}
            className="bg-gradient-to-r from-violet-600 to-fuchsia-500"
          >
            {draftMutation.isPending ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Plus className="h-4 w-4 mr-2" />}
            {t('create.submit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
