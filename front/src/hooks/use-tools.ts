'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toolApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { ToolFailureRecord, ToolListItem, ToolResponse, ToolRuntimeSnapshot, ToolUpsertRequest } from '@/lib/api/types';

export function useTools(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['tools', 'list', params],
    queryFn: async () => asItems<ToolListItem>(await toolApi.list(params)),
    staleTime: 15_000,
  });
}

export function useToolDetail(toolId: string | null) {
  return useQuery<ToolResponse>({
    queryKey: ['tools', 'detail', toolId],
    queryFn: () => toolApi.detail(toolId!),
    enabled: Boolean(toolId),
    staleTime: 10_000,
  });
}

export function useToolSave() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ToolUpsertRequest) => toolApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tools', 'list'] }),
  });
}

export function useToolUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ toolId, data }: { toolId: string; data: ToolUpsertRequest }) => toolApi.update(toolId, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['tools', 'detail', vars.toolId] });
      queryClient.invalidateQueries({ queryKey: ['tools', 'list'] });
    },
  });
}

export function useToolDelete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (toolId: string) => toolApi.remove(toolId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tools', 'list'] }),
  });
}

export function useToolResolve() {
  return useMutation<ToolRuntimeSnapshot, Error, { toolId: string; runtimeId?: string; sessionId?: string; version?: string; label?: string }>({
    mutationFn: ({ toolId, ...body }) => toolApi.resolve(toolId, body),
  });
}

export function useToolFailures(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['tools', 'failures', params],
    queryFn: async () => asItems<ToolFailureRecord>(await toolApi.failures(params)),
    staleTime: 10_000,
  });
}
