'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppShell } from '@/components/layout/app-shell';
import {
  SkillsPage,
  GroupsPage,
  AgentsPage,
  ToolsPage,
  ModelProfilesPage,
  SandboxProfilesPage,
  SandboxesPage,
  NamespacesPage,
  GovernancePage,
  ProposalsPage,
  IamPage,
  AccessPage,
  OpsPage,
  DocsPage,
} from '@/components/pages';
import type { Tab } from '@/lib/api/types';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function PageRouter({ tab }: { tab: Tab }) {
  switch (tab) {
    case 'skills': return <SkillsPage />;
    case 'groups': return <GroupsPage />;
    case 'agents': return <AgentsPage />;
    case 'tools': return <ToolsPage />;
    case 'model-profiles': return <ModelProfilesPage />;
    case 'sandbox-profiles': return <SandboxProfilesPage />;
    case 'sandboxes': return <SandboxesPage />;
    case 'namespaces': return <NamespacesPage />;
    case 'governance': return <GovernancePage />;
    case 'ops': return <OpsPage />;
    case 'proposals': return <ProposalsPage />;
    case 'iam': return <IamPage />;
    case 'access': return <AccessPage />;
    case 'docs': return <DocsPage />;
    default: return <SkillsPage />;
  }
}

export default function SkillHubConsole() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppShell>
        {(tab) => <PageRouter tab={tab} />}
      </AppShell>
    </QueryClientProvider>
  );
}
