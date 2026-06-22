'use client';

import { ArrowLeftRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { SkillVersion, SkillVersionCompare } from '@/lib/api/types';

interface SkillCompareViewProps {
  versions: SkillVersion[];
  baseVersion: string;
  targetVersion: string;
  onBaseChange: (v: string) => void;
  onTargetChange: (v: string) => void;
  onCompare: () => void;
  compareResult: SkillVersionCompare | null;
}

export function SkillCompareView({
  versions,
  baseVersion,
  targetVersion,
  onBaseChange,
  onTargetChange,
  onCompare,
  compareResult,
}: SkillCompareViewProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Select value={baseVersion} onValueChange={onBaseChange}>
          <SelectTrigger className="w-32 h-8 text-xs"><SelectValue placeholder="Base" /></SelectTrigger>
          <SelectContent>
            {versions.map((v) => (
              <SelectItem key={v.version} value={v.version}>v{v.version}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <ArrowLeftRight className="h-4 w-4 text-muted-foreground" />
        <Select value={targetVersion} onValueChange={onTargetChange}>
          <SelectTrigger className="w-32 h-8 text-xs"><SelectValue placeholder="Target" /></SelectTrigger>
          <SelectContent>
            {versions.map((v) => (
              <SelectItem key={v.version} value={v.version}>v{v.version}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button size="sm" onClick={onCompare} className="bg-gradient-to-r from-violet-600 to-fuchsia-500">
          Compare
        </Button>
      </div>
      {compareResult && (
        <div className="grid grid-cols-2 gap-3">
          <Card className="border-border/50">
            <CardHeader className="p-3 pb-1"><CardTitle className="text-xs">Base (v{baseVersion})</CardTitle></CardHeader>
            <CardContent className="p-3 pt-1">
              <pre className="text-[11px] font-mono overflow-auto max-h-80 whitespace-pre-wrap bg-muted/50 p-2 rounded">
                {compareResult.baseSkillMd || 'No content'}
              </pre>
            </CardContent>
          </Card>
          <Card className="border-border/50">
            <CardHeader className="p-3 pb-1"><CardTitle className="text-xs">Target (v{targetVersion})</CardTitle></CardHeader>
            <CardContent className="p-3 pt-1">
              <pre className="text-[11px] font-mono overflow-auto max-h-80 whitespace-pre-wrap bg-muted/50 p-2 rounded">
                {compareResult.targetSkillMd || 'No content'}
              </pre>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
