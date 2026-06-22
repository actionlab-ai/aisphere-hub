'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sharesApi } from '@/lib/api';
import { deriveAccessMode } from '@/lib/api/types';
import type {
  AihubResourceType,
  CreateShareRequest,
  ResourceGrant,
  ShareListResponse,
  AccessMode,
} from '@/lib/api/types';

// Query key helper — keeps the cache consistent across consumers.
function sharesKey(resourceType: AihubResourceType, resourceId: string) {
  return ['shares', resourceType, resourceId] as const;
}

// ─── List shares ────────────────────────────────────────────────────
export function useResourceShares(
  resourceType: AihubResourceType,
  resourceId: string | null,
) {
  return useQuery({
    queryKey: sharesKey(resourceType, resourceId || ''),
    queryFn: async () => {
      const raw = await sharesApi.list(resourceType, resourceId!);
      // Normalize the response: the backend may return items at the
      // top level, or wrapped in a Page-like shape. We also compute
      // accessMode + canManage locally when the backend omits them.
      const items = (raw?.items ?? (Array.isArray(raw) ? (raw as ResourceGrant[]) : [])) as ResourceGrant[];
      const accessMode: AccessMode = raw?.accessMode ?? deriveAccessMode(items);
      const canManage: boolean = raw?.canManage ?? true; // optimistic; backend should override
      const result: ShareListResponse & { accessMode: AccessMode; canManage: boolean } = {
        items,
        accessMode,
        canManage,
        total: raw?.total,
        limit: raw?.limit,
        offset: raw?.offset,
      };
      return result;
    },
    enabled: Boolean(resourceId),
    staleTime: 15_000,
  });
}

// ─── Create share ───────────────────────────────────────────────────
export function useCreateShare() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      resourceType,
      resourceId,
      body,
    }: {
      resourceType: AihubResourceType;
      resourceId: string;
      body: CreateShareRequest;
    }) => sharesApi.create(resourceType, resourceId, body),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: sharesKey(vars.resourceType, vars.resourceId) });
    },
  });
}

// ─── Delete share ───────────────────────────────────────────────────
export function useDeleteShare() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      resourceType,
      resourceId,
      grantId,
    }: {
      resourceType: AihubResourceType;
      resourceId: string;
      grantId: string;
    }) => sharesApi.remove(resourceType, resourceId, grantId),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: sharesKey(vars.resourceType, vars.resourceId) });
    },
  });
}

// ─── Bulk delete all shares ("set to private") ──────────────────────
// The backend may eventually provide a single DELETE-all endpoint;
// for now the frontend loops over the grant list client-side.
export function useSetPrivate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      resourceType,
      resourceId,
      grantIds,
    }: {
      resourceType: AihubResourceType;
      resourceId: string;
      grantIds: string[];
    }) => {
      // Sequential delete — preserves order so we can stop on first error.
      const failures: string[] = [];
      for (const id of grantIds) {
        try {
          await sharesApi.remove(resourceType, resourceId, id);
        } catch (e) {
          failures.push(id);
        }
      }
      return { deleted: grantIds.length - failures.length, failures };
    },
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: sharesKey(vars.resourceType, vars.resourceId) });
    },
  });
}
