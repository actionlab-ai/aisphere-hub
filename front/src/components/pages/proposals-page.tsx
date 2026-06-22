'use client';

import { useState } from 'react';
import { CheckCircle2, XCircle, Eye, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useProposals, useProposalValidate, useProposalApprove, useProposalReject } from '@/hooks/use-proposals';
import { getStatusColor } from '@/lib/utils';
import { useT } from '@/lib/i18n';
import { toast } from 'sonner';
import type { Proposal } from '@/lib/api/types';

export function ProposalsPage() {
  const t = useT();
  const [status, setStatus] = useState('');
  const [selected, setSelected] = useState<Proposal | null>(null);
  const [approveLabel, setApproveLabel] = useState('gray');
  const [rejectReason, setRejectReason] = useState('');
  const [showRejectForm, setShowRejectForm] = useState(false);

  const { data: items = [], isLoading, refetch } = useProposals({
    status: status === 'all' ? undefined : status || undefined,
    pageNo: 1,
    pageSize: 50,
  });

  const validateMutation = useProposalValidate();
  const approveMutation = useProposalApprove();
  const rejectMutation = useProposalReject();

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="flex items-center gap-2">
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-36 h-8 text-xs"><SelectValue placeholder={t('common.all')} /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('common.all')}</SelectItem>
            <SelectItem value="submitted">Submitted</SelectItem>
            <SelectItem value="validated">Validated</SelectItem>
            <SelectItem value="approved">Approved</SelectItem>
            <SelectItem value="rejected">Rejected</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={() => refetch()}><RefreshCw className="h-3.5 w-3.5 mr-1" /> {t('skills.refresh')}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-xs">{t('ns.id')}</TableHead>
                <TableHead className="text-xs">Skill</TableHead>
                <TableHead className="text-xs">Base</TableHead>
                <TableHead className="text-xs">Candidate</TableHead>
                <TableHead className="text-xs">{t('common.status')}</TableHead>
                <TableHead className="text-xs">{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}><TableCell colSpan={6}><Skeleton className="h-4 w-full" /></TableCell></TableRow>
              )) : items.length === 0 ? (
                <TableRow><TableCell colSpan={6} className="text-center text-xs text-muted-foreground py-6">{t('proposals.empty')}</TableCell></TableRow>
              ) : items.map((p) => (
                <TableRow key={p.proposalId}>
                  <TableCell className="font-mono text-xs">{p.proposalId.slice(0, 8)}</TableCell>
                  <TableCell className="text-xs">{p.skillName}</TableCell>
                  <TableCell className="font-mono text-xs">{p.baseVersion}</TableCell>
                  <TableCell className="font-mono text-xs">{p.candidateVersion}</TableCell>
                  <TableCell><Badge variant="secondary" className={`text-[10px] ${getStatusColor(p.status)}`}>{p.status}</Badge></TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => setSelected(p)}><Eye className="h-3 w-3 mr-0.5" />View</Button>
                      <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => validateMutation.mutateAsync(p.proposalId).then(() => { refetch(); toast.success('Validated'); }).catch((e) => toast.error(e.message))}>Validate</Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Sheet open={Boolean(selected)} onOpenChange={(open) => { if (!open) { setSelected(null); setShowRejectForm(false); } }}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          {selected && (
            <>
              <SheetHeader>
                <SheetTitle className="text-sm">Proposal: {selected.proposalId.slice(0, 12)}</SheetTitle>
              </SheetHeader>
              <ScrollArea className="h-[calc(100%-60px)] mt-4">
                <div className="space-y-4">
                  <div className="space-y-2">
                    <p className="text-sm">{selected.reason}</p>
                    <pre className="text-[10px] font-mono bg-muted p-2 rounded overflow-auto max-h-40">{JSON.stringify(selected, null, 2)}</pre>
                  </div>
                  <div className="space-y-2">
                    <Label className="text-xs font-medium">Approve Label</Label>
                    <Input value={approveLabel} onChange={(e) => setApproveLabel(e.target.value)} className="h-8 text-xs" />
                  </div>
                  {showRejectForm && (
                    <div className="space-y-2">
                      <Label className="text-xs font-medium">{t('proposals.rejectReason')}</Label>
                      <Textarea
                        value={rejectReason}
                        onChange={(e) => setRejectReason(e.target.value)}
                        placeholder={t('proposals.rejectReasonPlaceholder')}
                        rows={3}
                        className="text-xs"
                      />
                    </div>
                  )}
                  <div className="flex gap-2">
                    <Button size="sm" className="bg-emerald-600 hover:bg-emerald-700" onClick={async () => {
                      try {
                        await approveMutation.mutateAsync({ id: selected.proposalId, options: { publish: true, online: true, label: approveLabel } });
                        toast.success(t('proposals.approved'));
                        setSelected(null);
                        refetch();
                      } catch (e: unknown) { toast.error(e instanceof Error ? e.message : t('proposals.actionFailed')); }
                    }}><CheckCircle2 className="h-3 w-3 mr-1" /> {t('proposals.approve')} → {approveLabel}</Button>
                    <Button size="sm" variant="destructive" onClick={async () => {
                      if (!showRejectForm) {
                        setShowRejectForm(true);
                        return;
                      }
                      const reason = rejectReason.trim() || 'Rejected';
                      try {
                        await rejectMutation.mutateAsync({ id: selected.proposalId, reason });
                        toast.success(t('proposals.rejected'));
                        setSelected(null);
                        setShowRejectForm(false);
                        setRejectReason('');
                        refetch();
                      } catch (e: unknown) { toast.error(e instanceof Error ? e.message : t('proposals.actionFailed')); }
                    }}><XCircle className="h-3 w-3 mr-1" /> {t('proposals.reject')}</Button>
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
