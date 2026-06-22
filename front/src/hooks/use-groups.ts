'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { groupApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { SkillGroup, GroupUpdate } from '@/lib/api/types';

export function useGroups(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['groups', 'list', params],
    queryFn: async () => {
      const page = await groupApi.list(params);
      return asItems<SkillGroup>(page);
    },
    staleTime: 15_000,
  });
}

export function useGroupDetail(groupName: string | null) {
  return useQuery({
    queryKey: ['groups', 'detail', groupName],
    queryFn: () => groupApi.detail(groupName!),
    enabled: Boolean(groupName),
    staleTime: 10_000,
  });
}

export function useGroupSkills(groupName: string | null) {
  return useQuery({
    queryKey: ['groups', 'skills', groupName],
    queryFn: () => groupApi.groupSkills(groupName!),
    enabled: Boolean(groupName),
    staleTime: 10_000,
  });
}

export function useGroupSave() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (group: SkillGroup) => groupApi.save(group),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
    },
  });
}

export function useGroupUpdate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupName, data }: { groupName: string; data: GroupUpdate }) =>
      groupApi.update(groupName, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['groups', 'detail', vars.groupName] });
      queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
    },
  });
}

export function useGroupDelete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (groupName: string) => groupApi.remove(groupName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
    },
  });
}

export function useGroupBind() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupName, member }: { groupName: string; member: Record<string, unknown> }) =>
      groupApi.bind(groupName, member),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['groups', 'detail', vars.groupName] });
      queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
    },
  });
}

export function useGroupUnbind() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ groupName, skillName }: { groupName: string; skillName: string }) =>
      groupApi.unbind(groupName, skillName),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['groups', 'detail', vars.groupName] });
      queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
    },
  });
}
