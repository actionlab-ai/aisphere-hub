'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { namespaceApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { NamespaceInfo, NamespaceMember } from '@/lib/api/types';

export function useNamespaces() {
  return useQuery({
    queryKey: ['namespaces', 'list'],
    queryFn: async () => {
      const data = await namespaceApi.list();
      return asItems<NamespaceInfo>(data);
    },
    staleTime: 30_000,
  });
}

export function useNamespaceMembers(namespaceId: string) {
  return useQuery({
    queryKey: ['namespaces', 'members', namespaceId],
    queryFn: async () => {
      const data = await namespaceApi.members(namespaceId);
      return asItems<NamespaceMember>(data);
    },
    enabled: Boolean(namespaceId),
    staleTime: 30_000,
  });
}

export function useNamespaceSave() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) => namespaceApi.save(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['namespaces'] });
    },
  });
}

export function useNamespaceSaveMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ namespaceId, data }: { namespaceId: string; data: Record<string, unknown> }) =>
      namespaceApi.saveMember(namespaceId, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['namespaces', 'members', vars.namespaceId] });
    },
  });
}

export function useNamespaceDeleteMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ namespaceId, subjectId }: { namespaceId: string; subjectId: string }) =>
      namespaceApi.deleteMember(namespaceId, subjectId),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['namespaces', 'members', vars.namespaceId] });
    },
  });
}
