'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { AgentListItem, AgentResponse, AgentRuntimeSnapshot, AgentUpsertRequest } from '@/lib/api/types';

export function useAgents(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['agents', 'list', params],
    queryFn: async () => asItems<AgentListItem>(await agentApi.list(params)),
    staleTime: 15_000,
  });
}

export function useAgentDetail(agentId: string | null) {
  return useQuery<AgentResponse>({
    queryKey: ['agents', 'detail', agentId],
    queryFn: () => agentApi.detail(agentId!),
    enabled: Boolean(agentId),
    staleTime: 10_000,
  });
}

export function useAgentSave() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AgentUpsertRequest) => agentApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['agents', 'list'] }),
  });
}

export function useAgentUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ agentId, data }: { agentId: string; data: AgentUpsertRequest }) => agentApi.update(agentId, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['agents', 'detail', vars.agentId] });
      queryClient.invalidateQueries({ queryKey: ['agents', 'list'] });
    },
  });
}

export function useAgentDelete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (agentId: string) => agentApi.remove(agentId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['agents', 'list'] }),
  });
}

export function useAgentResolve() {
  return useMutation<AgentRuntimeSnapshot, Error, { agentId: string; runtimeId?: string; sessionId?: string; version?: string; label?: string; policy?: string }>({
    mutationFn: ({ agentId, ...body }) => agentApi.resolve(agentId, body),
  });
}
