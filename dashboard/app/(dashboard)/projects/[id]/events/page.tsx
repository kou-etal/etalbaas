"use client";

import { useParams } from "next/navigation";
import { Radio, ChevronDown, AlertCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useEvents } from "@/features/events/hooks";
import { formatDate } from "@/lib/utils/format";
import React, { useState } from "react";

export default function EventsPage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: events, isLoading, isError } = useEvents(projectId);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  if (isError) {
    return (
      <div className="p-6">
        <EmptyState
          icon={AlertCircle}
          title="Failed to load events"
          description="An error occurred while fetching event history. Please try again."
        />
      </div>
    );
  }

  if (!events || events.length === 0) {
    return (
      <div className="p-6">
        <EmptyState
          icon={Radio}
          title="No events"
          description="Events will appear here as they are published to your project's event bus."
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="rounded-lg border">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground w-8"></th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Subject</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Type</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Time</th>
            </tr>
          </thead>
          <tbody>
            {events.map((event) => (
              <React.Fragment key={event.id}>
                <tr
                  className="border-b last:border-0 hover:bg-muted/30 transition-colors cursor-pointer"
                  onClick={() => setExpandedId(expandedId === event.id ? null : event.id)}
                >
                  <td className="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      aria-expanded={expandedId === event.id}
                      aria-label="Toggle details"
                    >
                      <ChevronDown
                        className={`h-3 w-3 transition-transform ${expandedId === event.id ? "rotate-180" : ""}`}
                      />
                    </Button>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-sm font-medium">{event.subject || "-"}</span>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant="outline" className="text-xs">{event.type || "custom"}</Badge>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant="success" className="text-xs">delivered</Badge>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs text-muted-foreground">
                      {event.createdAt ? formatDate(event.createdAt) : "-"}
                    </span>
                  </td>
                </tr>
                {expandedId === event.id && (
                  <tr className="border-b">
                    <td colSpan={5} className="px-8 py-4 bg-muted/20">
                      <pre className="text-xs font-mono whitespace-pre-wrap text-muted-foreground max-h-64 overflow-auto">
                        {JSON.stringify(event, null, 2)}
                      </pre>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
