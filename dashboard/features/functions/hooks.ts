"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { functionClient } from "@/lib/api/clients";

export interface EnvVar {
  name: string;
  value?: string;
  secretName?: string;
}

export interface FunctionItem {
  id: string;
  projectId: string;
  name: string;
  displayName: string;
  kind: string;
  mode: string;
  inlineSource?: { code: string; filename: string };
  gitSource?: { repoUrl: string; branch: string; subpath: string };
  presetRuntime?: { preset: string; requirements: string[] };
  customRuntime?: { dockerfile: string };
  timeoutSec: number;
  gpuConfig?: { type: string; provider: string; product: string };
  triggers?: Array<{
    databaseChange?: { table: string; events: string[]; filter?: string; includeColumns?: string[] };
    objectStorage?: { bucket: string; events: string[]; prefix: string };
  }>;
  envVars: EnvVar[];
  status: string;
  buildImageRef: string;
  buildImageDigest: string;
  buildDurationSec: number;
  lastBuiltAt: string;
  createdAt: string;
  updatedAt: string;
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

export function useFunction(projectId: string, functionId: string) {
  return useQuery<FunctionItem | undefined>({
    queryKey: queryKeys.functions.detail(functionId),
    queryFn: async () => {
      const response = await functionClient.getFunction({ projectId, functionId });
      return response.function;
    },
    enabled: !!projectId && !!functionId,
  });
}

export function useCreateFunction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      projectId: string;
      name: string;
      kind: string;
      mode: string;
      inlineSource?: { code: string; filename: string };
      gitSource?: { repoUrl: string; branch: string; subpath: string };
    }) => {
      const response = await functionClient.createFunction(data);
      return response.function;
    },
    onSuccess: (_data: unknown, variables: {
      projectId: string;
      name: string;
      kind: string;
      mode: string;
    }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.functions.list(variables.projectId),
      });
    },
  });
}

export function useUpdateFunction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { projectId: string; functionId: string; [key: string]: unknown }) => {
      const response = await functionClient.updateFunction(data);
      return response.function;
    },
    onSuccess: (_data: unknown, variables: { projectId: string; functionId: string; [key: string]: unknown }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.functions.detail(variables.functionId as string),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.functions.list(variables.projectId as string),
      });
    },
  });
}

export function useInvocations(projectId: string, functionId: string) {
  return useQuery({
    queryKey: queryKeys.invocations.list(projectId, functionId),
    queryFn: async () => {
      const response = await functionClient.listInvocations({ projectId, functionId });
      return response.invocations || [];
    },
    enabled: !!projectId && !!functionId,
  });
}

export function useDeleteFunction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { projectId: string; functionId: string }) => {
      await functionClient.deleteFunction({ projectId: params.projectId, functionId: params.functionId });
      return params;
    },
    onSuccess: (_data: unknown, variables: { projectId: string; functionId: string }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.functions.list(variables.projectId),
      });
    },
  });
}
