'use client';

import { useMutation, useQuery } from '@tanstack/react-query';
import { accessApi } from '@/lib/api';

export function useAccessOverview() {
  return useQuery({ queryKey: ['access', 'overview'], queryFn: () => accessApi.overview(), staleTime: 30_000 });
}

export function useAccessResources() {
  return useQuery({ queryKey: ['access', 'resources'], queryFn: () => accessApi.resources(), staleTime: 60_000 });
}

export function useAccessLinks() {
  return useQuery({ queryKey: ['access', 'links'], queryFn: () => accessApi.links(), staleTime: 60_000 });
}

export function useEvaluateAccess() {
  return useMutation({ mutationFn: ({ subject, object, action }: { subject: string; object: string; action: string }) => accessApi.evaluate(subject, object, action) });
}
