'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authApi } from '@/lib/api';
import { getToken, clearTokens, setTokens } from '@/lib/api/client';

export function useAuthStatus() {
  return useQuery({
    queryKey: ['auth', 'setupStatus'],
    queryFn: () => authApi.setupStatus(),
    staleTime: 30_000,
  });
}

export function useMe() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => authApi.me(),
    enabled: Boolean(getToken()),
    staleTime: 60_000,
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      authApi.login(username, password),
    onSuccess: (data) => {
      setTokens(data.accessToken, data.refreshToken);
      queryClient.invalidateQueries({ queryKey: ['auth'] });
    },
  });
}

export function useSetup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) => authApi.setup(data),
    onSuccess: (data) => {
      setTokens(data.tokens.accessToken, data.tokens.refreshToken);
      queryClient.invalidateQueries({ queryKey: ['auth'] });
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return () => {
    clearTokens();
    queryClient.clear();
  };
}
