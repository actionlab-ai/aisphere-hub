'use client';

import { Cpu, Layers, Shield, ClipboardCheck } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export function DocsPage() {
  return (
    <div className="p-4 md:p-6 max-w-4xl mx-auto space-y-6">
      <div className="space-y-2">
        <h2 className="text-xl font-semibold">SkillHub Console Documentation</h2>
        <p className="text-sm text-muted-foreground">
          The SkillHub Console is a management interface for Agent Skill Registry, supporting skill lifecycle management,
          skillset orchestration, Agent Proposal governance, and access control.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="border-border/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2"><Cpu className="h-4 w-4 text-violet-500" /> Skills</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
              <li>Search, upload (ZIP), and browse skills</li>
              <li>Version management: submit, publish, online/offline</li>
              <li>Create drafts and force publish</li>
              <li>File browser with markdown preview</li>
              <li>Version comparison (diff view)</li>
              <li>Labels, bizTags, metadata, and scope management</li>
              <li>Social features: star, rating, subscribe</li>
            </ul>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2"><Layers className="h-4 w-4 text-fuchsia-500" /> SkillSets</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
              <li>Create and manage skill groups</li>
              <li>Bind/unbind skills with version, label, order</li>
              <li>SkillSet detail with members tab and manifest viewer</li>
              <li>SkillSet-level settings and scope control</li>
              <li>Public/private skillset scoping</li>
            </ul>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2"><Shield className="h-4 w-4 text-amber-500" /> Access / IAM</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
              <li>Namespace-based access control (RBAC)</li>
              <li>Member management with roles</li>
              <li>Local account and Agent account management</li>
              <li>API Key / Token lifecycle</li>
            </ul>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2"><ClipboardCheck className="h-4 w-4 text-emerald-500" /> Governance</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
              <li>Agent Proposal review workflow</li>
              <li>Validate, approve, reject proposals</li>
              <li>Delta and evidence inspection</li>
              <li>Gray release support</li>
            </ul>
          </CardContent>
        </Card>
      </div>

      <Card className="border-border/50">
        <CardHeader className="pb-2"><CardTitle className="text-sm">Key API Endpoints</CardTitle></CardHeader>
        <CardContent>
          <pre className="text-[11px] font-mono bg-muted/50 p-3 rounded-lg overflow-auto whitespace-pre-wrap">
{`# Canonical SkillHub API
GET  /v3/aihub/skills                    List skills
POST /v3/aihub/skills/upload             Upload skill ZIP
POST /v3/aihub/skills/draft              Create draft
GET  /v3/aihub/skill/:name               Get skill detail
DELETE /v3/aihub/skill/:name             Delete skill
POST /v3/aihub/skill/:name/publish       Publish version
POST /v3/aihub/skill/:name/online        Bring online
POST /v3/aihub/skill/:name/offline       Take offline
PUT  /v3/aihub/skill/:name/scope         Update scope
PUT  /v3/aihub/skill/:name/biz-tags      Update biz tags

# SkillSets
GET  /v3/aihub/skillsets                    List skillsets
POST /v3/aihub/skillsets                    Create skillset
GET  /v3/aihub/skillset/:name              Get skillset
PUT  /v3/aihub/skillset/:name              Update skillset
DELETE /v3/aihub/skillset/:name            Delete skillset
POST /v3/aihub/skillset/:name/skills       Bind skill
DELETE /v3/aihub/skillset/:name/skills/:skillName  Unbind

# Governance
GET  /v3/admin/ai/skill-proposals/list      List proposals
POST /v3/admin/ai/skill-proposals/:id/validate  Validate
POST /v3/admin/ai/skill-proposals/:id/approve   Approve
POST /v3/admin/ai/skill-proposals/:id/reject    Reject

# Auth
GET  /v3/auth/setup/status                  Check setup status
POST /v3/auth/setup                         Initial setup
POST /v3/auth/login                         Login

# Runtime (Agent)
GET  /v3/client/ai/skills/:name             Client skill resolution
GET  /registry-global/.well-known/agent-skills/index.json  Discovery`}
          </pre>
        </CardContent>
      </Card>

      <Card className="border-border/50">
        <CardHeader className="pb-2"><CardTitle className="text-sm">Deployment Notes</CardTitle></CardHeader>
        <CardContent>
          <p className="text-xs text-muted-foreground">
            Build the frontend and place output in <code className="bg-muted px-1 rounded">web/aihub-console/dist</code>.
            Configure <code className="bg-muted px-1 rounded">web.enabled=true</code> in the backend config to serve via <code className="bg-muted px-1 rounded">/ui/</code>.
            For external OIDC mode, the upper-layer gateway can handle login and pass the Bearer Token to SkillHub.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
