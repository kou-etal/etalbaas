"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { storageClient } from "@/lib/api/clients";

export interface Bucket {
  id: string;
  name: string;
  isPublic: boolean;
  objectCount: number;
  createdAt: string;
}

export function useBuckets(projectId: string) {
  return useQuery<Bucket[]>({
    queryKey: queryKeys.storage.buckets(projectId),
    queryFn: async () => {
      const response = await storageClient.listBuckets({ projectId });
      return response.buckets || [];
    },
    enabled: !!projectId,
  });
}

export function useCreateBucket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { projectId: string; name: string; isPublic: boolean }) => {
      const response = await storageClient.createBucket(data);
      return response.bucket;
    },
    onSuccess: (_data: unknown, variables: { projectId: string; name: string; isPublic: boolean }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.storage.buckets(variables.projectId),
      });
    },
  });
}
