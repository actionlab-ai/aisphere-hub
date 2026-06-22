'use client';

import { useState } from 'react';
import { ClipboardCheck, FileText, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { StatCard, CardGridSkeleton } from '@/components/shared';
import { useProposals, useProposalValidate, useProposalApprove, useProposalReject } from '@/hooks/use-proposals';
import { getStatusColor } from '@/lib/utils';
import { toast } from 'sonner';
import type { Proposal } from '@/lib/api/types';

export function GovernancePage() {
  const [status, setStatus] = useState('');
  const [selected, setSelected] = useState<Proposal | null>(null);

  const { data: items = [], isLoading, refetch } = useProposals({
    status: status || undefined,
    pageNo: 1,
    pageSize: 100,
  });

  const validateMutation = useProposalValidate();
  const approveMutation = useProposalApprove();
  const rejectMutation = useProposalReject();

  const doAction = async (id: string, action: string) => {
    try {
      switch (action) {
        case 'validate':
          await validateMutation.mutateAsync(id);
          toast.success('Validated');
          break;
        case 'approve':
          await approveMutation.mutateAsync({ id, options: { publish: true, online: true, label: 'gray' } });
          toast.success('Approved to gray');
          break;
        case 'reject':
          await rejectMutation.mutateAsync({ id, reason: 'Rejected from governance inbox' });
          toast.success('Rejected');
          break;
      }
      refetch();
      if (selected?.proposalId === id) setSelected(null);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : 'Action failed');
    }
  };

  const openCount = items.filter((x) => !['approved', 'rejected', 'promoted'].includes(x.status || '')).length;

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
        <StatCard icon={<ClipboardCheck className="h-4 w-4" />} label="Open" value={openCount} />
        <StatCard icon={<FileText className="h-4 w-4" />} label="Total" value={items.length} />
        <StatCard icon={<CheckCircle2 className="h-4 w-4" />} label="Approved" value={items.filter((x) => x.status === 'approved').length} />
      </div>

      <div className="flex items-center gap-2">
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-36 h-8 text-xs"><SelectValue placeholder="All Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="submitted">Submitted</SelectItem>
            <SelectItem value="validating">Validating</SelectItem>
            <SelectItem value="approved">Approved</SelectItem>
            <SelectItem value="rejected">Rejected</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={() => refetch()}>Refresh</Button>
      </div>

      {isLoading ? (
        <CardGridSkeleton count={6} />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {items.map((p) => (
            <Card key={p.proposalId} className="border-border/50 hover:shadow-sm transition-shadow cursor-pointer" onClick={() => setSelected(p)}>
              <CardContent className="p-4 space-y-2">
                <div className="flex items-start justify-between">
                  <h3 className="font-medium text-sm">{p.skillName}</h3>
                  <Badge variant="secondary" className={`text-[10px] ${getStatusColor(p.status)}`}>{p.status}</Badge>
                </div>
                <p className="text-xs text-muted-foreground line-clamp-2">{p.reason || 'No reason provided'}</p>
                <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
                  <span>v{p.baseVersion || '-'}</span>
                  <span>→</span>
                  <span>v{p.candidateVersion || '-'}</span>
                </div>
                <div className="flex gap-1.5">
                  <Button variant="outline" size="sm" className="h-7 text-[10px]" onClick={(e) => { e.stopPropagation(); doAction(p.proposalId, 'validate'); }}>Validate</Button>
                  <Button size="sm" className="h-7 text-[10px] bg-emerald-600 hover:bg-emerald-700 text-white" onClick={(e) => { e.stopPropagation(); doAction(p.proposalId, 'approve'); }}>Approve</Button>
                  <Button variant="outline" size="sm" className="h-7 text-[10px] text-destructive" onClick={(e) => { e.stopPropagation(); doAction(p.proposalId, 'reject'); }}>Reject</Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Sheet open={Boolean(selected)} onOpenChange={(open) => { if (!open) setSelected(null); }}>
        <SheetContent side="right" className="w-full sm:max-w-xl p-0">
          {selected && (
            <>
              <SheetHeader className="p-4 pb-0">
                <SheetTitle className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-amber-500/10 to-orange-500/10 flex items-center justify-center">
                    <ClipboardCheck className="h-5 w-5 text-amber-600" />
                  </div>
                  <div>
                    <div>{selected.skillName}</div>
                    <Badge variant="secondary" className={`text-[10px] ${getStatusColor(selected.status)}`}>{selected.status}</Badge>
                  </div>
                </SheetTitle>
              </SheetHeader>
              <ScrollArea className="h-[calc(100%-80px)] p-4">
                <div className="space-y-4">
                  <p className="text-sm text-muted-foreground">{selected.reason || 'No reason'}</p>
                  <div className="flex items-center gap-3 text-xs">
                    <span>Base: <code className="font-mono bg-muted px-1 py-0.5 rounded">v{selected.baseVersion || '-'}</code></span>
                    <span>Target: <code className="font-mono bg-muted px-1 py-0.5 rounded">v{selected.candidateVersion || '-'}</code></span>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <Card className="border-border/50">
                      <CardContent className="p-3">
                        <h4 className="text-xs font-medium mb-1">Delta</h4>
                        <pre className="text-[10px] font-mono overflow-auto max-h-60 whitespace-pre-wrap bg-muted/50 p-2 rounded">
                          {JSON.stringify(selected.delta, null, 2)}
                        </pre>
                      </CardContent>
                    </Card>
                    <Card className="border-border/50">
                      <CardContent className="p-3">
                        <h4 className="text-xs font-medium mb-1">Evidence</h4>
                        <pre className="text-[10px] font-mono overflow-auto max-h-60 whitespace-pre-wrap bg-muted/50 p-2 rounded">
                          {JSON.stringify(selected.evidence, null, 2)}
                        </pre>
                      </CardContent>
                    </Card>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" className="bg-emerald-600 hover:bg-emerald-700" onClick={() => doAction(selected.proposalId, 'approve')}>
                      <CheckCircle2 className="h-3 w-3 mr-1" /> Approve to Gray
                    </Button>
                    <Button size="sm" variant="destructive" onClick={() => doAction(selected.proposalId, 'reject')}>
                      <XCircle className="h-3 w-3 mr-1" /> Reject
                    </Button>
                  </div>
                </div>
              </ScrollArea>
            </>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}
