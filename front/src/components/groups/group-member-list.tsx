'use client';

import { Trash2, Plus, Search, ArrowUp, ArrowDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Switch } from '@/components/ui/switch';
import { ConfirmDialog } from '@/components/shared';
import { useGroupBind, useGroupUnbind } from '@/hooks/use-groups';
import { useSkills } from '@/hooks/use-skills';
import { toast } from 'sonner';
import type { SkillGroup, SkillGroupMember } from '@/lib/api/types';
import { useState } from 'react';

interface GroupMemberListProps {
  group: SkillGroup;
  onUpdate: () => void;
}

export function GroupMemberList({ group, onUpdate }: GroupMemberListProps) {
  const [search, setSearch] = useState('');
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [newSkillName, setNewSkillName] = useState('');
  const [newVersion, setNewVersion] = useState('');
  const [newLabel, setNewLabel] = useState('stable');
  const [newRequired, setNewRequired] = useState(true);
  const [newOrder, setNewOrder] = useState(0);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const bindMutation = useGroupBind();
  const unbindMutation = useGroupUnbind();
  const { data: allSkills } = useSkills({ pageNo: 1, pageSize: 200 });

  const handleBind = async () => {
    if (!newSkillName) return;
    try {
      await bindMutation.mutateAsync({
        groupName: group.name,
        member: {
          skillName: newSkillName,
          version: newVersion || undefined,
          label: newLabel || undefined,
          required: newRequired,
          order: newOrder,
        },
      });
      toast.success(`Bound "${newSkillName}" to group`);
      setNewSkillName('');
      setNewVersion('');
      setNewLabel('stable');
      setNewRequired(true);
      setNewOrder(0);
      setShowAddDialog(false);
      onUpdate();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to bind');
    }
  };

  const handleUnbind = async (skillName: string) => {
    try {
      await unbindMutation.mutateAsync({ groupName: group.name, skillName });
      toast.success(`Unbound "${skillName}"`);
      setDeleteConfirm(null);
      onUpdate();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Failed to unbind');
    }
  };

  const members = group.members || [];

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Badge variant="secondary">{members.length} skills</Badge>
        </div>
        <Button size="sm" onClick={() => setShowAddDialog(true)}>
          <Plus className="h-3.5 w-3.5 mr-1" /> Add Skill
        </Button>
      </div>

      {/* Add Skill form */}
      {showAddDialog && (
        <div className="border rounded-lg p-3 space-y-2 bg-muted/30">
          <div className="flex items-center gap-2">
            <Input
              placeholder="Skill name"
              value={newSkillName}
              onChange={(e) => setNewSkillName(e.target.value)}
              className="h-8 text-xs flex-1"
            />
            <Input
              placeholder="Version"
              value={newVersion}
              onChange={(e) => setNewVersion(e.target.value)}
              className="h-8 text-xs w-24"
            />
            <Input
              placeholder="Label"
              value={newLabel}
              onChange={(e) => setNewLabel(e.target.value)}
              className="h-8 text-xs w-24"
            />
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Switch checked={newRequired} onCheckedChange={setNewRequired} />
              <span className="text-xs text-muted-foreground">Required</span>
            </div>
            <Input
              type="number"
              placeholder="Order"
              value={newOrder}
              onChange={(e) => setNewOrder(Number(e.target.value))}
              className="h-8 text-xs w-20"
            />
            <Button size="sm" onClick={handleBind} disabled={!newSkillName || bindMutation.isPending} className="bg-gradient-to-r from-violet-600 to-fuchsia-500">
              Bind
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setShowAddDialog(false)}>Cancel</Button>
          </div>
          {/* Quick pick from registry */}
          {allSkills && allSkills.length > 0 && (
            <div className="flex flex-wrap gap-1 pt-1">
              <span className="text-[10px] text-muted-foreground mr-1">Quick pick:</span>
              {allSkills
                .filter((s) => s.name.toLowerCase().includes(search.toLowerCase()) || !search)
                .slice(0, 10)
                .map((s) => (
                  <Badge
                    key={s.name}
                    variant="outline"
                    className="text-[10px] cursor-pointer hover:bg-violet-500/10"
                    onClick={() => setNewSkillName(s.name)}
                  >
                    {s.name}
                  </Badge>
                ))}
            </div>
          )}
        </div>
      )}

      {/* Members table */}
      {members.length === 0 ? (
        <div className="text-xs text-muted-foreground text-center py-6">
          No skills bound to this group yet.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-xs">Skill</TableHead>
              <TableHead className="text-xs">Version</TableHead>
              <TableHead className="text-xs">Label</TableHead>
              <TableHead className="text-xs">Required</TableHead>
              <TableHead className="text-xs">Order</TableHead>
              <TableHead className="text-xs w-10"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {members.map((m, i) => (
              <TableRow key={`${m.skillName}-${i}`}>
                <TableCell className="font-medium text-xs">{m.skillName}</TableCell>
                <TableCell className="text-xs font-mono">{m.version || '-'}</TableCell>
                <TableCell className="text-xs">{m.label || '-'}</TableCell>
                <TableCell>
                  <Badge variant={m.required ? 'default' : 'secondary'} className="text-[10px]">
                    {m.required ? 'Yes' : 'No'}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs">{m.order || 0}</TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 w-6 text-destructive"
                    onClick={() => setDeleteConfirm(m.skillName)}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <ConfirmDialog
        open={Boolean(deleteConfirm)}
        onOpenChange={(open) => { if (!open) setDeleteConfirm(null); }}
        title="Unbind Skill"
        description={`Are you sure you want to unbind "${deleteConfirm}" from this group?`}
        confirmLabel="Unbind"
        variant="destructive"
        onConfirm={() => deleteConfirm && handleUnbind(deleteConfirm)}
      />
    </div>
  );
}
