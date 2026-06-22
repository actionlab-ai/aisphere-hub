'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { iamApi } from '@/lib/api';
import type { LocalUser } from '@/lib/api/types';

export function useIamUsers() {
  return useQuery({
    queryKey: ['iam', 'users'],
    queryFn: () => iamApi.listUsers(),
    staleTime: 30_000,
  });
}

export function useIamSaveUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (u: LocalUser & { password?: string }) => iamApi.saveUser(u),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['iam', 'users'] });
    },
  });
}

export function useIamDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (username: string) => iamApi.deleteUser(username),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['iam', 'users'] });
    },
  });
}
