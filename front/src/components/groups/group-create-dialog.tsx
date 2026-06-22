'use client';

import { useState } from 'react';
import { Loader2, Save } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { ResourceIdInput } from '@/components/shared';
import { useGroupSave } from '@/hooks/use-groups';
import { useT } from '@/lib/i18n';
import { isValidResourceId } from '@/lib/utils';
import { toast } from 'sonner';
import type { SkillGroup } from '@/lib/api/types';

interface GroupCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editGroup?: SkillGroup | null;
}

export function GroupCreateDialog({ open, onOpenChange, editGroup }: GroupCreateDialogProps) {
  const t = useT();
  const [name, setName] = useState(editGroup?.name || '');
  const [displayName, setDisplayName] = useState(editGroup?.displayName || '');
  const [description, setDescription] = useState(editGroup?.description || '');

  const saveMutation = useGroupSave();

  const handleSave = async () => {
    if (!name) {
      toast.error(t('group.nameRequired'));
      return;
    }
    if (!isValidResourceId(name)) {
      toast.error(t('id.invalid'));
      return;
    }
    try {
      await saveMutation.mutateAsync({
        name,
        displayName: displayName || undefined,
        description: description || undefined,
        // Note: scope is intentionally omitted — access mode is now
        // managed by the ResourceSharePanel via IAM ResourceGrants.
        members: editGroup?.members,
      });
      toast.success(editGroup ? t('group.updated') : t('group.created'));
      setName('');
      setDisplayName('');
      setDescription('');
      onOpenChange(false);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : (editGroup ? t('group.updateFailed') : t('group.createFailed')));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editGroup ? t('group.edit.title') : t('group.create.title')}</DialogTitle>
          <DialogDescription>
            {editGroup ? t('group.edit.desc') : t('group.create.desc')}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <ResourceIdInput
            value={name}
            onChange={setName}
            label={t('group.name')}
            placeholder={t('group.namePlaceholder')}
            disabled={saveMutation.isPending || !!editGroup}
            required
          />
          <div className="space-y-1.5">
            <Label>{t('group.displayName')}</Label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t('group.displayNamePlaceholder')}
              disabled={saveMutation.isPending}
            />
          </div>
          <div className="space-y-1.5">
            <Label>{t('group.description')}</Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('group.descriptionPlaceholder')}
              rows={3}
              disabled={saveMutation.isPending}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saveMutation.isPending}>{t('group.cancel')}</Button>
          <Button
            onClick={handleSave}
            disabled={!name || !isValidResourceId(name) || saveMutation.isPending}
            className="bg-gradient-to-r from-violet-600 to-fuchsia-500"
          >
            {saveMutation.isPending ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
            {editGroup ? t('group.update') : t('group.create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
