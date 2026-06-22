'use client';

import { useState } from 'react';
import { Loader2, Upload } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { useSkillUpload } from '@/hooks/use-skills';
import { toast } from 'sonner';

interface SkillUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SkillUploadDialog({ open, onOpenChange }: SkillUploadDialogProps) {
  const [file, setFile] = useState<File | null>(null);
  const [targetVersion, setTargetVersion] = useState('');
  const [commitMsg, setCommitMsg] = useState('');
  const [overwrite, setOverwrite] = useState(true);
  const uploadMutation = useSkillUpload();

  const handleUpload = async () => {
    if (!file) return;
    try {
      await uploadMutation.mutateAsync({ file, overwrite, targetVersion: targetVersion || undefined, commitMsg: commitMsg || undefined });
      toast.success('Skill uploaded successfully');
      setFile(null);
      setTargetVersion('');
      setCommitMsg('');
      onOpenChange(false);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Upload failed');
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (f) setFile(f);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Upload Skill Package</DialogTitle>
          <DialogDescription>Upload a .zip file containing SKILL.md and resources</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>File</Label>
            <Input type="file" accept=".zip" onChange={handleFileChange} disabled={uploadMutation.isPending} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label>Target Version</Label>
              <Input
                value={targetVersion}
                onChange={(e) => setTargetVersion(e.target.value)}
                placeholder="e.g. 1.0.0"
                disabled={uploadMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label>Overwrite</Label>
              <div className="flex items-center gap-2 h-9">
                <Button
                  type="button"
                  variant={overwrite ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setOverwrite(true)}
                  className="text-xs"
                >
                  Yes
                </Button>
                <Button
                  type="button"
                  variant={!overwrite ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setOverwrite(false)}
                  className="text-xs"
                >
                  No
                </Button>
              </div>
            </div>
          </div>
          <div className="space-y-2">
            <Label>Commit Message</Label>
            <Input
              value={commitMsg}
              onChange={(e) => setCommitMsg(e.target.value)}
              placeholder="Describe this upload..."
              disabled={uploadMutation.isPending}
            />
          </div>
          {uploadMutation.isPending && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" /> Uploading...
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={uploadMutation.isPending}>Cancel</Button>
          <Button onClick={handleUpload} disabled={!file || uploadMutation.isPending} className="bg-gradient-to-r from-violet-600 to-fuchsia-500">
            {uploadMutation.isPending ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Upload className="h-4 w-4 mr-2" />}
            Upload
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
