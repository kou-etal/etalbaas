"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { storageClient, storageRestClient } from "@/lib/api/clients";
import type { StorageObject } from "@/lib/api/clients";

export type { StorageObject };

export interface Bucket {
  id: string;
  name: string;
  accessLevel: string;
  fileSizeLimit: number;
  allowedMimeTypes: string[];
  objectCount?: number;
  createdAt: string;
  updatedAt: string;
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
    mutationFn: async (data: { projectId: string; name: string; accessLevel: string }) => {
      const response = await storageClient.createBucket(data);
      return response.bucket;
    },
    onSuccess: (_data: unknown, variables: { projectId: string; name: string; accessLevel: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.storage.buckets(variables.projectId),
      });
    },
  });
}

export function useListObjects(projectId: string, bucketName: string) {
  return useQuery<StorageObject[]>({
    queryKey: queryKeys.storage.objects(projectId, bucketName),
    queryFn: () => storageRestClient.listObjects(projectId, bucketName),
    enabled: !!projectId && !!bucketName,
  });
}

export function useUploadObject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { projectId: string; bucket: string; path: string; file: File }) => {
      return storageRestClient.uploadObject(data.projectId, data.bucket, data.path, data.file);
    },
    onSuccess: (_data: unknown, variables: { projectId: string; bucket: string; path: string; file: File }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.storage.objects(variables.projectId, variables.bucket),
      });
    },
  });
}

export function useDeleteObject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { projectId: string; bucket: string; path: string }) => {
      return storageRestClient.deleteObject(data.projectId, data.bucket, data.path);
    },
    onSuccess: (_data: unknown, variables: { projectId: string; bucket: string; path: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.storage.objects(variables.projectId, variables.bucket),
      });
    },
  });
}
