"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { projectClient } from "@/lib/api/clients";

export interface Project {
  id: string;
  displayName: string;
  description: string;
  status: string;
  postgresEnabled: boolean;
  postgresExtensions: string[];
  redisEnabled: boolean;
  postgrestEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export function useProjects() {
  return useQuery<Project[]>({
    queryKey: queryKeys.projects.list(),
    queryFn: async () => {
      const response = await projectClient.listProjects({});
      return response.projects || [];
    },
  });
}

export function useProject(id: string) {
  return useQuery<Project | undefined>({
    queryKey: queryKeys.projects.detail(id),
    queryFn: async () => {
      const response = await projectClient.getProject({ projectId: id });
      return response.project;
    },
    enabled: !!id,
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      displayName: string;
      description: string;
      postgresEnabled?: boolean;
      postgresExtensions?: string[];
      redisEnabled?: boolean;
      postgrestEnabled?: boolean;
    }) => {
      const response = await projectClient.createProject(data);
      return response.project;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

export function useDeleteProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (projectId: string) => {
      await projectClient.deleteProject({ projectId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

export function usePauseProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (projectId: string) => {
      const response = await projectClient.pauseProject({ projectId });
      return response.project;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

export function useResumeProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (projectId: string) => {
      const response = await projectClient.resumeProject({ projectId });
      return response.project;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}
