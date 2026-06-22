'use client';

import { motion } from 'framer-motion';
import { Download, MoreHorizontal, Trash2, Pencil } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { getScopeColor } from '@/lib/utils';
import type { SkillGroup } from '@/lib/api/types';

interface GroupCardProps {
  group: SkillGroup;
  onClick: () => void;
  onEdit?: (group: SkillGroup) => void;
  onDelete?: (groupName: string) => void;
}

export function GroupCard({ group, onClick, onEdit, onDelete }: GroupCardProps) {
  return (
    <motion.div whileHover={{ x: 2 }} transition={{ duration: 0.1 }}>
      <Card className="cursor-pointer transition-all border-border/50 hover:shadow-sm group">
        <CardContent className="p-4">
          <div className="flex items-start justify-between">
            <div className="space-y-1 flex-1" onClick={onClick}>
              <div className="flex items-center gap-2">
                <h3 className="font-medium text-sm">{group.displayName || group.name}</h3>
                <Badge variant="outline" className={`text-[10px] ${getScopeColor(group.scope)}`}>{group.scope || '-'}</Badge>
              </div>
              <p className="text-xs text-muted-foreground">{group.description || 'No description'}</p>
            </div>
            <div className="flex items-center gap-1">
              <Badge variant="secondary" className="text-[10px]">{group.members?.length || 0} skills</Badge>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <MoreHorizontal className="h-3.5 w-3.5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-32">
                  <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onEdit?.(group); }}>
                    <Pencil className="h-3 w-3 mr-2" /> Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem className="text-destructive" onClick={(e) => { e.stopPropagation(); onDelete?.(group.name); }}>
                    <Trash2 className="h-3 w-3 mr-2" /> Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
          {group.members && group.members.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2" onClick={onClick}>
              {group.members.slice(0, 5).map((m, i) => (
                <Badge key={i} variant="secondary" className="text-[10px]">{m.skillName}</Badge>
              ))}
              {group.members.length > 5 && <Badge variant="secondary" className="text-[10px]">+{group.members.length - 5}</Badge>}
            </div>
          )}
          <div className="flex items-center gap-3 mt-2 text-[10px] text-muted-foreground" onClick={onClick}>
            {group.owner && <span>Owner: {group.owner}</span>}
            {group.downloadCount !== undefined && (
              <span className="flex items-center gap-0.5"><Download className="h-2.5 w-2.5" />{group.downloadCount}</span>
            )}
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}
