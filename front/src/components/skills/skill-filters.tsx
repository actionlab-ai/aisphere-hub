'use client';

import { Search, RefreshCw, Upload, Plus, Filter, LayoutGrid, Layers3 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import { useT } from '@/lib/i18n';

export type SkillViewMode = 'grid' | 'group';

interface SkillFiltersProps {
  search: string;
  onSearchChange: (v: string) => void;
  groupName: string;
  onGroupNameChange: (v: string) => void;
  statusFilter: string;
  onStatusFilterChange: (v: string) => void;
  scopeFilter: string;
  onScopeFilterChange: (v: string) => void;
  onRefresh: () => void;
  onUploadClick: () => void;
  onCreateClick: () => void;
  viewMode: SkillViewMode;
  onViewModeChange: (v: SkillViewMode) => void;
}

export function SkillFilters({
  search,
  onSearchChange,
  groupName,
  onGroupNameChange,
  statusFilter,
  onStatusFilterChange,
  scopeFilter,
  onScopeFilterChange,
  onRefresh,
  onUploadClick,
  onCreateClick,
  viewMode,
  onViewModeChange,
}: SkillFiltersProps) {
  const t = useT();
  const hasActiveFilters = statusFilter !== 'all' || scopeFilter !== 'all' || groupName || search;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {/* Search */}
      <div className="relative flex-1 min-w-[220px] max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
        <Input
          placeholder={t('skills.search')}
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-9 h-9 bg-card"
        />
        {search && (
          <button
            onClick={() => onSearchChange('')}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground hover:text-foreground"
          >
            {t('skills.clear')}
          </button>
        )}
      </div>

      {/* Status filter */}
      <Select value={statusFilter} onValueChange={onStatusFilterChange}>
        <SelectTrigger className={cn('w-32 h-9 bg-card text-xs', statusFilter !== 'all' && 'border-violet-500/40')}>
          <Filter className="h-3 w-3 mr-1" />
          <SelectValue placeholder={t('skills.statusAll')} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{t('skills.statusAll')}</SelectItem>
          <SelectItem value="online">{t('skills.statusOnline')}</SelectItem>
          <SelectItem value="offline">{t('skills.statusOffline')}</SelectItem>
          <SelectItem value="draft">{t('skills.statusDraft')}</SelectItem>
          <SelectItem value="published">{t('skills.statusPublished')}</SelectItem>
        </SelectContent>
      </Select>

      {/* Access mode filter (was "Scope") */}
      <Select value={scopeFilter} onValueChange={onScopeFilterChange}>
        <SelectTrigger className={cn('w-28 h-9 bg-card text-xs', scopeFilter !== 'all' && 'border-violet-500/40')}>
          <SelectValue placeholder={t('accessMode.label')} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{t('common.all')}</SelectItem>
          <SelectItem value="public">{t('accessMode.public')}</SelectItem>
          <SelectItem value="private">{t('accessMode.private')}</SelectItem>
        </SelectContent>
      </Select>

      {/* Group filter */}
      <Input
        placeholder={t('skills.groupFilter')}
        value={groupName}
        onChange={(e) => onGroupNameChange(e.target.value)}
        className={cn('w-28 h-9 bg-card text-xs', groupName && 'border-violet-500/40')}
      />

      {/* Reset all filters */}
      {hasActiveFilters && (
        <Button
          variant="ghost"
          size="sm"
          className="h-9 text-xs text-muted-foreground"
          onClick={() => {
            onSearchChange('');
            onGroupNameChange('');
            onStatusFilterChange('all');
            onScopeFilterChange('all');
          }}
        >
          {t('skills.reset')}
        </Button>
      )}

      <div className="flex-1" />

      {/* View mode toggle */}
      <div className="flex items-center bg-muted/40 rounded-md p-0.5 gap-0.5">
        <button
          onClick={() => onViewModeChange('grid')}
          className={cn(
            'flex items-center gap-1 px-2 h-7 rounded text-[11px] transition-colors',
            viewMode === 'grid'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
          title={t('skills.viewGrid')}
        >
          <LayoutGrid className="h-3 w-3" />
          <span className="hidden sm:inline">{t('skills.viewGrid')}</span>
        </button>
        <button
          onClick={() => onViewModeChange('group')}
          className={cn(
            'flex items-center gap-1 px-2 h-7 rounded text-[11px] transition-colors',
            viewMode === 'group'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
          title={t('skills.viewGroup')}
        >
          <Layers3 className="h-3 w-3" />
          <span className="hidden sm:inline">{t('skills.viewGroup')}</span>
        </button>
      </div>

      {/* Actions */}
      <Button variant="outline" size="sm" className="h-9 px-2" onClick={onRefresh} title={t('skills.refresh')}>
        <RefreshCw className="h-3.5 w-3.5" />
      </Button>
      <Button size="sm" variant="outline" className="h-9" onClick={onCreateClick}>
        <Plus className="h-3.5 w-3.5 mr-1.5" /> {t('skills.createDraft')}
      </Button>
      <Button
        size="sm"
        className="h-9 bg-gradient-to-r from-violet-600 to-fuchsia-500 hover:from-violet-700 hover:to-fuchsia-600 shadow-sm shadow-violet-500/20"
        onClick={onUploadClick}
      >
        <Upload className="h-3.5 w-3.5 mr-1.5" /> {t('skills.uploadZip')}
      </Button>
    </div>
  );
}
