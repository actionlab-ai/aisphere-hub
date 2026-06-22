import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { sandboxProfileApi } from '@/lib/api';
import type { SandboxProfile } from '@/lib/api/types';

export function useSandboxProfiles() {
  return useQuery({ queryKey: ['sandbox-profiles'], queryFn: sandboxProfileApi.list });
}
export function useSaveSandboxProfile() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (p: SandboxProfile) => sandboxProfileApi.save(p), onSuccess: () => qc.invalidateQueries({queryKey:['sandbox-profiles']}) });
}
export function useDeleteSandboxProfile() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => sandboxProfileApi.remove(id), onSuccess: () => qc.invalidateQueries({queryKey:['sandbox-profiles']}) });
}
