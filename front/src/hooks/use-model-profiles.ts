import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { modelProfileApi } from '@/lib/api';
import type { ModelProfile } from '@/lib/api/types';

export function useModelProfiles() {
  return useQuery({ queryKey: ['model-profiles'], queryFn: modelProfileApi.list });
}
export function useSaveModelProfile() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (p: ModelProfile) => modelProfileApi.save(p), onSuccess: () => qc.invalidateQueries({queryKey:['model-profiles']}) });
}
export function useDeleteModelProfile() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => modelProfileApi.remove(id), onSuccess: () => qc.invalidateQueries({queryKey:['model-profiles']}) });
}
