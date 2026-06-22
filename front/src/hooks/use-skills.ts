'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { skillApi, socialApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { Skill, SkillDraft } from '@/lib/api/types';

export function useSkills(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['skills', 'list', params],
    queryFn: async () => {
      const page = await skillApi.list(params);
      return asItems<Skill>(page);
    },
    staleTime: 15_000,
  });
}

export function useSkillDetail(skillName: string | null) {
  return useQuery({
    queryKey: ['skills', 'detail', skillName],
    queryFn: async () => {
      const d = await skillApi.detail(skillName!);
      const versions = [...(d.versions || [])].sort((a, b) =>
        String(b.version).localeCompare(String(a.version))
      );
      d.versions = versions;
      return d;
    },
    enabled: Boolean(skillName),
    staleTime: 10_000,
  });
}

export function useSkillFiles(skillName: string | null, version: string | null) {
  return useQuery({
    queryKey: ['skills', 'files', skillName, version],
    queryFn: () => skillApi.files(skillName!, version!),
    enabled: Boolean(skillName && version),
    staleTime: 30_000,
  });
}

export function useSkillFileContent(skillName: string | null, version: string | null, path: string | null) {
  return useQuery({
    queryKey: ['skills', 'file', skillName, version, path],
    queryFn: () => skillApi.file(skillName!, version!, path!),
    enabled: Boolean(skillName && version && path),
    staleTime: 60_000,
  });
}

export function useSkillCompare(skillName: string | null, base: string, target: string) {
  return useQuery({
    queryKey: ['skills', 'compare', skillName, base, target],
    queryFn: () => skillApi.compare(skillName!, base, target),
    enabled: Boolean(skillName && base && target && base !== target),
    staleTime: 60_000,
  });
}

export function useSkillSocial(skillName: string | null) {
  return useQuery({
    queryKey: ['skills', 'social', skillName],
    queryFn: () => socialApi.stats(skillName!),
    enabled: Boolean(skillName),
    staleTime: 30_000,
  });
}

export function useSkillUpload() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, overwrite, targetVersion, commitMsg }: {
      file: File; overwrite?: boolean; targetVersion?: string; commitMsg?: string;
    }) => skillApi.upload(file, overwrite, targetVersion, commitMsg),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
    },
  });
}

export function useSkillBatchUpload() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ files, overwrite }: { files: File[]; overwrite?: boolean }) =>
      skillApi.batchUpload(files, overwrite),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
    },
  });
}

export function useSkillDelete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (skillName: string) => skillApi.remove(skillName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail'] });
    },
  });
}

export function useSkillPublish() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, version }: { skillName: string; version: string }) =>
      skillApi.publish(skillName, version),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
    },
  });
}

export function useSkillSubmit() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, version }: { skillName: string; version: string }) =>
      skillApi.submit(skillName, version),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSkillOnline() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, version }: { skillName: string; version?: string }) =>
      skillApi.online(skillName, version),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
    },
  });
}

export function useSkillOffline() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, version }: { skillName: string; version?: string }) =>
      skillApi.offline(skillName, version),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
    },
  });
}

export function useSkillLabels() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, labels }: { skillName: string; labels: Record<string, string> }) =>
      skillApi.labels(skillName, labels),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSkillDraft() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: SkillDraft) => skillApi.draft(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'list'] });
    },
  });
}

export function useSkillForcePublish() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, version }: { skillName: string; version: string }) =>
      skillApi.forcePublish(skillName, version),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSkillRedraft() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, version }: { skillName: string; version: string }) =>
      skillApi.redraft(skillName, version),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSkillBizTags() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, tags }: { skillName: string; tags: string[] }) =>
      skillApi.bizTags(skillName, tags),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSkillMetadata() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, metadata }: { skillName: string; metadata: Record<string, unknown> }) =>
      skillApi.metadata(skillName, metadata),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSkillScope() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, scope }: { skillName: string; scope: string }) =>
      skillApi.scope(skillName, scope),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'detail', vars.skillName] });
    },
  });
}

export function useSocialStar() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, starred }: { skillName: string; starred: boolean }) =>
      socialApi.star(skillName, starred),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'social', vars.skillName] });
    },
  });
}

export function useSocialRating() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, rating, comment }: { skillName: string; rating: number; comment?: string }) =>
      socialApi.rating(skillName, rating, comment),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'social', vars.skillName] });
    },
  });
}

export function useSocialSubscribe() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ skillName, subscribed }: { skillName: string; subscribed: boolean }) =>
      socialApi.subscribe(skillName, subscribed),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['skills', 'social', vars.skillName] });
    },
  });
}
