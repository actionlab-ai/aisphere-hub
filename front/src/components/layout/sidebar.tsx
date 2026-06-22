'use client';

import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Cpu, Layers, Bot, Hammer, Boxes, Shield, ClipboardCheck, Activity, FileText, Users, BookOpen,
  ChevronDown, ChevronRight, LogOut, Moon, Sun, X, Sparkles, CircleDot,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Badge } from '@/components/ui/badge';
import { useTheme } from 'next-themes';
import { cn } from '@/lib/utils';
import { useT } from '@/lib/i18n';
import type { Tab } from '@/lib/api/types';

// Tab → icon + i18n keys
const navSections: { titleKey: string; items: { key: Tab; labelKey: string; hintKey: string; icon: React.ReactNode }[] }[] = [
  {
    titleKey: 'nav.registry',
    items: [
      { key: 'skills', labelKey: 'nav.skills', hintKey: 'nav.skills.hint', icon: <Cpu className="h-4 w-4" /> },
      { key: 'groups', labelKey: 'nav.groups', hintKey: 'nav.groups.hint', icon: <Layers className="h-4 w-4" /> },
      { key: 'agents', labelKey: 'nav.agents', hintKey: 'nav.agents.hint', icon: <Bot className="h-4 w-4" /> },
      { key: 'tools', labelKey: 'nav.tools', hintKey: 'nav.tools.hint', icon: <Hammer className="h-4 w-4" /> },
      { key: 'model-profiles', labelKey: 'nav.modelProfiles', hintKey: 'nav.modelProfiles.hint', icon: <CircleDot className="h-4 w-4" /> },
      { key: 'sandbox-profiles', labelKey: 'nav.sandboxProfiles', hintKey: 'nav.sandboxProfiles.hint', icon: <Boxes className="h-4 w-4" /> },
      { key: 'sandboxes', labelKey: 'nav.sandboxes', hintKey: 'nav.sandboxes.hint', icon: <Boxes className="h-4 w-4" /> },
    ],
  },
  {
    titleKey: 'nav.governance',
    items: [
      { key: 'proposals', labelKey: 'nav.proposals', hintKey: 'nav.proposals.hint', icon: <FileText className="h-4 w-4" /> },
      { key: 'governance', labelKey: 'nav.governance', hintKey: 'nav.governance.hint', icon: <ClipboardCheck className="h-4 w-4" /> },
    ],
  },
  {
    titleKey: 'nav.access',
    items: [
      { key: 'access', labelKey: 'nav.access', hintKey: 'nav.access.hint', icon: <Shield className="h-4 w-4" /> },
      { key: 'iam', labelKey: 'nav.iam', hintKey: 'nav.iam.hint', icon: <Users className="h-4 w-4" /> },
    ],
  },
  {
    titleKey: 'nav.operations',
    items: [
      { key: 'ops', labelKey: 'nav.ops', hintKey: 'nav.ops.hint', icon: <Activity className="h-4 w-4" /> },
      { key: 'docs', labelKey: 'nav.docs', hintKey: 'nav.docs.hint', icon: <BookOpen className="h-4 w-4" /> },
    ],
  },
];

interface SidebarProps {
  activeTab: Tab;
  onTabChange: (tab: Tab) => void;
  collapsed: boolean;
  onToggleCollapse: () => void;
  principal: Record<string, unknown> | null;
  onLogout: () => void;
}

function isAccessAdmin(principal: Record<string, unknown> | null): boolean {
  const roles = [
    ...(((principal?.roles as string[]) || [])),
    ...(((principal?.externalRoles as string[]) || [])),
    ...(((principal?.permissions as string[]) || [])),
  ].map((x) => String(x).toLowerCase());
  return roles.some((r) => ['admin', 'role:admin', 'skill-admin', 'aihub-admin', 'platform-admin', '*', 'access:admin:read'].includes(r));
}

function visibleSections(sections: typeof navSections, principal: Record<string, unknown> | null) {
  const accessAdmin = isAccessAdmin(principal);
  return sections
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => item.key !== 'access' || accessAdmin),
    }))
    .filter((section) => section.items.length > 0);
}

function getInitials(name: string): string {
  return name.slice(0, 2).toUpperCase();
}

function getRoleLabel(principal: Record<string, unknown> | null, t: (k: string) => string): string {
  const roles = (principal?.roles as string[]) || [];
  if (roles.includes('admin') || roles.includes('role:admin')) return 'admin';
  if (roles.includes('skill-admin')) return 'skill admin';
  if (roles.length > 0) return roles[0];
  return 'member';
}

export function Sidebar({
  activeTab,
  onTabChange,
  collapsed,
  onToggleCollapse,
  principal,
  onLogout,
}: SidebarProps) {
  const { theme, setTheme } = useTheme();
  const t = useT();
  const username = (principal?.subjectId || principal?.username || 'user') as string;
  const initials = getInitials(username);
  const role = getRoleLabel(principal, t);

  return (
    <motion.aside
      className="hidden md:flex flex-col border-r bg-sidebar h-full shrink-0"
      animate={{ width: collapsed ? 56 : 240 }}
      transition={{ duration: 0.2, ease: 'easeInOut' }}
    >
      {/* Brand */}
      <div className="flex items-center gap-2.5 px-3 h-12 border-b shrink-0">
        <div className="flex items-center justify-center w-7 h-7 rounded-lg bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white shrink-0 shadow-sm shadow-violet-500/30">
          <Sparkles className="h-3.5 w-3.5" />
        </div>
        {!collapsed && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="overflow-hidden">
            <div className="font-semibold text-sm tracking-tight leading-tight">{t('app.name')}</div>
            <div className="text-[10px] text-muted-foreground leading-tight">{t('app.subtitle')}</div>
          </motion.div>
        )}
      </div>

      {/* Navigation */}
      <ScrollArea className="flex-1 py-2 scrollbar-thin">
        <nav className="flex flex-col gap-0.5 px-2">
          {visibleSections(navSections, principal).map((section) => (
            <div key={section.titleKey} className="mb-1">
              {!collapsed && (
                <div className="px-2.5 py-1 text-[10px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                  {t(section.titleKey)}
                </div>
              )}
              {collapsed && <div className="my-1.5 mx-2"><Separator /></div>}
              {section.items.map((item) => {
                const isActive = activeTab === item.key;
                return (
                  <Tooltip key={item.key} delayDuration={collapsed ? 0 : 999}>
                    <TooltipTrigger asChild>
                      <button
                        onClick={() => onTabChange(item.key)}
                        className={cn(
                          'relative flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 w-full text-left group',
                          isActive
                            ? 'text-foreground font-medium bg-accent'
                            : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground',
                        )}
                      >
                        {isActive && (
                          <motion.div
                            layoutId="sidebar-active-indicator"
                            className="absolute left-0 top-1 bottom-1 w-[3px] rounded-full bg-gradient-to-b from-violet-500 to-fuchsia-500"
                            transition={{ type: 'spring', stiffness: 500, damping: 35 }}
                          />
                        )}
                        <span className={cn('shrink-0', isActive && 'text-violet-600 dark:text-violet-400')}>
                          {item.icon}
                        </span>
                        {!collapsed && (
                          <span className="truncate text-[13px]">{t(item.labelKey)}</span>
                        )}
                        {!collapsed && isActive && (
                          <ChevronRight className="h-3 w-3 ml-auto text-violet-500 shrink-0" />
                        )}
                      </button>
                    </TooltipTrigger>
                    {collapsed && (
                      <TooltipContent side="right" className="flex flex-col gap-0.5">
                        <p className="font-medium">{t(item.labelKey)}</p>
                        <p className="text-xs text-muted-foreground">{t(item.hintKey)}</p>
                      </TooltipContent>
                    )}
                  </Tooltip>
                );
              })}
            </div>
          ))}
        </nav>
      </ScrollArea>

      {/* Status footer */}
      {!collapsed && (
        <div className="px-3 pb-1.5">
          <div className="rounded-md border bg-card/50 px-2.5 py-1.5 flex items-center gap-1.5">
            <CircleDot className="h-2.5 w-2.5 text-emerald-500 animate-pulse-soft shrink-0" />
            <span className="text-[10px] text-muted-foreground">{t('app.systemOk')}</span>
            <span className="ml-auto text-[10px] text-muted-foreground/70 tabular-nums">{t('app.version')}</span>
          </div>
        </div>
      )}

      {/* User profile section */}
      <div className="border-t px-2 py-2">
        <div className="flex items-center gap-2 px-1.5 py-1">
          <div className="w-7 h-7 rounded-full bg-gradient-to-br from-violet-500/20 to-fuchsia-500/20 flex items-center justify-center text-[10px] font-bold text-violet-600 dark:text-violet-400 shrink-0">
            {initials}
          </div>
          {!collapsed && (
            <div className="flex-1 min-w-0">
              <div className="text-xs font-medium truncate">{username}</div>
              <Badge variant="secondary" className="text-[9px] px-1 py-0 h-3.5 mt-0.5 capitalize">{role}</Badge>
            </div>
          )}
          {!collapsed && (
            <div className="flex items-center gap-0.5">
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>
                {theme === 'dark' ? <Sun className="h-3 w-3" /> : <Moon className="h-3 w-3" />}
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground" onClick={onLogout}>
                <LogOut className="h-3 w-3" />
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Collapse toggle */}
      <button
        onClick={onToggleCollapse}
        className="flex items-center justify-center h-8 border-t text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
      >
        {collapsed ? <ChevronDown className="h-3.5 w-3.5 rotate-180" /> : <ChevronDown className="h-3.5 w-3.5" />}
      </button>
    </motion.aside>
  );
}

// ─── Mobile Sidebar Overlay ──────────────────────────────────────────────────
interface MobileSidebarProps {
  open: boolean;
  onClose: () => void;
  activeTab: Tab;
  onTabChange: (tab: Tab) => void;
  principal: Record<string, unknown> | null;
  onLogout: () => void;
}

export function MobileSidebar({ open, onClose, activeTab, onTabChange, principal, onLogout }: MobileSidebarProps) {
  const { theme, setTheme } = useTheme();
  const t = useT();
  const username = (principal?.subjectId || principal?.username || 'user') as string;
  const role = getRoleLabel(principal, t);

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="fixed inset-0 z-50 md:hidden"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        >
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />
          <motion.div
            className="absolute left-0 top-0 bottom-0 w-64 bg-sidebar border-r flex flex-col"
            initial={{ x: -260 }}
            animate={{ x: 0 }}
            exit={{ x: -260 }}
            transition={{ duration: 0.2 }}
          >
            {/* Header */}
            <div className="flex items-center justify-between px-4 h-12 border-b shrink-0">
              <div className="flex items-center gap-2.5">
                <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white flex items-center justify-center shadow-sm shadow-violet-500/30">
                  <Sparkles className="h-3.5 w-3.5" />
                </div>
                <div>
                  <div className="font-semibold text-sm">{t('app.name')}</div>
                  <div className="text-[10px] text-muted-foreground">{t('app.subtitle')}</div>
                </div>
              </div>
              <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
                <X className="h-4 w-4" />
              </Button>
            </div>

            {/* Nav */}
            <ScrollArea className="flex-1 py-2 px-2 scrollbar-thin">
              {visibleSections(navSections, principal).map((section) => (
                <div key={section.titleKey} className="mb-2">
                  <div className="px-2 py-1 text-[10px] font-semibold text-muted-foreground/70 uppercase tracking-wider">
                    {t(section.titleKey)}
                  </div>
                  {section.items.map((item) => (
                    <button
                      key={item.key}
                      onClick={() => { onTabChange(item.key); onClose(); }}
                      className={cn(
                        'flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm w-full text-left transition-colors',
                        activeTab === item.key
                          ? 'bg-violet-600/10 text-foreground font-medium'
                          : 'text-muted-foreground hover:bg-accent/60',
                      )}
                    >
                      {item.icon}
                      {t(item.labelKey)}
                    </button>
                  ))}
                </div>
              ))}
            </ScrollArea>

            {/* Footer */}
            <div className="border-t px-3 py-2 space-y-1.5 shrink-0">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <div className="w-6 h-6 rounded-full bg-gradient-to-br from-violet-500/20 to-fuchsia-500/20 flex items-center justify-center text-[9px] font-bold text-violet-600">
                  {getInitials(username)}
                </div>
                <div className="min-w-0">
                  <div className="font-medium text-foreground truncate">{username}</div>
                  <div className="text-[10px] capitalize">{role}</div>
                </div>
              </div>
              <div className="flex gap-1">
                <Button variant="ghost" size="sm" className="h-7 text-xs flex-1" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>
                  {theme === 'dark' ? <Sun className="h-3 w-3 mr-1" /> : <Moon className="h-3 w-3 mr-1" />}
                  {theme === 'dark' ? 'Light' : 'Dark'}
                </Button>
                <Button variant="ghost" size="sm" className="h-7 text-xs flex-1 text-destructive" onClick={onLogout}>
                  <LogOut className="h-3 w-3 mr-1" /> Logout
                </Button>
              </div>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
