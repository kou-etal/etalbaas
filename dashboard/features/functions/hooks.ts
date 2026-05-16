"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { functionClient } from "@/lib/api/clients";

export interface FunctionItem {
  id: string;
  name: string;
  runtime: string;
  kind: string;
  mode: string;
  status: string;
  entrypoint: string;
  image: string;
  lastBuildAt: string;
  updatedAt: string;
  timeout: number;
  minReplicas: number;
  maxReplicas: number;
  gpu: string;
  resources: {
    cpu: string;
    memory: string;
  };
  envVars: Record<string, string>;
}

export function useFunctions(projectId: string) {
  return useQuery<FunctionItem[]>({
    queryKey: queryKeys.functions.list(projectId),
    queryFn: async () => {
      const response = await functionClient.listFunctions({ projectId });
      return response.functions || [];
    },
    enabled: !!projectId,
  });
}

export function useFunction(functionId: string) {
  return useQuery<FunctionItem | undefined>({
    queryKey: queryKeys.functions.detail(functionId),
    queryFn: async () => {
      const response = await functionClient.getFunction({ functionId });
      return response.function;
    },
    enabled: !!functionId,
  });
}

export function useCreateFunction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      projectId: string;
      name: string;
      runtime: string;
      kind: string;
      mode: string;
    }) => {
      const response = await functionClient.createFunction(data);
      return response.function;
    },
    onSuccess: (_data: unknown, variables: {
      projectId: string;
      name: string;
      runtime: string;
      kind: string;
      mode: string;
    }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.functions.list(variables.projectId),
      });
    },
  });
}

export function useDeleteFunction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { functionId: string; projectId: string }) => {
      await functionClient.deleteFunction({ functionId: params.functionId });
      return params;
    },
    onSuccess: (_data: unknown, variables: { functionId: string; projectId: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.functions.list(variables.projectId),
      });
    },
  });
}
