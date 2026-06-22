'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Loader2 } from 'lucide-react';
import { TooltipProvider } from '@/components/ui/tooltip';
import { Sidebar, MobileSidebar } from './sidebar';
import { Topbar } from './topbar';
import { authApi } from '@/lib/api';
import { setTokens, clearTokens, getToken, getAccessSpace, asItems } from '@/lib/api/client';
import { useNotifications } from '@/hooks/use-ops';
import { LoginPage } from '@/components/auth/login-page';
import { SetupPage } from '@/components/auth/setup-page';
import { SkillEditor } from '@/components/editor/skill-editor';
import { useT } from '@/lib/i18n';
import type { Tab, Notification } from '@/lib/api/types';

interface AppShellProps {
  children: (tab: Tab) => React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const t = useT();
  const [setupRequired, setSetupRequired] = useState(false);
  const [ready, setReady] = useState(false);
  const [authed, setAuthed] = useState(() => {
    if (typeof window === 'undefined') return false;
    // The user is considered "authed" if EITHER a legacy local JWT
    // token is in localStorage OR the aisphere-auth integration is
    // enabled (in which case the aisphere_session cookie carries
    // identity and the backend /v3/auth/me will resolve it via the
    // aisphereauth provider). We don't read the cookie directly from
    // JS because it is HttpOnly.
    const aisphereEnabled =
      process.env.NEXT_PUBLIC_AISPHERE_AUTH_ENABLED === 'true' ||
      process.env.NEXT_PUBLIC_AISPHERE_AUTH_ENABLED === '1';
    return Boolean(getToken()) || aisphereEnabled;
  });
  const [tab, setTab] = useState<Tab>('skills');
  const [principal, setPrincipal] = useState<Record<string, unknown> | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [editingSkill, setEditingSkill] = useState<string | null>(null);
  const accessSpaceId = getAccessSpace();

  const { data: notificationsData } = useNotifications({ pageNo: 1, pageSize: 30 });

  const unreadNotifications = (notificationsData || []).filter((n: Notification) => !n.read).length;

  useEffect(() => {
    authApi.setupStatus()
      .then((s) => setSetupRequired(Boolean(s.setupRequired)))
      .catch(() => {})
      .finally(() => setReady(true));
  }, []);

  useEffect(() => {
    if (!authed) return;
    authApi.me().then((m) => setPrincipal((m as Record<string, unknown>)?.principal as Record<string, unknown> || m as Record<string, unknown> || null)).catch(() => {
      // When the aisphere-auth integration is enabled a failed /me
      // means the platform session is gone. Bounce through SkillHub
      // to aisphere-auth so it can re-establish the session cookie.
      const aisphereEnabled =
        process.env.NEXT_PUBLIC_AISPHERE_AUTH_ENABLED === 'true' ||
        process.env.NEXT_PUBLIC_AISPHERE_AUTH_ENABLED === '1';
      if (aisphereEnabled && !getToken()) {
        const here = window.location.origin + window.location.pathname;
        window.location.href = `/v3/auth/aisphere/login?redirect=${encodeURIComponent(here)}`;
        return;
      }
      setPrincipal({ subjectId: 'preview-user', username: 'preview' });
    });
  }, [authed]);

  const handleLogout = useCallback(() => {
    clearTokens();
    setAuthed(false);
    setPrincipal(null);
  }, []);

  // Open the skill editor
  const openSkillEditor = useCallback((skillName: string) => {
    setEditingSkill(skillName);
  }, []);

  // Close the skill editor
  const closeSkillEditor = useCallback(() => {
    setEditingSkill(null);
  }, []);

  // Tab change should close the editor
  const handleTabChange = useCallback((nextTab: Tab) => {
    setEditingSkill(null);
    setTab(nextTab);
  }, []);

  if (!ready) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-background gap-4">
        <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white font-bold flex items-center justify-center text-lg animate-pulse">
          SH
        </div>
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          {t('app.loading')}
        </div>
      </div>
    );
  }

  if (setupRequired) {
    return <SetupPage onDone={() => { setSetupRequired(false); setAuthed(true); }} />;
  }

  if (!authed) {
    return <LoginPage onDone={() => setAuthed(true)} />;
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen bg-background overflow-hidden">
        <Sidebar
          activeTab={tab}
          onTabChange={handleTabChange}
          collapsed={!sidebarOpen}
          onToggleCollapse={() => setSidebarOpen(!sidebarOpen)}
          principal={principal}
          onLogout={handleLogout}
        />

        <div className="flex-1 flex flex-col min-w-0">
          <Topbar
            activeTab={tab}
            onMenuClick={() => setMobileSidebarOpen(true)}
            accessSpaceId={accessSpaceId}
            unreadNotifications={unreadNotifications}
            editorMode={Boolean(editingSkill)}
            onExitEditor={closeSkillEditor}
          />

          <main className="flex-1 overflow-hidden">
            <AnimatePresence mode="wait">
              {editingSkill ? (
                <motion.div
                  key={`editor:${editingSkill}`}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.15 }}
                  className="h-full"
                >
                  <SkillEditor skillName={editingSkill} onBack={closeSkillEditor} />
                </motion.div>
              ) : (
                <motion.div
                  key={tab}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: 0.15 }}
                  className="h-full"
                >
                  {/* Pass openSkillEditor to children via context */}
                  <SkillEditorContext.Provider value={openSkillEditor}>
                    {children(tab)}
                  </SkillEditorContext.Provider>
                </motion.div>
              )}
            </AnimatePresence>
          </main>
        </div>

        <MobileSidebar
          open={mobileSidebarOpen}
          onClose={() => setMobileSidebarOpen(false)}
          activeTab={tab}
          onTabChange={handleTabChange}
          principal={principal}
          onLogout={handleLogout}
        />
      </div>
    </TooltipProvider>
  );
}

// Context to allow any child page to open the skill editor
export const SkillEditorContext = React.createContext<(skillName: string) => void>(() => {});

export function useOpenSkillEditor() {
  return React.useContext(SkillEditorContext);
}
