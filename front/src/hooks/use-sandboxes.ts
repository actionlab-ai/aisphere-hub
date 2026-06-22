'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { sandboxApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { SandboxEnsureRequest, SandboxStatus, SandboxToolCallRequest } from '@/lib/api/types';

export function useSandboxes(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['sandboxes', 'list', params],
    queryFn: async () => asItems<SandboxStatus>(await sandboxApi.list(params)),
    staleTime: 5_000,
  });
}

export function useSandboxDetail(sandboxId: string | null) {
  return useQuery<SandboxStatus>({
    queryKey: ['sandboxes', 'detail', sandboxId],
    queryFn: () => sandboxApi.get(sandboxId!),
    enabled: Boolean(sandboxId),
    staleTime: 3_000,
  });
}

export function useSandboxEnsure() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: SandboxEnsureRequest) => sandboxApi.ensure(body),
    onSuccess: (item) => {
      queryClient.invalidateQueries({ queryKey: ['sandboxes', 'list'] });
      queryClient.invalidateQueries({ queryKey: ['sandboxes', 'detail', item.sandboxId] });
    },
  });
}

export function useSandboxRestart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (sandboxId: string) => sandboxApi.restart(sandboxId),
    onSuccess: (item) => {
      queryClient.invalidateQueries({ queryKey: ['sandboxes', 'list'] });
      queryClient.invalidateQueries({ queryKey: ['sandboxes', 'detail', item.sandboxId] });
    },
  });
}

export function useSandboxDelete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ sandboxId, deleteWorkspace = false }: { sandboxId: string; deleteWorkspace?: boolean }) => sandboxApi.remove(sandboxId, deleteWorkspace),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['sandboxes', 'list'] }),
  });
}


export function useSandboxTools(sandboxId: string | null) {
  return useQuery({
    queryKey: ['sandboxes', 'tools', sandboxId],
    queryFn: () => sandboxApi.tools(sandboxId!),
    enabled: Boolean(sandboxId),
    staleTime: 5_000,
  });
}

export function useSandboxToolCall() {
  return useMutation({
    mutationFn: ({ sandboxId, body }: { sandboxId: string; body: SandboxToolCallRequest }) => sandboxApi.callTool(sandboxId, body),
  });
}
