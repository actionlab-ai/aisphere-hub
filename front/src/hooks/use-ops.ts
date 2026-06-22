'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { auditApi, tokenApi, metricsApi, notificationApi } from '@/lib/api';
import { asItems } from '@/lib/api/client';
import type { AuditLog, TokenInfo, MetricsSnapshot, Notification } from '@/lib/api/types';

export function useAuditLogs(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['ops', 'audit', params],
    queryFn: async () => {
      const page = await auditApi.list(params);
      return asItems<AuditLog>(page);
    },
    staleTime: 15_000,
  });
}

export function useMetrics() {
  return useQuery({
    queryKey: ['ops', 'metrics'],
    queryFn: () => metricsApi.get(),
    staleTime: 10_000,
  });
}

export function useTokens(subjectId = '') {
  return useQuery({
    queryKey: ['ops', 'tokens', subjectId],
    queryFn: async () => {
      const data = await tokenApi.list(subjectId);
      return asItems<TokenInfo>(data);
    },
    staleTime: 15_000,
  });
}

export function useTokenCreate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) => tokenApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ops', 'tokens'] });
    },
  });
}

export function useTokenDelete() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (keyId: string) => tokenApi.remove(keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ops', 'tokens'] });
    },
  });
}

export function useNotifications(params: Record<string, unknown> = {}) {
  return useQuery({
    queryKey: ['ops', 'notifications', params],
    queryFn: async () => {
      const data = await notificationApi.list(params);
      return asItems<Notification>(data);
    },
    staleTime: 10_000,
  });
}

export function useNotificationMarkRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => notificationApi.markRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ops', 'notifications'] });
    },
  });
}
