'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { proposalApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { Proposal } from '@/lib/api/types';

export function useProposals(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['proposals', 'list', params],
    queryFn: async () => {
      const page = await proposalApi.list(params);
      return asItems<Proposal>(page);
    },
    staleTime: 15_000,
  });
}

export function useProposalDetail(id: string | null) {
  return useQuery({
    queryKey: ['proposals', 'detail', id],
    queryFn: () => proposalApi.detail(id!),
    enabled: Boolean(id),
    staleTime: 10_000,
  });
}

export function useProposalValidate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => proposalApi.validate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proposals'] });
    },
  });
}

export function useProposalApprove() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, options }: { id: string; options: Record<string, unknown> }) =>
      proposalApi.approve(id, options),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proposals'] });
    },
  });
}

export function useProposalReject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      proposalApi.reject(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proposals'] });
    },
  });
}
