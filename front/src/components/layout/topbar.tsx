'use client';

import { Bell, Menu, ChevronRight, FileCode2, Search, Command } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useT } from '@/lib/i18n';
import { LanguageToggle } from './language-toggle';
import type { Tab } from '@/lib/api/types';

interface TopbarProps {
  activeTab: Tab;
  onMenuClick: () => void;
  accessSpaceId: string;
  unreadNotifications: number;
  editorMode?: boolean;
  onExitEditor?: () => void;
}

export function Topbar({ activeTab, onMenuClick, accessSpaceId, unreadNotifications, editorMode, onExitEditor }: TopbarProps) {
  const t = useT();

  // Title/hint per tab, looked up via i18n
  const titleKey = `${activeTab}.title`;
  const hintKey = `${activeTab}.hint`;
  const tabLabel = t(titleKey);
  const tabHint = t(hintKey);

  return (
    <header className="flex items-center justify-between h-12 px-3 md:px-4 border-b bg-card/80 backdrop-blur-sm shrink-0 gap-3">
      <div className="flex items-center gap-2.5 min-w-0">
        <Button variant="ghost" size="icon" className="md:hidden h-8 w-8 shrink-0" onClick={onMenuClick}>
          <Menu className="h-4 w-4" />
        </Button>

        {/* Breadcrumbs */}
        <nav className="flex items-center gap-1.5 min-w-0">
          {editorMode ? (
            <>
              <button
                onClick={onExitEditor}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors shrink-0"
              >
                {tabLabel}
              </button>
              <ChevronRight className="h-3 w-3 text-muted-foreground/50 shrink-0" />
              <div className="flex items-center gap-1.5 min-w-0">
                <FileCode2 className="h-3.5 w-3.5 text-violet-500 shrink-0" />
                <span className="text-xs font-medium truncate">{t('topbar.editor')}</span>
              </div>
            </>
          ) : (
            <div className="min-w-0">
              <h1 className="font-semibold text-sm leading-tight truncate">{tabLabel}</h1>
              <p className="text-[10px] text-muted-foreground leading-tight truncate">{tabHint}</p>
            </div>
          )}
        </nav>
      </div>

      <div className="flex items-center gap-1.5 shrink-0">
        {/* Search hint (placeholder for future command palette) */}
        <button
          className="hidden md:flex items-center gap-2 h-7 px-2.5 rounded-md border bg-muted/30 hover:bg-muted/50 transition-colors text-xs text-muted-foreground"
          onClick={() => {/* placeholder for command palette */}}
          title={`${t('topbar.search')} (⌘K)`}
        >
          <Search className="h-3 w-3" />
          <span>{t('topbar.search')}…</span>
          <kbd className="ml-1 flex items-center gap-0.5 text-[10px] bg-background px-1 py-0.5 rounded border">
            <Command className="h-2.5 w-2.5" />K
          </kbd>
        </button>

        {/* Language toggle */}
        <LanguageToggle />

        <Badge variant="outline" className="text-[10px] gap-1 hidden sm:inline-flex">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse-soft" />
          {accessSpaceId}
        </Badge>

        {unreadNotifications > 0 && (
          <Button variant="ghost" size="icon" className="h-7 w-7 relative">
            <Bell className="h-3.5 w-3.5" />
            <span className="absolute -top-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-violet-600 text-[8px] text-white flex items-center justify-center font-bold">
              {unreadNotifications > 9 ? '9+' : unreadNotifications}
            </span>
          </Button>
        )}
      </div>
    </header>
  );
}
