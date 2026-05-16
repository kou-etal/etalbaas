"use client";

import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "@/lib/api/queries";
import { eventClient } from "@/lib/api/clients";

export interface EventItem {
  id: string;
  subject: string;
  type: string;
  createdAt: string;
  data?: Record<string, unknown>;
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
