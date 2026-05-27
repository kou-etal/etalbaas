"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { secretClient } from "@/lib/api/clients";

export interface SecretItem {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export function useSecrets(projectId: string) {
  return useQuery<SecretItem[]>({
    queryKey: queryKeys.secrets.list(projectId),
    queryFn: async () => {
      const response = await secretClient.listSecrets({ projectId });
      return response.secrets || [];
    },
    enabled: !!projectId,
  });
}

export function useCreateSecret() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { projectId: string; name: string; value: string; description?: string }) => {
      await secretClient.createSecret(data);
    },
    onSuccess: (_data: unknown, variables: { projectId: string; name: string; value: string; description?: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.secrets.list(variables.projectId),
      });
    },
  });
}

export function useUpdateSecretValue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { projectId: string; secretId: string; value: string }) => {
      await secretClient.updateSecretValue(params);
      return params;
    },
    onSuccess: (_data: unknown, variables: { projectId: string; secretId: string; value: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.secrets.list(variables.projectId),
      });
    },
  });
}

export function useDeleteSecret() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { projectId: string; secretId: string }) => {
      await secretClient.deleteSecret(params);
      return params;
    },
    onSuccess: (_data: unknown, variables: { projectId: string; secretId: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.secrets.list(variables.projectId),
      });
    },
  });
}
