'use client';

import { motion } from 'framer-motion';
import { Download, MoreHorizontal, Play, Pause, Send, CheckCircle2, Trash2, FileCode2, ChevronRight, Layers } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { getStatusColor, getScopeColor } from '@/lib/utils';
import { useT } from '@/lib/i18n';
import type { Skill } from '@/lib/api/types';

interface SkillCardProps {
  skill: Skill;
  onClick: () => void;
  onAction?: (action: string, skill: Skill) => void;
}

export function SkillCard({ skill, onClick, onAction }: SkillCardProps) {
  const t = useT();
  const status = skill.status || (skill.enable === false ? 'disable' : 'enable');
  // Map raw status to a translation key suffix
  const statusKeySuffix =
    status === 'disable' ? 'disabled' :
    status === 'enable' || status === 'active' ? 'active' :
    status === 'online' ? 'online' :
    status === 'offline' ? 'offline' :
    status === 'draft' ? 'draft' :
    status === 'published' ? 'published' :
    status;
  const statusLabel = t(`skills.status${statusKeySuffix.charAt(0).toUpperCase()}${statusKeySuffix.slice(1)}`);
  const stableVersion = skill.labels?.stable || skill.stableVersion;
  const latestVersion = skill.labels?.latest || skill.latestVersion;

  return (
    <motion.div
      whileHover={{ y: -2 }}
      transition={{ duration: 0.15 }}
      onClick={onClick}
      className="cursor-pointer"
    >
      <Card className="hover:shadow-md hover:shadow-violet-500/5 hover:border-violet-500/30 transition-all border-border/50 group relative overflow-hidden">
        {/* Subtle top gradient bar */}
        <div className="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-violet-500/0 via-violet-500/0 to-violet-500/0 group-hover:from-violet-500/60 group-hover:via-fuchsia-500/60 group-hover:to-violet-500/0 transition-all duration-300" />
        <CardContent className="p-4 space-y-3">
          <div className="flex items-start justify-between gap-2">
            <div className="flex items-center gap-2.5 min-w-0">
              <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-violet-500/15 to-fuchsia-500/15 flex items-center justify-center text-xs font-bold text-violet-600 dark:text-violet-300 shrink-0">
                {(skill.displayName || skill.name).slice(0, 2).toUpperCase()}
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="font-medium text-sm truncate group-hover:text-violet-600 transition-colors leading-tight">
                  {skill.displayName || skill.name}
                </h3>
                <div className="text-[10px] text-muted-foreground font-mono truncate">{skill.name}</div>
              </div>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreHorizontal className="h-3.5 w-3.5" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-44" onClick={(e) => e.stopPropagation()}>
                <DropdownMenuItem onClick={() => onAction?.('online', skill)}>
                  <Play className="h-3.5 w-3.5 mr-2" /> {t('skillCard.bringOnline')}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onAction?.('offline', skill)}>
                  <Pause className="h-3.5 w-3.5 mr-2" /> {t('skillCard.takeOffline')}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => onAction?.('submit', skill)}>
                  <Send className="h-3.5 w-3.5 mr-2" /> {t('skillCard.submit')}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onAction?.('publish', skill)}>
                  <CheckCircle2 className="h-3.5 w-3.5 mr-2" /> {t('skillCard.publish')}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="text-destructive" onClick={() => onAction?.('delete', skill)}>
                  <Trash2 className="h-3.5 w-3.5 mr-2" /> {t('skillCard.delete')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed min-h-[2rem]">
            {skill.description || t('skillCard.noDesc')}
          </p>

          <div className="flex items-center gap-1.5 flex-wrap">
            <Badge variant="secondary" className={`text-[10px] px-1.5 h-4 ${getStatusColor(status)}`}>
              {statusLabel}
            </Badge>
            <Badge variant="outline" className={`text-[10px] px-1.5 h-4 ${getScopeColor(skill.scope)}`}>
              {/* Display the legacy `scope` field using access-mode labels.
                  The backend may eventually stop returning `scope` in favor
                  of the IAM ResourceGrant-derived `accessMode`. Until then,
                  we map public→公开/Public, private→私有/Private. */}
              {t(`accessMode.${(skill.scope || 'public').toLowerCase() === 'public' ? 'public' : 'private'}`)}
            </Badge>
            {skill.bizTags && Array.isArray(skill.bizTags) && skill.bizTags.length > 0 && (
              skill.bizTags.slice(0, 2).map((tag) => (
                <Badge key={tag} variant="outline" className="text-[9px] px-1 h-4 text-muted-foreground">
                  {tag}
                </Badge>
              ))
            )}
          </div>

          <div className="grid grid-cols-3 gap-2 pt-1 border-t">
            <div>
              <div className="text-[9px] text-muted-foreground uppercase tracking-wide">{t('skillCard.stable')}</div>
              <div className="text-[10px] font-mono text-foreground truncate">
                {stableVersion ? `v${stableVersion}` : '—'}
              </div>
            </div>
            <div>
              <div className="text-[9px] text-muted-foreground uppercase tracking-wide">{t('skillCard.latest')}</div>
              <div className="text-[10px] font-mono text-foreground truncate">
                {latestVersion ? `v${latestVersion}` : '—'}
              </div>
            </div>
            <div className="text-right">
              <div className="text-[9px] text-muted-foreground uppercase tracking-wide flex items-center justify-end gap-0.5">
                <Download className="h-2 w-2" /> {t('skillCard.pulls')}
              </div>
              <div className="text-[10px] font-mono text-foreground tabular-nums">{skill.downloadCount || 0}</div>
            </div>
          </div>

          <div className="flex items-center justify-between text-[10px] text-muted-foreground pt-0.5">
            <span>{skill.owner || 'unknown'}</span>
            <span className="flex items-center gap-1 group-hover:text-violet-600 transition-colors">
              <FileCode2 className="h-3 w-3" />
              <span>{t('skillCard.openEditor')}</span>
              <ChevronRight className="h-2.5 w-2.5" />
            </span>
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}

// ─── Compact card for grouped views ─────────────────────────────────
interface SkillCardCompactProps {
  skill: Skill;
  onClick: () => void;
}

export function SkillCardCompact({ skill, onClick }: SkillCardCompactProps) {
  const t = useT();
  const status = skill.status || (skill.enable === false ? 'disable' : 'enable');
  const statusKeySuffix =
    status === 'disable' ? 'disabled' :
    status === 'enable' || status === 'active' ? 'active' :
    status === 'online' ? 'online' :
    status === 'offline' ? 'offline' :
    status === 'draft' ? 'draft' :
    status === 'published' ? 'published' :
    status;
  const statusLabel = t(`skills.status${statusKeySuffix.charAt(0).toUpperCase()}${statusKeySuffix.slice(1)}`);

  return (
    <motion.div
      whileHover={{ y: -1 }}
      transition={{ duration: 0.15 }}
      onClick={onClick}
      className="cursor-pointer"
    >
      <Card className="hover:shadow-sm hover:border-violet-500/30 transition-all border-border/50 group">
        <CardContent className="p-3 flex items-center gap-3">
          <div className="w-8 h-8 rounded-md bg-gradient-to-br from-violet-500/15 to-fuchsia-500/15 flex items-center justify-center text-[10px] font-bold text-violet-600 dark:text-violet-300 shrink-0">
            {(skill.displayName || skill.name).slice(0, 2).toUpperCase()}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5">
              <span className="text-xs font-medium truncate group-hover:text-violet-600 transition-colors">
                {skill.displayName || skill.name}
              </span>
              <Badge variant="secondary" className={`text-[9px] px-1 h-3.5 shrink-0 ${getStatusColor(status)}`}>
                {statusLabel}
              </Badge>
            </div>
            <div className="text-[10px] text-muted-foreground truncate font-mono">
              {skill.name}
              {skill.labels?.stable && <span className="ml-2">· v{skill.labels.stable}</span>}
            </div>
          </div>
          <ChevronRight className="h-3 w-3 text-muted-foreground/50 group-hover:text-violet-500 transition-colors shrink-0" />
        </CardContent>
      </Card>
    </motion.div>
  );
}
