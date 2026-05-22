"use client";

import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { eventClient } from "@/lib/api/clients";

export interface EventItem {
  id: string;
  projectId: string;
  functionId: string;
  invocationId: string;
  trigger?: {
    databaseChange?: { table: string; event: string };
    objectStorage?: { bucket: string; objectKey: string; event: string };
  };
  status: string;
  attemptCount: number;
  lastError: string;
  traceId: string;
  createdAt: string;
}

export function useEvents(projectId: string) {
  return useQuery<EventItem[]>({
    queryKey: queryKeys.events.list(projectId),
    queryFn: async () => {
      const response = await eventClient.listEventHistory({ projectId });
      return response.events || [];
    },
    enabled: !!projectId,
  });
}
