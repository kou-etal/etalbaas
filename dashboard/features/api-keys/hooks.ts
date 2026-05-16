"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { projectClient } from "@/lib/api/clients";

export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  createdAt: string;
}

export function useApiKeys(projectId: string) {
  return useQuery<ApiKey[]>({
    queryKey: [...queryKeys.projects.detail(projectId), "api-keys"],
    queryFn: async () => {
      const response = await projectClient.listApiKeys({ projectId });
      return response.apiKeys || [];
    },
    enabled: !!projectId,
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { projectId: string; name: string }) => {
      const response = await projectClient.createApiKey(data);
      return response;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.projects.detail(variables.projectId), "api-keys"],
      });
    },
  });
}

export function useRevokeApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { projectId: string; keyId: string }) => {
      await projectClient.revokeApiKey(params);
      return params;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.projects.detail(variables.projectId), "api-keys"],
      });
    },
  });
}
